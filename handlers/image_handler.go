package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"pesxchange-backend/database"
	"pesxchange-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	supabase "github.com/supabase-community/supabase-go"
)

const (
	maxFileSize         = 5 * 1024 * 1024  // 5MB per image
	maxBase64Size       = 7000000           // ~5MB base64 encoded
	maxImagesPerUpload  = 5                 // Maximum images per request
	maxImageDimension   = 8192              // Maximum width/height in pixels
	bucketName          = "item-images"     // Storage bucket name
)

type ImageHandler struct{}

func NewImageHandler() *ImageHandler {
	return &ImageHandler{}
}

// UploadImage handles image upload to Supabase Storage with comprehensive security validations
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

	// SECURITY: Enforce maximum images per upload
	if len(files) > maxImagesPerUpload {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Maximum %d images allowed per upload", maxImagesPerUpload),
		})
	}

	var uploadedURLs []string
	var rejectedFiles []string
	// Use storage client (with service key) for uploads
	storageClient := database.GetStorageClient()

	for _, file := range files {
		// SECURITY: Validate file size
		if file.Size > maxFileSize {
			rejectedFiles = append(rejectedFiles, fmt.Sprintf("%s (exceeds 5MB limit)", file.Filename))
			continue
		}

		// Open file for validation
		src, err := file.Open()
		if err != nil {
			rejectedFiles = append(rejectedFiles, fmt.Sprintf("%s (failed to open)", file.Filename))
			continue
		}

		// SECURITY: Validate file type using magic bytes
		contentType, ext, err := validateImageFile(src)
		if err != nil {
			src.Close()
			rejectedFiles = append(rejectedFiles, fmt.Sprintf("%s (invalid image type: %s)", file.Filename, err.Error()))
			continue
		}

		// Read file content with size limit
		buf := new(bytes.Buffer)
		// Use LimitReader to prevent memory exhaustion
		limitedReader := io.LimitReader(src, maxFileSize+1)
		written, err := buf.ReadFrom(limitedReader)
		src.Close()

		if err != nil {
			rejectedFiles = append(rejectedFiles, fmt.Sprintf("%s (failed to read)", file.Filename))
			continue
		}

		// Double-check size after reading
		if written > maxFileSize {
			rejectedFiles = append(rejectedFiles, fmt.Sprintf("%s (file too large)", file.Filename))
			continue
		}

		// Generate unique filename with validated extension
		filename := fmt.Sprintf("%s_%d%s", 
			uuid.New().String(), 
			time.Now().Unix(), 
			ext)

		// Upload to Supabase Storage with proper content-type
		err = uploadToSupabase(storageClient, bucketName, filename, buf.Bytes(), contentType)
		
		if err != nil {
			rejectedFiles = append(rejectedFiles, fmt.Sprintf("%s (upload failed: %s)", file.Filename, err.Error()))
			continue
		}

		// Get public URL
		publicURL := storageClient.Storage.GetPublicUrl(bucketName, filename)
		uploadedURLs = append(uploadedURLs, publicURL.SignedURL)
	}

	// Return appropriate response
	if len(uploadedURLs) == 0 {
		errMsg := "Failed to upload any images"
		if len(rejectedFiles) > 0 {
			errMsg = fmt.Sprintf("All images rejected: %s", strings.Join(rejectedFiles, ", "))
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   errMsg,
		})
	}

	// Success response with warnings if some files were rejected
	response := models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"urls": uploadedURLs,
		},
	}

	if len(rejectedFiles) > 0 {
		response.Message = fmt.Sprintf("Uploaded %d images, rejected %d: %s", 
			len(uploadedURLs), len(rejectedFiles), strings.Join(rejectedFiles, ", "))
	}

	return c.JSON(response)
}

// ConvertBase64ToStorage converts existing base64 images to Supabase Storage with security validations
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

	// SECURITY: Enforce maximum images per request
	if len(req.Images) > maxImagesPerUpload {
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Maximum %d images allowed per conversion", maxImagesPerUpload),
		})
	}

	var convertedURLs []string
	var rejectedImages []string
	// Use storage client (with service key) for uploads
	storageClient := database.GetStorageClient()

	for i, img := range req.Images {
		if !strings.HasPrefix(img, "data:image/") {
			// Already a URL, keep as is
			convertedURLs = append(convertedURLs, img)
			continue
		}

		// Parse base64 image
		parts := strings.Split(img, ",")
		if len(parts) != 2 {
			rejectedImages = append(rejectedImages, fmt.Sprintf("image %d (invalid format)", i))
			continue
		}

		// SECURITY: Check base64 string size before decoding
		if len(parts[1]) > maxBase64Size {
			rejectedImages = append(rejectedImages, fmt.Sprintf("image %d (exceeds size limit)", i))
			continue
		}

		// Decode base64
		imageData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			rejectedImages = append(rejectedImages, fmt.Sprintf("image %d (decode failed)", i))
			continue
		}

		// SECURITY: Validate decoded size
		if len(imageData) > maxFileSize {
			rejectedImages = append(rejectedImages, fmt.Sprintf("image %d (decoded size exceeds 5MB)", i))
			continue
		}

		// SECURITY: Validate image using magic bytes
		contentType := http.DetectContentType(imageData)
		ext, err := getExtensionFromContentType(contentType)
		if err != nil {
			rejectedImages = append(rejectedImages, fmt.Sprintf("image %d (invalid image type)", i))
			continue
		}

		// Generate filename with validated extension
		filename := fmt.Sprintf("%s_%d_%d%s", req.ItemID, time.Now().Unix(), i, ext)

		// Upload to Supabase Storage with proper content-type
		err = uploadToSupabase(storageClient, bucketName, filename, imageData, contentType)
		
		if err != nil {
			rejectedImages = append(rejectedImages, fmt.Sprintf("image %d (upload failed: %s)", i, err.Error()))
			continue
		}

		// Get public URL
		publicURL := storageClient.Storage.GetPublicUrl(bucketName, filename)
		convertedURLs = append(convertedURLs, publicURL.SignedURL)
	}

	// Return appropriate response
	if len(convertedURLs) == 0 {
		errMsg := "Failed to convert any images"
		if len(rejectedImages) > 0 {
			errMsg = fmt.Sprintf("All images rejected: %s", strings.Join(rejectedImages, ", "))
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.APIResponse{
			Success: false,
			Error:   errMsg,
		})
	}

	// Success response with warnings if some images were rejected
	response := models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"urls": convertedURLs,
		},
	}

	if len(rejectedImages) > 0 {
		response.Message = fmt.Sprintf("Converted %d images, rejected %d: %s",
			len(convertedURLs), len(rejectedImages), strings.Join(rejectedImages, ", "))
	}

	return c.JSON(response)
}

// validateImageFile performs comprehensive validation using magic bytes
// Returns content type, file extension, and error
func validateImageFile(file io.ReadSeeker) (string, string, error) {
	// Read first 512 bytes for magic byte detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", "", fmt.Errorf("failed to read file: %w", err)
	}

	// Reset file pointer to beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return "", "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Detect content type using magic bytes
	contentType := http.DetectContentType(buffer[:n])

	// Validate against allowed types
	ext, err := getExtensionFromContentType(contentType)
	if err != nil {
		return "", "", err
	}

	return contentType, ext, nil
}

// getExtensionFromContentType maps MIME types to file extensions
func getExtensionFromContentType(contentType string) (string, error) {
	validTypes := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		// Note: GIF removed for security - can contain executable code
	}

	ext, ok := validTypes[contentType]
	if !ok {
		return "", fmt.Errorf("unsupported image type: %s", contentType)
	}

	return ext, nil
}

// uploadToSupabase uploads a file to Supabase Storage with proper content-type
// This bypasses the SDK's UploadFile which doesn't set content-type correctly
func uploadToSupabase(client *supabase.Client, bucket, filename string, data []byte, contentType string) error {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file header with explicit content-type
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{contentType}
	
	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create form part: %w", err)
	}

	_, err = part.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make HTTP request directly
	supabaseURL := os.Getenv("SUPABASE_URL")
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, filename)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("SUPABASE_SERVICE_KEY")))
	req.Header.Set("apikey", os.Getenv("SUPABASE_SERVICE_KEY"))

	// Execute request
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
