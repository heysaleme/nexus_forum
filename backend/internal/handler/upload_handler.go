package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadFile handles file and image uploads for chat attachments.
// Size limit: 10MB
// MIME whitelist: PNG, JPEG, GIF, PDF, TXT, ZIP
// It checks both file headers and the first 512 bytes with http.DetectContentType.
func (h *Handlers) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	// 1. Size check: max 10MB
	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 10MB limit"})
		return
	}

	// Open the uploaded file to detect its content type
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	// Read first 512 bytes for MIME detection
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	mimeType := http.DetectContentType(buffer[:n])

	// Whitelist check
	allowedTypes := map[string]bool{
		"image/png":       true,
		"image/jpeg":      true,
		"image/gif":       true,
		"application/pdf": true,
		"text/plain":      true,
		"application/zip": true,
		"application/x-zip-compressed": true,
	}

	cleanMime := strings.Split(mimeType, ";")[0]
	if !allowedTypes[cleanMime] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type: " + cleanMime})
		return
	}

	// 2. Ensure upload dir exists
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
		return
	}

	// Generate safe unique filename
	safeFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
	dstPath := filepath.Join(uploadDir, safeFilename)

	// Save file
	if err := c.SaveUploadedFile(file, dstPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// Return URL (relative to root)
	url := "/uploads/" + safeFilename
	c.JSON(http.StatusOK, gin.H{
		"url":       url,
		"mime_type": cleanMime,
		"filename":  file.Filename,
	})
}
