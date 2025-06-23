package models

import "star-fire/pkg/public"

type ClientModel struct {
	Client *Client       `json:"client"`
	Model  *public.Model `json:"model"`
}

type MarketplaceModel struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Size         string         `json:"size"`
	Quantization string         `json:"quantization"`
	ClientModels []*ClientModel `json:"client_models"`
}
