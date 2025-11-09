package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

//go:embed templates/*
var templatesFS embed.FS

// Server represents the UI web server
type Server struct {
	port      int
	templates *template.Template
}

// NewServer creates a new UI server
func NewServer(port int) *Server {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.html"))
	return &Server{
		port:      port,
		templates: tmpl,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/upload", s.handleUpload)
	http.HandleFunc("/api/stats", s.handleStats)

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, nil)
}

// handleIndex serves the main page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// uploadResponse represents the JSON response for upload
type uploadResponse struct {
	Success   bool                    `json:"success"`
	Error     string                  `json:"error,omitempty"`
	Resources []*resource.Resource    `json:"resources,omitempty"`
	Metadata  resource.CollectionMetadata `json:"metadata,omitempty"`
}

// handleUpload handles file uploads
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB max
		s.sendError(w, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to get file: %v", err))
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	// Parse based on file extension
	var collection resource.Collection
	filename := strings.ToLower(header.Filename)

	if strings.HasSuffix(filename, ".json") {
		if err := json.Unmarshal(content, &collection); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to parse JSON: %v", err))
			return
		}
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		if err := yaml.Unmarshal(content, &collection); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to parse YAML: %v", err))
			return
		}
	} else {
		s.sendError(w, "Unsupported file format. Please upload JSON or YAML files.")
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploadResponse{
		Success:   true,
		Resources: collection.Resources,
		Metadata:  collection.Metadata,
	})
}

// handleStats returns statistics for the loaded data
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder - stats are calculated on the client side
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(uploadResponse{
		Success: false,
		Error:   message,
	})
}
