package controllers

import (
	"errors"
	"net/http"

	"github.com/bhindley/Kitch/services"
	"github.com/gin-gonic/gin"
)

// CreateItemRequest is the expected JSON body for creating an item by barcode.
type CreateItemRequest struct {
	Barcode string `json:"barcode" binding:"required"`
}

// ItemController handles HTTP requests for items.
type ItemController struct {
	service *services.ItemService
}

// NewItemController creates a new ItemController.
func NewItemController(service *services.ItemService) *ItemController {
	return &ItemController{service: service}
}

// CreateItem handles POST /api/items.
// It looks up a product by barcode, checking the datastore first, then Open Food Facts.
func (ctrl *ItemController) CreateItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "barcode is required"})
		return
	}

	item, err := ctrl.service.CreateByBarcode(c.Request.Context(), req.Barcode)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found for barcode"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}
