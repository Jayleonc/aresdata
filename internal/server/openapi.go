package server

import (
	"net/http"
	"os"
)

// OpenAPIHandler serves the generated OpenAPI YAML file.
func OpenAPIHandler(openapiPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(openapiPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("openapi.yaml not found"))
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = file.Stat()
		_, err = file.Seek(0, 0)
		if err == nil {
			_, _ = w.Write([]byte("# Served by backend\n"))
		}
		buf := make([]byte, 4096)
		for {
			n, err := file.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}
}

// SwaggerUIHandler serves static files for Swagger UI
func SwaggerUIHandler() http.Handler {
	return http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("swagger-ui")))
}
