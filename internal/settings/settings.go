package settings

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
	TextWithChannelLink  string `json:"TextWithChannelLink"`
	InstructionsForActivatingTheCoupon 	string `json:"InstructionsForActivatingTheCoupon"`
	ExceedingTicketActivationLimit   	string `json:"exceedingTicketActivationLimit"`
	MessageTiketActivationReady 		string `json:"MessageTiketActivationReady"`
	SubscribeRequiredText string `json:"SubscribeRequiredText"`
    SubscribeButtonText   string `json:"SubscribeButtonText,omitempty"`
}

type Links struct {
    Main   string `json:"Main"`
    Rating string `json:"Rating"`
	Channel string `json:"Channel"`
	ChannelID int64  `json:"ChannelID"`
}

type Settings struct {
    Message Messages `json:"Messages"`
    Link    Links    `json:"Links"`
	DayActivatianLimit int `json:"DayActivatianLimit"`	
	NumberTicketToCheck int `json:"NumberTicketToCheck"`
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

