package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/nithish-95/ShoalsofSapphire/internal/data"
	"github.com/nithish-95/ShoalsofSapphire/internal/handlers"
	"github.com/nithish-95/ShoalsofSapphire/internal/templates"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Initialize templates
	if err := templates.InitTemplates(); err != nil {
		log.Fatalf("Error initializing templates: %v", err)
	}

	// Load fish data
	if err := data.LoadFishData(); err != nil {
		log.Fatalf("Failed to load fish data: %v", err)
	}
	fmt.Printf("Loaded %d fish species from database\n", len(data.FishDB.Fishes))

	// Create handlers instance with injected dependencies
	h := handlers.NewHandlers(&data.FishDB, templates.Templates)

	// Create router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Web routes
	r.Get("/", h.Home)
	r.Get("/list", h.ListFish)
	r.Get("/fish/{species}", h.FishDetail)
	r.Get("/favorites", h.FavoritesPage)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/fish", h.APIAllFish)
		r.Get("/fish/species", h.APIFishBySpecies)
		r.Get("/fish/{species}", h.APIFishDetail)
		r.Get("/search", h.APISearch)
		r.Get("/ask-ai", h.APIAskAI)
		r.Get("/suggestions", h.APISuggestions)
	})

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start server
	fmt.Println("Server starting on port 8080:  http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
