package middleware

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mateo/ms-gestor-de-info/backend/internal/utils"
)

// encabezado con el que se envia el ID entre microservicios
const RequestIDHeader = "X-Request-ID"

const maxRequestIDLength = 64

// ID se devuelve en la respuesta y aparece en los logs y en los errores
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !isValidRequestID(id) {
			id = newRequestID()
		}
		c.Set(utils.RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// evita que un cliente inyecte texto arbitrario en los logs
func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, r := range id {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !isAlnum && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

// genera un UUID sin dependencias externas
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
