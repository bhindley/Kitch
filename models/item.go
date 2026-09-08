package models

// Item represents a product in the kitchen inventory.
type Item struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Barcode       string `json:"barcode"`
	Brand         string `json:"brand"`
	ContainerSize string `json:"container_size"`
	ImageURL      string `json:"image_url"`
}
