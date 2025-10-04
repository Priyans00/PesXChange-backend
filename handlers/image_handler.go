package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pesxchange-backend/database"
	"pesxchange-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ImageHandler struct{}

func NewImageHandler() *ImageHandler {
	return &ImageHandler{}
}

// UploadImage handles image upload to Supabase Storage
func (h *ImageHandler) UploadImage(c *fiber.Ctx) error {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to parse multipart form",
		})
	}

	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "No images provided",
		})
	}

	var uploadedURLs []string
	client := database.GetClient()

	for _, file := range files {
		// Validate file type
		if !isValidImageType(file.Header.Get("Content-Type")) {
			continue
		}

		// Generate unique filename
		filename := fmt.Sprintf("%s_%d%s", 
			uuid.New().String(), 
			time.Now().Unix(), 
			filepath.Ext(file.Filename))

		// Open file
		src, err := file.Open()
		if err != nil {
			continue
		}
		defer src.Close()

		// Read file content
		buf := new(bytes.Buffer)
		buf.ReadFrom(src)

		// Upload to Supabase Storage
		_, err = client.Storage.UploadFile("item-images", filename, bytes.NewReader(buf.Bytes()))
		
		if err != nil {
			continue
		}

		// Get public URL
		publicURL := client.Storage.GetPublicUrl("item-images", filename)

		uploadedURLs = append(uploadedURLs, publicURL.SignedURL)
	}

	if len(uploadedURLs) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(models.APIResponse{
			Success: false,
			Error:   "Failed to upload any images",
		})
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"urls": uploadedURLs,
		},
	})
}

// ConvertBase64ToStorage converts existing base64 images to Supabase Storage
func (h *ImageHandler) ConvertBase64ToStorage(c *fiber.Ctx) error {
	var req struct {
		Images []string `json:"images"`
		ItemID string   `json:"item_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	var convertedURLs []string
	client := database.GetClient()

	for i, img := range req.Images {
		if !strings.HasPrefix(img, "data:image/") {
			// Already a URL, keep as is
			convertedURLs = append(convertedURLs, img)
			continue
		}

		// Parse base64 image
		parts := strings.Split(img, ",")
		if len(parts) != 2 {
			continue
		}

		// Decode base64
		imageData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}

		// Determine file extension
		var ext string
		if strings.Contains(parts[0], "image/jpeg") || strings.Contains(parts[0], "image/jpg") {
			ext = ".jpg"
		} else if strings.Contains(parts[0], "image/png") {
			ext = ".png"
		} else if strings.Contains(parts[0], "image/webp") {
			ext = ".webp"
		} else {
			ext = ".jpg"
		}

		// Generate filename
		filename := fmt.Sprintf("%s_%d_%d%s", req.ItemID, time.Now().Unix(), i, ext)

		// Upload to Supabase Storage
		_, err = client.Storage.UploadFile("item-images", filename, bytes.NewReader(imageData))
		
		if err != nil {
			continue
		}

		// Get public URL
		publicURL := client.Storage.GetPublicUrl("item-images", filename)

		convertedURLs = append(convertedURLs, publicURL.SignedURL)
	}

	return c.JSON(models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"urls": convertedURLs,
		},
	})
}

func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg", 
		"image/png",
		"image/webp",
		"image/gif",
	}
	
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}