package apidocs

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var specification []byte

func Specification(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", specification)
}

func UI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html><html><head><title>Magic Stream Movies API</title><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>body{margin:0;background:#fafafa}</style></head><body><redoc spec-url="/openapi.yaml"></redoc><script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script></body></html>`))
}
