package handler

import (
	"embed"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed docs_swagger_ui.html
var swaggerUIHTML embed.FS

// DocsHandler serves OpenAPI spec and Swagger UI.
type DocsHandler struct {
	specPath string
}

// NewDocsHandler creates a handler that serves the OpenAPI spec from specPath.
func NewDocsHandler(specPath string) *DocsHandler {
	return &DocsHandler{specPath: specPath}
}

// ServeSpec serves the raw OpenAPI YAML file.
func (h *DocsHandler) ServeSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	http.ServeFile(w, r, filepath.Clean(h.specPath))
}

// ServeUI serves a Swagger UI page that loads the spec.
func (h *DocsHandler) ServeUI(w http.ResponseWriter, r *http.Request) {
	// Serve index at root or /index.html, 404 for other sub-paths
	p := strings.TrimPrefix(r.URL.Path, "/api/docs")
	if p != "" && p != "/" && p != "/index.html" {
		http.NotFound(w, r)
		return
	}
	data, err := swaggerUIHTML.ReadFile("docs_swagger_ui.html")
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
