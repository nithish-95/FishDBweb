package data

import (
	"encoding/json"
	"os"

	"github.com/nithish-95/ShoalsofSapphire/internal/models"
)

var (
	FishDB models.FishDatabase
)

// Load fish data from JSON file
func LoadFishData() error {
	data, err := os.ReadFile("db/fish_data.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &FishDB)
}
