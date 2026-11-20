package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"loteria-backend/internal/config"
	"loteria-backend/internal/models"
)

type AuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type registerRequest struct {
	Nombre   string `json:"nombre" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Telefono string `json:"telefono"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	var existente models.User
	if err := h.db.Where("email = ?", req.Email).First(&existente).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ya existe una cuenta con ese correo"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear la cuenta"})
		return
	}

	user := models.User{
		Nombre:       req.Nombre,
		Email:        req.Email,
		PasswordHash: string(hash),
		Rol:          models.RolCliente,
		Telefono:     req.Telefono,
		Saldo:        0,
		Bono:         0,
		TemaCasino:   models.TemaPremium,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la cuenta"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "nombre": user.Nombre, "email": user.Email, "rol": user.Rol})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciales incorrectas"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciales incorrectas"})
		return
	}

	token, err := h.signToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo emitir el token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"nombre":     user.Nombre,
			"email":      user.Email,
			"rol":        user.Rol,
			"saldo":      user.Saldo,
			"bono":       user.Bono,
			"tema_casino": user.TemaCasino,
		},
	})
}

type temaRequest struct {
	Tema string `json:"tema"`
}

// GuardarTema guarda el ambiente visual del casino elegido por el usuario.
func (h *AuthHandler) GuardarTema(c *gin.Context) {
	var req temaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}
	if !models.TemaCasinoValido(req.Tema) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ambiente no disponible"})
		return
	}
	userID := c.GetUint("userID")
	if err := h.db.Model(&models.User{}).Where("id = ?", userID).
		Update("tema_casino", req.Tema).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el ambiente"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tema_casino": req.Tema})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint("userID")
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) signToken(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"uid": user.ID,
		"rol": user.Rol,
		"exp": time.Now().Add(72 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.cfg.JWTSecret))
}
