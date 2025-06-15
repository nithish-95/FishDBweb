package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

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
	json.NewEncoder(w).Encode(fishDB.Fishes)
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

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/fish", apiAllFish)
		r.Get("/fish/{species}", apiFishDetail)
		r.Get("/search", apiSearch)
	})

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server
	fmt.Println("Server starting on port 8080:  http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
