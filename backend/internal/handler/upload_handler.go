package handler

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"nexus-forum-backend/internal/media"

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

	fullData, err := media.ReadAll(io.MultiReader(bytes.NewReader(sniff[:n]), src), file.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	mimeType := http.DetectContentType(fullData)
	cleanMime := strings.Split(mimeType, ";")[0]
	if strings.HasPrefix(cleanMime, "image/") {
		if compressed, newMime, cerr := media.CompressImageIfNeeded(cleanMime, fullData); cerr == nil {
			fullData = compressed
			cleanMime = newMime
		}
	}
	reader := bytes.NewReader(fullData)
	storageURL, mimeType, err := h.UploadService.Upload(c.Request.Context(), category, file.Filename, reader, int64(len(fullData)), fullData[:min(512, len(fullData))])
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accessURL, err := h.UploadService.AccessibleURL(c.Request.Context(), storageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare media URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":         accessURL,
		"file_url":    accessURL,
		"storage_url": storageURL,
		"mime_type":   mimeType,
		"filename":    file.Filename,
		"backend":     string(h.UploadService.Backend()),
	})
}
