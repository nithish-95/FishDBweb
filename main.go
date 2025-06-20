package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Configuration struct
type Config struct {
	OllamaURL   string
	OllamaModel string
	InDocker    bool
}

// Fish struct represents a fish species
type Fish struct {
	Class      string   `json:"class"`
	Order      string   `json:"order"`
	Family     string   `json:"family"`
	Species    string   `json:"species"`
	CommonName string   `json:"common_name"`
	Image      string   `json:"image"`
	Features   []string `json:"features"`
}

// PublicationDetails struct represents publication metadata
type PublicationDetails struct {
	SpecialPublicationNo int      `json:"special_publication_no"`
	ISSN                 string   `json:"issn"`
	Publisher            string   `json:"publisher"`
	Authors              []string `json:"authors"`
	TechnicalSupport     []string `json:"technical_support"`
	Director             string   `json:"director"`
}

// FishDatabase struct represents the entire fish database
type FishDatabase struct {
	Title              string             `json:"title"`
	PublicationDetails PublicationDetails `json:"publication_details"`
	Fishes             []Fish             `json:"fishes"`
}

// OllamaGenerateRequest struct for AI requests
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse represents a simplified response from Ollama
type OllamaGenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

var (
	fishDB    FishDatabase
	templates *template.Template
	config    Config
)

// Initialize templates with custom functions
func initTemplates() error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"len": func(s []Fish) int { return len(s) },
		"lt":  func(a, b int) bool { return a < b },
		"gt":  func(a, b int) bool { return a > b },
	}

	var err error
	templates = template.New("").Funcs(funcMap)
	templates, err = templates.ParseGlob("templates/*.html")
	return err
}

// Load configuration from environment variables
func loadConfig() {
	config = Config{
		OllamaURL:   os.Getenv("OLLAMA_URL"),
		OllamaModel: os.Getenv("OLLAMA_MODEL"),
		InDocker:    os.Getenv("IN_DOCKER") == "true",
	}

	// Set default Ollama URL based on environment
	if config.OllamaURL == "" {
		if config.InDocker {
			config.OllamaURL = "http://host.docker.internal:11434/api/generate"
		} else {
			config.OllamaURL = "http://localhost:11434/api/generate"
		}
	}

	// Set default model
	if config.OllamaModel == "" {
		config.OllamaModel = "tinyllama"
	}
}

// Load fish data from JSON file
func loadFishData() error {
	data, err := os.ReadFile("db/fish_data.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &fishDB)
}

// Home page handler
func home(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title     string
		Publisher string
		AllFish   []Fish
	}{
		Title:     fishDB.Title,
		Publisher: fishDB.PublicationDetails.Publisher,
		AllFish:   fishDB.Fishes,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := templates.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// List fish handler
func listFish(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search")
	var fishList []Fish

	if searchQuery != "" {
		fishList = searchFish(searchQuery)
	} else {
		fishList = fishDB.Fishes
	}

	data := struct {
		SearchQuery string
		FishList    []Fish
		Publication PublicationDetails
	}{
		SearchQuery: searchQuery,
		FishList:    fishList,
		Publication: fishDB.PublicationDetails,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := templates.ExecuteTemplate(w, "list.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Fish detail handler
func fishDetail(w http.ResponseWriter, r *http.Request) {
	speciesParam := chi.URLParam(r, "species")

	var fish Fish
	var index int
	found := false
	for i, f := range fishDB.Fishes {
		if f.Species == speciesParam {
			fish = f
			index = i
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Fish not found", http.StatusNotFound)
		return
	}

	data := struct {
		Fish        Fish
		Index       int
		AllFish     []Fish
		Publication struct {
			Title     string
			Publisher string
			ISSN      string
		}
	}{
		Fish:    fish,
		Index:   index,
		AllFish: fishDB.Fishes,
		Publication: struct {
			Title     string
			Publisher string
			ISSN      string
		}{
			Title:     fishDB.Title,
			Publisher: fishDB.PublicationDetails.Publisher,
			ISSN:      fishDB.PublicationDetails.ISSN,
		},
	}

	w.Header().Set("Content-Type", "text/html")
	if err := templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Search fish function
func searchFish(query string) []Fish {
	var results []Fish
	query = strings.ToLower(query)

	for _, fish := range fishDB.Fishes {
		if strings.Contains(strings.ToLower(fish.CommonName), query) ||
			strings.Contains(strings.ToLower(fish.Species), query) ||
			strings.Contains(strings.ToLower(fish.Family), query) ||
			strings.Contains(strings.ToLower(fish.Order), query) ||
			strings.Contains(strings.ToLower(fish.Class), query) {
			results = append(results, fish)
		}
	}

	return results
}

// API endpoint: Get all fish
func apiAllFish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fishDB.Fishes); err != nil {
		log.Printf("JSON encoding error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// API endpoint: Get specific fish by species
func apiFishDetail(w http.ResponseWriter, r *http.Request) {
	speciesParam := chi.URLParam(r, "species")

	for _, fish := range fishDB.Fishes {
		if fish.Species == speciesParam {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(fish); err != nil {
				log.Printf("JSON encoding error: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}

	http.Error(w, "Fish not found", http.StatusNotFound)
}

// API endpoint: Search fish
func apiSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	results := searchFish(query)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("JSON encoding error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// API endpoint: Ask AI
func apiAskAI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Sanitize input and create prompt
	query = template.HTMLEscapeString(query)
	prompt := fmt.Sprintf(
		"You are a marine biologist. Provide accurate information about %s. "+
			"Include scientific details about its species, habitat, behavior, "+
			"diet, and conservation status. Use professional terminology but "+
			"keep it accessible for enthusiasts.", query)

	ollamaReq := OllamaGenerateRequest{
		Model:  config.OllamaModel,
		Prompt: prompt,
		Stream: false,
	}

	jsonReq, err := json.Marshal(ollamaReq)
	if err != nil {
		log.Printf("Error marshalling Ollama request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(config.OllamaURL, "application/json", bytes.NewBuffer(jsonReq))
	if err != nil {
		log.Printf("Error calling Ollama API: %v", err)
		http.Error(w, "Failed to get AI response", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Ollama API error: Status %d, Body: %s", resp.StatusCode, body)
		http.Error(w, "Ollama API error", http.StatusInternalServerError)
		return
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		log.Printf("Error decoding Ollama response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"ai_response": ollamaResp.Response}); err != nil {
		log.Printf("JSON encoding error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func main() {
	// Load configuration
	loadConfig()

	// Initialize templates
	if err := initTemplates(); err != nil {
		log.Fatalf("Error initializing templates: %v", err)
	}

	// Load fish data
	if err := loadFishData(); err != nil {
		log.Fatalf("Failed to load fish data: %v", err)
	}
	fmt.Printf("Loaded %d fish species from database\n", len(fishDB.Fishes))

	// Create router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Web routes
	r.Get("/", home)
	r.Get("/list", listFish)
	r.Get("/fish/{species}", fishDetail)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/fish", apiAllFish)
		r.Get("/fish/{species}", apiFishDetail)
		r.Get("/search", apiSearch)
		r.Get("/ask-ai", apiAskAI)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server
	fmt.Printf("Server starting on port 8080\n")
	fmt.Printf("Ollama Configuration: URL=%s, Model=%s\n", config.OllamaURL, config.OllamaModel)
	log.Fatal(http.ListenAndServe(":8080", r))
}
