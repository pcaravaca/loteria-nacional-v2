package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

// pngFalso son los primeros bytes de la firma PNG: DetectContentType solo
// necesita la cabecera para identificar la imagen.
var pngFalso = []byte("\x89PNG\r\n\x1a\n\xc0\x00\x00\x00")

func peticionFoto(t *testing.T, campo string, nombre string, contenido []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(campo, nombre)
	if err != nil {
		t.Fatalf("crear form file: %v", err)
	}
	if _, err := fw.Write(contenido); err != nil {
		t.Fatalf("escribir contenido: %v", err)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/cuenta/identificacion/foto", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestSubirFotoDocumento(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "jugador")
	cfg := &config.Config{UploadDir: t.TempDir()}
	h := NewCuentaHandler(db, cfg)

	c, w := contextoPrueba(http.MethodPost, "/", nil, u.ID)
	c.Request = peticionFoto(t, "archivo", "doc.png", pngFalso)
	h.SubirFotoDocumento(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d (%s)", w.Code, w.Body.String())
	}
	var actualizado models.User
	_ = db.First(&actualizado, u.ID)
	if actualizado.FotoDocumento == "" {
		t.Fatal("la foto no quedo registrada en el usuario")
	}
	if _, err := os.Stat(actualizado.FotoDocumento); err != nil {
		t.Errorf("el archivo no existe en disco: %v", err)
	}
	if filepath.Ext(actualizado.FotoDocumento) != ".png" {
		t.Errorf("extension inesperada: %s", actualizado.FotoDocumento)
	}

	// Un archivo que no es imagen se rechaza y no borra la foto anterior.
	c2, w2 := contextoPrueba(http.MethodPost, "/", nil, u.ID)
	c2.Request = peticionFoto(t, "archivo", "not.txt", []byte("texto plano"))
	h.SubirFotoDocumento(c2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("un archivo que no es imagen deberia rechazarse, status=%d", w2.Code)
	}
	var posterior models.User
	_ = db.First(&posterior, u.ID)
	if posterior.FotoDocumento != actualizado.FotoDocumento {
		t.Error("un envio rechazado no debia reemplazar la foto anterior")
	}
}

func TestSubirFotoSinArchivo(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "jugador")
	h := NewCuentaHandler(db, &config.Config{UploadDir: t.TempDir()})

	c, w := contextoPrueba(http.MethodPost, "/", nil, u.ID)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	c.Request.Header.Set("Content-Type", "application/json")
	h.SubirFotoDocumento(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("sin archivo deberia ser 400, fue %d", w.Code)
	}
}

func TestIdentificacionConFechaNacimiento(t *testing.T) {
	db := dbPruebas(t)
	u := usuarioPrueba(t, db, "jugador")
	h := NewCuentaHandler(db, &config.Config{UploadDir: t.TempDir()})

	enviar := func(body string) *httptest.ResponseRecorder {
		c, w := contextoPrueba(http.MethodPut, "/", nil, u.ID)
		c.Request = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		h.IdentificacionGuardar(c)
		return w
	}

	// Menor de edad: se rechaza.
	menor := time.Now().AddDate(-10, 0, 0).Format("2006-01-02")
	w := enviar(`{"tipo":"CEDULA_FISICA","numero":"123456","fecha_nacimiento":"` + menor + `"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("un menor de edad deberia rechazarse, status=%d", w.Code)
	}

	// Fecha con formato invalido: se rechaza.
	w = enviar(`{"tipo":"CEDULA_FISICA","numero":"123456","fecha_nacimiento":"15/11/1990"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("una fecha mal formada deberia rechazarse, status=%d", w.Code)
	}

	// Mayor de edad: se guarda y deja el documento sin verificar.
	mayor := time.Now().AddDate(-30, 0, 0).Format("2006-01-02")
	w = enviar(`{"tipo":"CEDULA_FISICA","numero":"123456","fecha_nacimiento":"` + mayor + `"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d (%s)", w.Code, w.Body.String())
	}
	var guardado models.User
	_ = db.First(&guardado, u.ID)
	if guardado.FechaNacimiento == nil {
		t.Fatal("la fecha de nacimiento no quedo guardada")
	}
	if EdadEn(*guardado.FechaNacimiento, time.Now()) < EdadMinima {
		t.Error("la fecha guardada corresponde a un menor")
	}
	if guardado.DocumentoVerificado {
		t.Error("guardar identificacion no deberia dejar el documento verificado")
	}
}

func TestVerificarUsuarioExigeKYCCompleto(t *testing.T) {
	db := dbPruebas(t)
	admin := usuarioPrueba(t, db, "admin")
	u := usuarioPrueba(t, db, "jugador")
	admin.Rol = models.RolAdmin
	_ = db.Save(&admin).Error
	h := NewAdminHandler(db)

	verificar := func(id uint) *httptest.ResponseRecorder {
		c, w := contextoPrueba(http.MethodPost, "/", nil, admin.ID)
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(id), 10)}}
		h.VerificarUsuario(c)
		return w
	}

	// Sin identificacion completa no se puede verificar.
	if w := verificar(u.ID); w.Code != http.StatusBadRequest {
		t.Errorf("sin KYC completo deberia fallar, status=%d", w.Code)
	}

	u.TipoDocumento = models.DocCedulaFisica
	u.NumeroDocumento = "123456"
	u.FechaNacimiento = ptrFecha(time.Now().AddDate(-30, 0, 0))
	u.FotoDocumento = "/tmp/doc.png"
	_ = db.Save(&u).Error

	if w := verificar(u.ID); w.Code != http.StatusOK {
		t.Errorf("con KYC completo deberia verificarse, status=%d (%s)", w.Code, w.Body.String())
	}
	var verificado models.User
	_ = db.First(&verificado, u.ID)
	if !verificado.DocumentoVerificado {
		t.Error("el documento quedo verificado")
	}

	// La accion queda en la auditoria.
	var entradas []models.AuditLog
	_ = db.Find(&entradas).Error
	if len(entradas) == 0 || entradas[0].Accion != "kyc.verificar" {
		t.Error("la verificacion no quedo registrada en la auditoria")
	}
}

func ptrFecha(t time.Time) *time.Time { return &t }
