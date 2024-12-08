package models

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type PluginSettings struct {
	CriblOrgBaseUrl string                `json:"criblOrgBaseUrl"`
	ClientId        string                `json:"clientId"`
	QueryTimeoutSec *float64              `json:"queryTimeoutSec"`
	Secrets         *SecretPluginSettings `json:"-"`
}

type SecretPluginSettings struct {
	ClientSecret string `json:"clientSecret"`
}

func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
	}

	settings.Secrets = loadSecretPluginSettings(source.DecryptedSecureJSONData)

	return &settings, nil
}

func loadSecretPluginSettings(source map[string]string) *SecretPluginSettings {
	return &SecretPluginSettings{
		ClientSecret: source["clientSecret"],
	}
}
