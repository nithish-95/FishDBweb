package models

// Fish struct represents a fish species
type Fish struct {
	Class            string   `json:"class"`
	Order            string   `json:"order"`
	Family           string   `json:"family"`
	Species          string   `json:"species"`
	CommonName       string   `json:"common_name"`
	Image            string   `json:"image"`
	Features         []string `json:"features"`
	MinSizeCm        float64  `json:"min_size_cm,omitempty"`
	MaxSizeCm        float64  `json:"max_size_cm,omitempty"`
	Diet             string   `json:"diet,omitempty"`
	ConservationStatus string   `json:"conservation_status,omitempty"`
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

type KimiChatResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Created   int64  `json:"created"`
	Model     string `json:"model"`
	Choices   []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		CachedTokens     int `json:"cached_tokens"`
	} `json:"usage"`
}
