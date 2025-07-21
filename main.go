package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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

// Ollama Chat Request/Response structures
type OllamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  *ModelOptions `json:"options,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ModelOptions struct {
	Temperature float32 `json:"temperature"`
}

type OllamaChatResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

var (
	fishDB    FishDatabase
	templates *template.Template
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

	tmpl := template.New("").Funcs(funcMap)
	var err error
	templates, err = tmpl.ParseGlob("templates/*.html")
	return err
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
	rand.Seed(time.Now().UnixNano())
	shuffledFishes := make([]Fish, len(fishDB.Fishes))
	copy(shuffledFishes, fishDB.Fishes)
	rand.Shuffle(len(shuffledFishes), func(i, j int) {
		shuffledFishes[i], shuffledFishes[j] = shuffledFishes[j], shuffledFishes[i]
	})

	featuredCount := 4
	if len(shuffledFishes) < featuredCount {
		featuredCount = len(shuffledFishes)
	}

	data := struct {
		Title        string
		Publisher    string
		FeaturedFish []Fish
	}{
		Title:        fishDB.Title,
		Publisher:    fishDB.PublicationDetails.Publisher,
		FeaturedFish: shuffledFishes[:featuredCount],
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

// Favorites page handler
func favoritesPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title     string
		Publisher string
	}{
		Title:     fishDB.Title,
		Publisher: fishDB.PublicationDetails.Publisher,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := templates.ExecuteTemplate(w, "favorites.html", data); err != nil {
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

// API endpoint: Get all fish with pagination
func apiAllFish(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 12
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	startIndex := (page - 1) * limit
	endIndex := startIndex + limit

	if startIndex >= len(fishDB.Fishes) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Fish{})
		return
	}

	if endIndex > len(fishDB.Fishes) {
		endIndex = len(fishDB.Fishes)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fishDB.Fishes[startIndex:endIndex])
}

// API endpoint: Get fish by a list of species names
func apiFishBySpecies(w http.ResponseWriter, r *http.Request) {
	speciesQuery := r.URL.Query().Get("species")
	if speciesQuery == "" {
		http.Error(w, "Missing 'species' query parameter", http.StatusBadRequest)
		return
	}

	speciesList := strings.Split(speciesQuery, ",")
	speciesSet := make(map[string]struct{})
	for _, s := range speciesList {
		speciesSet[s] = struct{}{}
	}

	var results []Fish
	for _, fish := range fishDB.Fishes {
		if _, found := speciesSet[fish.Species]; found {
			results = append(results, fish)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// API endpoint: Get specific fish by species
func apiFishDetail(w http.ResponseWriter, r *http.Request) {
	speciesParam := chi.URLParam(r, "species")

	for _, fish := range fishDB.Fishes {
		if fish.Species == speciesParam {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(fish)
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
	json.NewEncoder(w).Encode(results)
}

// API endpoint: Ask AI
func apiAskAI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Create the chat request for Kimi k2
	kimiReq := OllamaChatRequest{ // Reusing OllamaChatRequest struct as it's compatible
		Model: "kimi-k2", // Changed to Kimi k2 model
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: query,
			},
		},
		Stream: false,
		Options: &ModelOptions{
			Temperature: 0.7, // Adjust for creativity vs accuracy
		},
	}

	jsonReq, err := json.Marshal(kimiReq)
	if err != nil {
		log.Printf("Error marshalling Kimi k2 request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest(
		"POST",
		"YOUR_KIMI_K2_API_ENDPOINT_HERE", // Placeholder for Kimi k2 API endpoint
		bytes.NewBuffer(jsonReq),
	)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer YOUR_KIMI_K2_API_KEY_HERE") // Placeholder for Kimi k2 API key

	// Make HTTP call
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling Kimi k2 API: %v", err)
		http.Error(w, "Failed to get AI response", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Kimi k2 API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
		http.Error(w, "Kimi k2 API error", http.StatusInternalServerError)
		return
	}

	// Parse the response
	var kimiResp OllamaChatResponse // Reusing OllamaChatResponse struct
	if err := json.NewDecoder(resp.Body).Decode(&kimiResp); err != nil {
		log.Printf("Error decoding Kimi k2 response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the AI response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ai_response": kimiResp.Message.Content})
}

func main() {
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
	r.Get("/favorites", favoritesPage)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/fish", apiAllFish)
		r.Get("/fish/species", apiFishBySpecies)
		r.Get("/fish/{species}", apiFishDetail)
		r.Get("/search", apiSearch)
		r.Get("/ask-ai", apiAskAI)
	})

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server
	fmt.Println("Server starting on port 8080:  http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
