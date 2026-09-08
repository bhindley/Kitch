package controllers

import (
	"errors"
	"net/http"

	"github.com/bhindley/Kitch/models"
	"github.com/bhindley/Kitch/services"
	"github.com/gin-gonic/gin"
)

// CreateItemRequest is the expected JSON body for creating an item.
type CreateItemRequest struct {
	Name          string `json:"name" binding:"required"`
	Barcode       string `json:"barcode"`
	Brand         string `json:"brand"`
	ContainerSize string `json:"container_size"`
	ImageURL      string `json:"image_url"`
}

// UpdateItemRequest is the expected JSON body for updating an item.
type UpdateItemRequest struct {
	Name          string `json:"name"`
	Barcode       string `json:"barcode"`
	Brand         string `json:"brand"`
	ContainerSize string `json:"container_size"`
	ImageURL      string `json:"image_url"`
}

// ItemController handles HTTP requests for items.
type ItemController struct {
	service *services.ItemService
}

// NewItemController creates a new ItemController.
func NewItemController(service *services.ItemService) *ItemController {
	return &ItemController{service: service}
}

// LookupBarcode handles GET /api/barcode/:barcode.
// It checks the datastore for existing items, then fetches product data from
// Open Food Facts. Nothing is persisted.
func (ctrl *ItemController) LookupBarcode(c *gin.Context) {
	barcode := c.Param("barcode")
	if barcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "barcode is required"})
		return
	}

	result, err := ctrl.service.LookupBarcode(c.Request.Context(), barcode)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found for barcode"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListItems handles GET /api/items.
// Supports an optional ?q= query parameter for searching by name, brand, or barcode.
func (ctrl *ItemController) ListItems(c *gin.Context) {
	query := c.Query("q")

	items, err := ctrl.service.List(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if items == nil {
		items = []models.Item{}
	}

	c.JSON(http.StatusOK, items)
}

// CreateItem handles POST /api/items.
// It accepts all item attributes in the body and persists a new item.
func (ctrl *ItemController) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	item, err := ctrl.service.Create(c.Request.Context(), models.Item{
		Name:          req.Name,
		Barcode:       req.Barcode,
		Brand:         req.Brand,
		ContainerSize: req.ContainerSize,
		ImageURL:      req.ImageURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// UpdateItem handles PUT /api/items/:id.
func (ctrl *ItemController) UpdateItem(c *gin.Context) {
	id := c.Param("id")

	var req UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updated, err := ctrl.service.Update(c.Request.Context(), id, models.Item{
		Name:          req.Name,
		Barcode:       req.Barcode,
		Brand:         req.Brand,
		ContainerSize: req.ContainerSize,
		ImageURL:      req.ImageURL,
	})
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteItem handles DELETE /api/items/:id.
func (ctrl *ItemController) DeleteItem(c *gin.Context) {
	id := c.Param("id")

	if err := ctrl.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
