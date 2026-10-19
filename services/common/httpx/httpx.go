// Package httpx: maps domain errors (errs.Kind) to HTTP JSON responses {"error": "..."}.
package httpx

import (
	"log"
	"net/http"
	"strings"

	"eam/services/common/errs"
	"github.com/gin-gonic/gin"
)

var statusOf = map[errs.Kind]int{
	errs.KindInvalid:      http.StatusBadRequest,
	errs.KindNotFound:     http.StatusNotFound,
	errs.KindConflict:     http.StatusConflict,
	errs.KindForbidden:    http.StatusForbidden,
	errs.KindUnauthorized: http.StatusUnauthorized,
	errs.KindUnavailable:  http.StatusBadGateway,
}

// Error writes the error; internal errors are only logged, the user sees a generic message.
func Error(c *gin.Context, err error) {
	if code, ok := statusOf[errs.KindOf(err)]; ok {
		c.AbortWithStatusJSON(code, gin.H{"error": err.Error()})
		return
	}
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "an internal server error occurred"})
}

// Bind reads the JSON body; returns false + 400 when invalid.
func Bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "payload is invalid"})
		return false
	}
	return true
}

// CSVSafe prevents CSV/formula injection when the file is opened in Excel (prefixes = + - @ tab CR).
func CSVSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
