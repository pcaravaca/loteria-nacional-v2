package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader es el encabezado con el que el cliente puede traer su propio
// identificador de peticion (asi un reintento conserva el mismo). Si no viene,
// el servidor genera uno.
const RequestIDHeader = "X-Request-ID"

// RequestID deja en el contexto un identificador unico por peticion y lo
// devuelve en la respuesta. Lo usa el registro de acciones para que CS pueda
// correlacionar una jugada de punta a punta.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = NuevoRequestID()
		}
		c.Set("request_id", id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// NuevoRequestID genera un identificador opaco y unico.
func NuevoRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-0000000000000000"
	}
	return "req-" + hex.EncodeToString(b[:])
}
