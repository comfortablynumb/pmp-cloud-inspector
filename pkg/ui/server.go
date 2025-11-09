package ui

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

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
	http.HandleFunc("/compare", s.handleCompare)
	http.HandleFunc("/api/stats", s.handleStats)

	addr := fmt.Sprintf(":%d", s.port)

	// #nosec G114 - This is a local development server for viewing cloud resources
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// handleIndex serves the main page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// uploadResponse represents the JSON response for upload
type uploadResponse struct {
	Success   bool                        `json:"success"`
	Error     string                      `json:"error,omitempty"`
	Resources []*resource.Resource        `json:"resources,omitempty"`
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
	defer func() {
		//nolint:errcheck // Ignore error on close since we can't return it from defer
		file.Close()
	}()

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
	if err := json.NewEncoder(w).Encode(uploadResponse{
		Success:   true,
		Resources: collection.Resources,
		Metadata:  collection.Metadata,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleStats returns statistics for the loaded data
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// This is a placeholder - stats are calculated on the client side
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	//nolint:errcheck // Error already being sent to client via HTTP status
	json.NewEncoder(w).Encode(uploadResponse{
		Success: false,
		Error:   message,
	})
}

// compareResponse represents the JSON response for compare
type compareResponse struct {
	Success          bool                 `json:"success"`
	Error            string               `json:"error,omitempty"`
	BaseTimestamp    time.Time            `json:"base_timestamp,omitempty"`
	CompareTimestamp time.Time            `json:"compare_timestamp,omitempty"`
	Added            []*resource.Resource `json:"added,omitempty"`
	Removed          []*resource.Resource `json:"removed,omitempty"`
	Modified         []resourceDiff       `json:"modified,omitempty"`
	Unchanged        []*resource.Resource `json:"unchanged,omitempty"`
	Summary          driftSummary         `json:"summary,omitempty"`
}

// resourceDiff represents changes to a resource
type resourceDiff struct {
	ResourceID   string                    `json:"resource_id"`
	ResourceType resource.ResourceType     `json:"resource_type"`
	Name         string                    `json:"name"`
	Changes      map[string]propertyChange `json:"changes"`
	BaseResource *resource.Resource        `json:"base_resource,omitempty"`
	NewResource  *resource.Resource        `json:"new_resource,omitempty"`
}

// propertyChange represents a change in a resource property
type propertyChange struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// driftSummary provides summary statistics
type driftSummary struct {
	TotalAdded     int `json:"total_added"`
	TotalRemoved   int `json:"total_removed"`
	TotalModified  int `json:"total_modified"`
	TotalUnchanged int `json:"total_unchanged"`
}

// handleCompare handles multiple file uploads and comparison
func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(64 << 20); err != nil { // 64 MB max
		s.sendCompareError(w, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	// Get both files
	baseFile, baseHeader, err := r.FormFile("baseFile")
	if err != nil {
		s.sendCompareError(w, fmt.Sprintf("Failed to get base file: %v", err))
		return
	}
	defer func() {
		//nolint:errcheck // Ignore error on close since we can't return it from defer
		baseFile.Close()
	}()

	compareFile, compareHeader, err := r.FormFile("compareFile")
	if err != nil {
		s.sendCompareError(w, fmt.Sprintf("Failed to get compare file: %v", err))
		return
	}
	defer func() {
		//nolint:errcheck // Ignore error on close since we can't return it from defer
		compareFile.Close()
	}()

	// Read and parse base file
	baseCollection, err := s.parseFile(baseFile, baseHeader.Filename)
	if err != nil {
		s.sendCompareError(w, fmt.Sprintf("Failed to parse base file: %v", err))
		return
	}

	// Read and parse compare file
	compareCollection, err := s.parseFile(compareFile, compareHeader.Filename)
	if err != nil {
		s.sendCompareError(w, fmt.Sprintf("Failed to parse compare file: %v", err))
		return
	}

	// Generate drift report
	report := s.generateDriftReport(baseCollection, compareCollection)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// parseFile parses a file based on its extension
func (s *Server) parseFile(file io.Reader, filename string) (*resource.Collection, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var collection resource.Collection
	filename = strings.ToLower(filename)

	if strings.HasSuffix(filename, ".json") {
		if err := json.Unmarshal(content, &collection); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		if err := yaml.Unmarshal(content, &collection); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported file format")
	}

	return &collection, nil
}

// generateDriftReport generates a drift report between two collections
func (s *Server) generateDriftReport(base, compare *resource.Collection) compareResponse {
	report := compareResponse{
		Success:          true,
		BaseTimestamp:    base.Metadata.Timestamp,
		CompareTimestamp: compare.Metadata.Timestamp,
		Added:            make([]*resource.Resource, 0),
		Removed:          make([]*resource.Resource, 0),
		Modified:         make([]resourceDiff, 0),
		Unchanged:        make([]*resource.Resource, 0),
	}

	// Create index maps for quick lookup
	baseIndex := make(map[string]*resource.Resource)
	compareIndex := make(map[string]*resource.Resource)

	for _, res := range base.Resources {
		baseIndex[res.ID] = res
	}

	for _, res := range compare.Resources {
		compareIndex[res.ID] = res
	}

	// Find removed resources (in base but not in compare)
	for id, res := range baseIndex {
		if _, exists := compareIndex[id]; !exists {
			report.Removed = append(report.Removed, res)
		}
	}

	// Find added, modified, and unchanged resources
	for id, compareRes := range compareIndex {
		baseRes, exists := baseIndex[id]
		if !exists {
			// Resource added
			report.Added = append(report.Added, compareRes)
		} else {
			// Check if modified
			diff := s.compareResources(baseRes, compareRes)
			if len(diff.Changes) > 0 {
				report.Modified = append(report.Modified, diff)
			} else {
				report.Unchanged = append(report.Unchanged, compareRes)
			}
		}
	}

	// Update summary
	report.Summary.TotalAdded = len(report.Added)
	report.Summary.TotalRemoved = len(report.Removed)
	report.Summary.TotalModified = len(report.Modified)
	report.Summary.TotalUnchanged = len(report.Unchanged)

	return report
}

// compareResources compares two resources and returns differences
func (s *Server) compareResources(base, compare *resource.Resource) resourceDiff {
	diff := resourceDiff{
		ResourceID:   base.ID,
		ResourceType: base.Type,
		Name:         base.Name,
		Changes:      make(map[string]propertyChange),
		BaseResource: base,
		NewResource:  compare,
	}

	// Compare basic fields
	if base.Name != compare.Name {
		diff.Changes["name"] = propertyChange{Old: base.Name, New: compare.Name}
	}
	if base.Region != compare.Region {
		diff.Changes["region"] = propertyChange{Old: base.Region, New: compare.Region}
	}

	// Compare tags using deep equality check
	if !s.deepEqual(base.Tags, compare.Tags) {
		diff.Changes["tags"] = propertyChange{Old: base.Tags, New: compare.Tags}
	}

	// Compare properties
	for key, baseValue := range base.Properties {
		compareValue, exists := compare.Properties[key]
		if !exists {
			diff.Changes[fmt.Sprintf("properties.%s", key)] = propertyChange{Old: baseValue, New: nil}
		} else if !s.deepEqual(baseValue, compareValue) {
			diff.Changes[fmt.Sprintf("properties.%s", key)] = propertyChange{Old: baseValue, New: compareValue}
		}
	}

	// Check for new properties
	for key, compareValue := range compare.Properties {
		if _, exists := base.Properties[key]; !exists {
			diff.Changes[fmt.Sprintf("properties.%s", key)] = propertyChange{Old: nil, New: compareValue}
		}
	}

	return diff
}

// deepEqual performs deep equality check using JSON serialization
func (s *Server) deepEqual(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// sendCompareError sends an error response for compare endpoint
func (s *Server) sendCompareError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	//nolint:errcheck // Error already being sent to client via HTTP status
	json.NewEncoder(w).Encode(compareResponse{
		Success: false,
		Error:   message,
	})
}
