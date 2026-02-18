package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Messages struct {
    Hello                string `json:"Hello"`
    RegistrationQuestion string `json:"RegistrationQuestion"`
    MenuText             string `json:"MenuText"`
    PromotionRules       string `json:"promotionRules"`
    ActiveTicketsCount   string `json:"ActiveTicketsCount"`
}

type Links struct {
    Main   string `json:"Main"`
    Rating string `json:"Rating"`
}

type Settings struct {
    Message Messages `json:"Messages"`
    Link    Links    `json:"Links"`
}

func NewSetting(path string) (Settings, error) {
	file, err := os.Open(path)
	if err != nil {
		return Settings{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var setting Settings
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&setting); err != nil {
		return Settings{}, fmt.Errorf("decode json: %w", err)
	}

	return setting, nil
}

