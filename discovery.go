package mmoney

import (
	"encoding/json"
	"net/http"
)

type AppHealth struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
	Version     string `json:"version"`
	Status      string `json:"status"`
}

var DefaultHealth = AppHealth{
	Name:        "mmoney",
	DisplayName: "MMoney",
	Icon:        "wallet",
	Version:     "1.0.0",
	Status:      "healthy",
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DefaultHealth)
}
