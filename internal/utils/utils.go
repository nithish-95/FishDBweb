package utils

import (
	"sort"
	"strings"

	"github.com/nithish-95/ShoalsofSapphire/internal/models"
)

// Helper to get unique values for filters
func GetUniqueCategories(fishDB models.FishDatabase) (map[string]struct{}, map[string]struct{}, map[string]struct{}) {
	classes := make(map[string]struct{})
	orders := make(map[string]struct{})
	families := make(map[string]struct{})

	for _, fish := range fishDB.Fishes {
		classes[fish.Class] = struct{}{}
		orders[fish.Order] = struct{}{}
		families[fish.Family] = struct{}{}
	}
	return classes, orders, families
}

// Filter and sort fish based on query parameters
func GetFilteredAndSortedFish(
	fishDB models.FishDatabase,
	searchQuery, classFilter, orderFilter, familyFilter, sortBy string,
) []models.Fish {
	filteredFish := []models.Fish{}
	lowerSearchQuery := strings.ToLower(searchQuery)
	lowerClassFilter := strings.ToLower(classFilter)
	lowerOrderFilter := strings.ToLower(orderFilter)
	lowerFamilyFilter := strings.ToLower(familyFilter)

	for _, fish := range fishDB.Fishes {
		// Apply text search filter
		matchesSearch := true
		if searchQuery != "" {
			if !(strings.Contains(strings.ToLower(fish.CommonName), lowerSearchQuery) ||
				strings.Contains(strings.ToLower(fish.Species), lowerSearchQuery) ||
				strings.Contains(strings.ToLower(fish.Family), lowerSearchQuery) ||
				strings.Contains(strings.ToLower(fish.Order), lowerSearchQuery) ||
				strings.Contains(strings.ToLower(fish.Class), lowerSearchQuery)) {
				matchesSearch = false
			}
		}

		// Apply taxonomic filters
		matchesClass := (classFilter == "" || classFilter == "All" || strings.ToLower(fish.Class) == lowerClassFilter)
		matchesOrder := (orderFilter == "" || orderFilter == "All" || strings.ToLower(fish.Order) == lowerOrderFilter)
		matchesFamily := (familyFilter == "" || familyFilter == "All" || strings.ToLower(fish.Family) == lowerFamilyFilter)

		if matchesSearch && matchesClass && matchesOrder && matchesFamily {
			filteredFish = append(filteredFish, fish)
		}
	}

	// Apply sorting
	sort.Slice(filteredFish, func(i, j int) bool {
		switch sortBy {
		case "common_name":
			return strings.ToLower(filteredFish[i].CommonName) < strings.ToLower(filteredFish[j].CommonName)
		case "species":
			return strings.ToLower(filteredFish[i].Species) < strings.ToLower(filteredFish[j].Species)
		case "family":
			return strings.ToLower(filteredFish[i].Family) < strings.ToLower(filteredFish[j].Family)
		default: // Default to common_name if sortBy is empty or unknown
			return strings.ToLower(filteredFish[i].CommonName) < strings.ToLower(filteredFish[j].CommonName)
		}
	})

	return filteredFish
}
