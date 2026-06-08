package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UploadFile stores media via UploadService (MinIO or local fallback).
// Form fields: file (required), category (optional).
func (h *Handlers) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	category := c.PostForm("category")

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	sniff := make([]byte, 512)
	n, err := src.Read(sniff)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	reader := io.MultiReader(bytes.NewReader(sniff[:n]), src)
	url, mimeType, err := h.UploadService.Upload(c.Request.Context(), category, file.Filename, reader, file.Size, sniff[:n])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":       url,
		"file_url":  url,
		"mime_type": mimeType,
		"filename":  file.Filename,
		"backend":   string(h.UploadService.Backend()),
	})
}
