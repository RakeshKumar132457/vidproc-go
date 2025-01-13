package swagger

import (
	"embed"
	"log"
	"net/http"
)

//go:embed docs/*
var swaggerFiles embed.FS

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Swagger request path: %s", r.URL.Path)

		switch r.URL.Path {
		case "", "/", "/index.html":
			data, err := swaggerFiles.ReadFile("docs/swagger-ui.html")
			if err != nil {
				log.Printf("Error reading swagger-ui.html: %v", err)
				http.Error(w, "Swagger UI not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)

		case "/swagger.yaml":
			data, err := swaggerFiles.ReadFile("docs/swagger.yaml")
			if err != nil {
				log.Printf("Error reading swagger.yaml: %v", err)
				http.Error(w, "Swagger spec not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/yaml")
			w.Write(data)

		default:
			log.Printf("Path not found: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
}
