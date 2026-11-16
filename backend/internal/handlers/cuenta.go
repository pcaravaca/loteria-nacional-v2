package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

// MaxUploadBytes es el tamano maximo aceptado para la foto del documento.
const MaxUploadBytes = 5 << 20 // 5 MB

// aleatorioHex devuelve una cadena aleatoria para nombrar los archivos.
func aleatorioHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// TiposDocumento es el catalogo de identificacion aceptado (KYC basico).
var TiposDocumento = map[string]string{
	models.DocCedulaFisica:    "Cédula física",
	models.DocCedulaJuridica:  "Cédula jurídica",
	models.DocDIMEX:           "DIMEX",
	models.DocPasaporte:       "Pasaporte",
	models.DocConsumidorFinal: "Consumidor final (sin documento)",
}

// AutoexclusionActiva indica si la autoexclusion sigue vigente.
func AutoexclusionActiva(hasta *time.Time) bool {
	return hasta != nil && time.Now().Before(*hasta)
}

// GuardJugada rechaza la jugada cuando hay autoexclusion vigente o cuando el
// operador exige identificacion verificada y el usuario no la tiene.
func GuardJugada(c *gin.Context, cfg *config.Config, u models.User) bool {
	if AutoexclusionActiva(u.AutoexcluidoHasta) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "cuenta autoexcluida; no puedes jugar hasta la fecha indicada",
			"hasta": u.AutoexcluidoHasta,
		})
		return false
	}
	if cfg.KYCRequired && !u.DocumentoVerificado {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "debes completar y verificar tu identificacion antes de jugar",
		})
		return false
	}
	return true
}

// CuentaHandler agrupa el juego responsable y la identificacion del jugador.
type CuentaHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewCuentaHandler(db *gorm.DB, cfg *config.Config) *CuentaHandler {
	return &CuentaHandler{db: db, cfg: cfg}
}

// AutoexclusionEstado devuelve si la cuenta esta autoexcluida y hasta cuando.
func (h *CuentaHandler) AutoexclusionEstado(c *gin.Context) {
	user, ok := h.usuario(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"activa": AutoexclusionActiva(user.AutoexcluidoHasta),
		"hasta":  user.AutoexcluidoHasta,
	})
}

// Autoexclusionactivar registra la autoexclusion voluntaria, que es irreversible
// durante el periodo pedido (minimo 6 meses).
func (h *CuentaHandler) AutoexclusionActivar(c *gin.Context) {
	user, ok := h.usuario(c)
	if !ok {
		return
	}
	var req struct {
		Meses int `json:"meses"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Meses <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indica los meses de autoexclusion"})
		return
	}
	if req.Meses < models.AutoexclusionMinimaMeses {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "la autoexclusion minima es de 6 meses y no tiene reversa",
		})
		return
	}
	if AutoexclusionActiva(user.AutoexcluidoHasta) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la cuenta ya esta autoexcluida", "hasta": user.AutoexcluidoHasta})
		return
	}
	hasta := time.Now().AddDate(0, req.Meses, 0)
	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).
		Update("autoexcluido_hasta", hasta).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar la autoexclusion"})
		return
	}
	RegistrarAuditoria(h.db, c, user.ID, "autoexclusion.activar", "usuario",
		strconv.FormatUint(uint64(user.ID), 10),
		fmt.Sprintf("meses=%d hasta=%s", req.Meses, hasta.Format(time.RFC3339)))
	c.JSON(http.StatusOK, gin.H{"activa": true, "hasta": hasta})
}

// IdentificacionGuarda tipo y numero de documento. Cambiar el documento quita
// la verificacion anterior, porque habria que revisarlo de nuevo.
type identificacionRequest struct {
	Tipo            string `json:"tipo"`
	Numero          string `json:"numero"`
	FechaNacimiento string `json:"fecha_nacimiento"`
}

// IdentificacionGuardar guarda la identificacion del jugador.
func (h *CuentaHandler) IdentificacionGuardar(c *gin.Context) {
	user, ok := h.usuario(c)
	if !ok {
		return
	}
	var req identificacionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}
	if _, valido := TiposDocumento[req.Tipo]; !valido {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tipo de documento no valido", "tipos": TiposDocumento})
		return
	}
	if req.Numero == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indica el numero de documento"})
		return
	}
	updates := map[string]any{
		"tipo_documento":       req.Tipo,
		"numero_documento":     req.Numero,
		"documento_verificado": false,
	}
	if strings.TrimSpace(req.FechaNacimiento) != "" {
		fecha, err := time.Parse("2006-01-02", req.FechaNacimiento)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "la fecha de nacimiento debe tener formato AAAA-MM-DD"})
			return
		}
		if EdadEn(fecha, time.Now()) < EdadMinima {
			c.JSON(http.StatusBadRequest, gin.H{"error": "debes ser mayor de edad para jugar"})
			return
		}
		updates["fecha_nacimiento"] = fecha
	}

	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la identificacion"})
		return
	}
	RegistrarAuditoria(h.db, c, user.ID, "kyc.identificacion", "usuario",
		strconv.FormatUint(uint64(user.ID), 10),
		fmt.Sprintf("%s %s", req.Tipo, req.Numero))
	c.JSON(http.StatusOK, gin.H{"tipo_documento": req.Tipo, "numero_documento": req.Numero, "documento_verificado": false})
}

// usuario devuelve el usuario del contexto JWT.
func (h *CuentaHandler) usuario(c *gin.Context) (models.User, bool) {
	userID := c.GetUint("userID")
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sesion invalida"})
		return user, false
	}
	return user, true
}

// SubirFotoDocumento guarda la imagen del documento (JPG o PNG, hasta 5 MB).
func (h *CuentaHandler) SubirFotoDocumento(c *gin.Context) {
	user, ok := h.usuario(c)
	if !ok {
		return
	}
	fileHeader, err := c.FormFile("archivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no se recibio el archivo"})
		return
	}
	if fileHeader.Size > MaxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "el archivo supera los 5 MB"})
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no se pudo leer el archivo"})
		return
	}
	defer src.Close()

	cabecera := make([]byte, 512)
	leidos, _ := io.ReadFull(src, cabecera)
	tipo := http.DetectContentType(cabecera[:leidos])
	ext := ""
	switch tipo {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "solo se aceptan imagenes JPG o PNG"})
		return
	}

	// La deteccion de tipo consume parte del archivo: hay que rebobinar para
	// guardar el contenido completo.
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo procesar el archivo"})
		return
	}

	dir := filepath.Join(h.cfg.UploadDir, "documentos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo preparar el directorio"})
		return
	}
	ruta := filepath.Join(dir, fmt.Sprintf("%d_%s%s", user.ID, aleatorioHex(8), ext))

	dst, err := os.Create(ruta)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el archivo"})
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(ruta)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el archivo"})
		return
	}

	if user.FotoDocumento != "" {
		_ = os.Remove(user.FotoDocumento)
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).
		Update("foto_documento", ruta).Error; err != nil {
		_ = os.Remove(ruta)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar el documento"})
		return
	}

	RegistrarAuditoria(h.db, c, user.ID, "kyc.documento", "usuario",
		strconv.FormatUint(uint64(user.ID), 10), "foto del documento subida")
	c.JSON(http.StatusOK, gin.H{"documento": true})
}
