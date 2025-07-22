package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/go-chi/chi/v5"

	"github.com/nithish-95/ShoalsofSapphire/internal/models"
	"github.com/nithish-95/ShoalsofSapphire/internal/utils"
)

// Handlers struct holds dependencies for HTTP handlers
type Handlers struct {
	FishDB    *models.FishDatabase
	Templates *template.Template
	AICache   map[string]string
	CacheMutex sync.Mutex
}

// NewHandlers creates a new Handlers instance
func NewHandlers(fishDB *models.FishDatabase, tmpl *template.Template) *Handlers {
	return &Handlers{
		FishDB:    fishDB,
		Templates: tmpl,
		AICache:   make(map[string]string),
		CacheMutex: sync.Mutex{},
	}
}

// Home page handler
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	rand.Seed(time.Now().UnixNano())
	shuffledFishes := make([]models.Fish, len(h.FishDB.Fishes))
	copy(shuffledFishes, h.FishDB.Fishes)
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
		FeaturedFish []models.Fish
	}{
		Title:        h.FishDB.Title,
		Publisher:    h.FishDB.PublicationDetails.Publisher,
		FeaturedFish: shuffledFishes[:featuredCount],
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.Templates.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// List fish handler
func (h *Handlers) ListFish(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search")
	classFilter := r.URL.Query().Get("class")
	orderFilter := r.URL.Query().Get("order")
	familyFilter := r.URL.Query().Get("family")
	sortBy := r.URL.Query().Get("sort")

	// Get all filtered and sorted fish first
	allFilteredFish := utils.GetFilteredAndSortedFish(*h.FishDB, searchQuery, classFilter, orderFilter, familyFilter, sortBy)

	// Determine the number of fish to show on the first page
	limit := 12
	initialFish := []models.Fish{}
	if len(allFilteredFish) > 0 {
		if len(allFilteredFish) < limit {
			initialFish = allFilteredFish
		} else {
			initialFish = allFilteredFish[:limit]
		}
	}

	uniqueClassesMap, uniqueOrdersMap, uniqueFamiliesMap := utils.GetUniqueCategories(*h.FishDB)

	// Convert maps to slices for template iteration
	uniqueClasses := []string{}
	for k := range uniqueClassesMap {
		uniqueClasses = append(uniqueClasses, k)
	}
	sort.Strings(uniqueClasses) // Sort for consistent display

	uniqueOrders := []string{}
	for k := range uniqueOrdersMap {
		uniqueOrders = append(uniqueOrders, k)
	}
	sort.Strings(uniqueOrders)

	uniqueFamilies := []string{}
	for k := range uniqueFamiliesMap {
		uniqueFamilies = append(uniqueFamilies, k)
	}
	sort.Strings(uniqueFamilies)

	data := struct {
		SearchQuery    string
		ClassFilter    string
		OrderFilter    string
		FamilyFilter   string
		SortBy         string
		FishList       []models.Fish // Only the initial page of fish
		TotalFishCount int           // Total count after filtering/sorting
		Publication    models.PublicationDetails
		UniqueClasses  []string
		UniqueOrders   []string
		UniqueFamilies []string
	}{
		SearchQuery:    searchQuery,
		ClassFilter:    classFilter,
		OrderFilter:    orderFilter,
		FamilyFilter:   familyFilter,
		SortBy:         sortBy,
		FishList:       initialFish,
		TotalFishCount: len(allFilteredFish),
		Publication:    h.FishDB.PublicationDetails,
		UniqueClasses:  uniqueClasses,
		UniqueOrders:   uniqueOrders,
		UniqueFamilies: uniqueFamilies,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.Templates.ExecuteTemplate(w, "list.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Fish detail handler
func (h *Handlers) FishDetail(w http.ResponseWriter, r *http.Request) {
	speciesParam := chi.URLParam(r, "species")

	var fish models.Fish
	var index int
	found := false
	for i, f := range h.FishDB.Fishes {
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

	// Find related species (same family, then same order if not enough)
	relatedSpecies := []models.Fish{}
	const maxRelated = 4 // Max number of related species to show

	// Try to find species from the same family first
	for _, f := range h.FishDB.Fishes {
		if f.Species != fish.Species && f.Family == fish.Family {
			relatedSpecies = append(relatedSpecies, f)
			if len(relatedSpecies) >= maxRelated {
				break
			}
		}
	}

	// If not enough, find species from the same order
	if len(relatedSpecies) < maxRelated {
		for _, f := range h.FishDB.Fishes {
			if f.Species != fish.Species && f.Order == fish.Order && f.Family != fish.Family {
				// Check if already added to avoid duplicates
				foundInRelated := false
				for _, rs := range relatedSpecies {
					if rs.Species == f.Species {
						foundInRelated = true
						break
					}
				}
				if !foundInRelated {
					relatedSpecies = append(relatedSpecies, f)
					if len(relatedSpecies) >= maxRelated {
						break
					}
				}
			}
		}
	}

	data := struct {
		Fish           models.Fish
		Index          int
		AllFish        []models.Fish
		RelatedSpecies []models.Fish
		Publication    models.PublicationDetails
		DatabaseTitle  string // Add this field
	}{
		Fish:           fish,
		Index:          index,
		AllFish:        h.FishDB.Fishes,
		RelatedSpecies: relatedSpecies,
		Publication:    h.FishDB.PublicationDetails,
		DatabaseTitle:  h.FishDB.Title, // Assign the main database title
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.Templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Favorites page handler
func (h *Handlers) FavoritesPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title     string
		Publisher string
	}{
		Title:     h.FishDB.Title,
		Publisher: h.FishDB.PublicationDetails.Publisher,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.Templates.ExecuteTemplate(w, "favorites.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// API endpoint: Get all fish with pagination, filtering, and sorting
func (h *Handlers) APIAllFish(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	searchQuery := r.URL.Query().Get("search")
	classFilter := r.URL.Query().Get("class")
	orderFilter := r.URL.Query().Get("order")
	familyFilter := r.URL.Query().Get("family")
	sortBy := r.URL.Query().Get("sort")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 12
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	// Get all filtered and sorted fish
	allFilteredFish := utils.GetFilteredAndSortedFish(*h.FishDB, searchQuery, classFilter, orderFilter, familyFilter, sortBy)

	startIndex := (page - 1) * limit
	endIndex := startIndex + limit

	if startIndex >= len(allFilteredFish) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]models.Fish{})
		return
	}

	if endIndex > len(allFilteredFish) {
		endIndex = len(allFilteredFish)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allFilteredFish[startIndex:endIndex])
}

// API endpoint: Get fish by a list of species names
func (h *Handlers) APIFishBySpecies(w http.ResponseWriter, r *http.Request) {
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

	var results []models.Fish
	for _, fish := range h.FishDB.Fishes {
		if _, found := speciesSet[fish.Species]; found {
			results = append(results, fish)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// API endpoint: Get specific fish by species
func (h *Handlers) APIFishDetail(w http.ResponseWriter, r *http.Request) {
	speciesParam := chi.URLParam(r, "species")

	for _, fish := range h.FishDB.Fishes {
		if fish.Species == speciesParam {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(fish)
			return
		}
	}

	http.Error(w, "Fish not found", http.StatusNotFound)
}

// API endpoint: Search fish (now deprecated in favor of apiAllFish with filters)
func (h *Handlers) APISearch(w http.ResponseWriter, r *http.Request) {
	// This endpoint is now effectively deprecated as apiAllFish handles all filtering and searching.
	// Redirect or return an error, or simply call apiAllFish internally.
	// For now, I'll just call apiAllFish to avoid breaking existing calls.
	h.APIAllFish(w, r)
}

// API endpoint: Ask AI
func (h *Handlers) APIAskAI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Create the chat request for Kimi k2
	kimiReq := models.OllamaChatRequest{ // Reusing OllamaChatRequest struct as it's compatible
		Model: "kimi-k2-0711-preview", // Changed to Kimi k2 model
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: query,
			},
		},
		Stream: false,
		Options: &models.ModelOptions{
			Temperature: 0.7, // Adjust for creativity vs accuracy
		},
	}

	// Check cache first
	h.CacheMutex.Lock()
	if cachedResponse, found := h.AICache[query]; found {
		h.CacheMutex.Unlock()
		log.Printf("Returning cached AI response for query: %s", query)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ai_response": cachedResponse})
		return
	}
	h.CacheMutex.Unlock()

	jsonReq, err := json.Marshal(kimiReq)
	if err != nil {
		log.Printf("Error marshalling Kimi k2 request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest(
		"POST",
		os.Getenv("KIMI_K2_URL"),
		bytes.NewBuffer(jsonReq),
	)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("KIMI_K2_API"))

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Kimi k2 response body: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	log.Printf("Kimi k2 API raw response body: %s", string(body))

	// Parse the response
	var kimiResp models.KimiChatResponse
	if err := json.Unmarshal(body, &kimiResp); err != nil {
		log.Printf("Error decoding Kimi k2 response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(kimiResp.Choices) == 0 || kimiResp.Choices[0].Message.Content == "" {
		log.Printf("Kimi k2 API response content is empty or malformed: %+v", kimiResp)
		http.Error(w, "Empty or malformed AI response", http.StatusInternalServerError)
		return
	}

	aiResponseContent := kimiResp.Choices[0].Message.Content

	// Store in cache
	h.CacheMutex.Lock()
	h.AICache[query] = aiResponseContent
	h.CacheMutex.Unlock()

	log.Printf("Kimi k2 AI response content: %s", aiResponseContent)

	// Return the AI response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ai_response": aiResponseContent})
}

// API endpoint for search suggestions
func (h *Handlers) APISuggestions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]string{})
		return
	}

	lowerQuery := strings.ToLower(query)
	suggestionsMap := make(map[string]struct{}) // Use a map to store unique suggestions

	for _, fish := range h.FishDB.Fishes {
		if strings.Contains(strings.ToLower(fish.CommonName), lowerQuery) {
			suggestionsMap[fish.CommonName] = struct{}{}
		}
		if strings.Contains(strings.ToLower(fish.Species), lowerQuery) {
			suggestionsMap[fish.Species] = struct{}{}
		}
	}

	suggestions := []string{}
	for s := range suggestionsMap {
		suggestions = append(suggestions, s)
	}
	sort.Strings(suggestions) // Sort suggestions alphabetically

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
