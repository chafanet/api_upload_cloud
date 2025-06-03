package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"upload_api_cloud/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultPartSize = 5 * 1024 * 1024 // 5MB
)

type UploadHandler struct {
	storageProvider storage.StorageProvider
	uploads         sync.Map // stores upload metadata
}

type uploadMetadata struct {
	UploadID   string
	Key        string
	Parts      []storage.PartInfo
	TotalParts int32
	mu         sync.Mutex // Add mutex for thread-safe part updates
}

func NewUploadHandler(provider storage.StorageProvider) *UploadHandler {
	return &UploadHandler{
		storageProvider: provider,
	}
}

// InitiateUpload handles the initial upload request
func (h *UploadHandler) InitiateUpload(c *gin.Context) {
	fileName := c.GetHeader("X-File-Name")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-File-Name header is required"})
		return
	}

	totalParts, err := strconv.ParseInt(c.GetHeader("X-Total-Parts"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid X-Total-Parts header"})
		return
	}

	// Generate a unique key for the file
	key := fmt.Sprintf("%s_%s", uuid.New().String(), fileName)

	// Initiate multipart upload with storage provider
	uploadID, err := h.storageProvider.InitiateMultipartUpload(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate upload"})
		return
	}

	// Store upload metadata
	metadata := &uploadMetadata{
		UploadID:   uploadID,
		Key:        key,
		Parts:      make([]storage.PartInfo, 0, totalParts),
		TotalParts: int32(totalParts),
	}
	h.uploads.Store(uploadID, metadata)

	c.JSON(http.StatusOK, gin.H{
		"upload_id": uploadID,
		"key":       key,
	})
}

// UploadPart handles individual part uploads
func (h *UploadHandler) UploadPart(c *gin.Context) {
	uploadID := c.GetHeader("X-Upload-ID")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Upload-ID header is required"})
		return
	}

	partNumber, err := strconv.ParseInt(c.GetHeader("X-Part-Number"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid X-Part-Number header"})
		return
	}

	// Retrieve upload metadata
	metadataObj, exists := h.uploads.Load(uploadID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload not found"})
		return
	}
	metadata := metadataObj.(*uploadMetadata)

	contentLength, err := strconv.ParseInt(c.GetHeader("Content-Length"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Content-Length header"})
		return
	}

	// Upload the part
	etag, err := h.storageProvider.UploadPart(c.Request.Context(), metadata.Key, uploadID, int32(partNumber), c.Request.Body, contentLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload part"})
		return
	}

	// Thread-safe update of parts
	metadata.mu.Lock()
	metadata.Parts = append(metadata.Parts, storage.PartInfo{
		PartNumber: int32(partNumber),
		ETag:       etag,
	})
	metadata.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"part_number": partNumber,
		"etag":        etag,
	})
}

// CompleteUpload finalizes the multipart upload
func (h *UploadHandler) CompleteUpload(c *gin.Context) {
	uploadID := c.GetHeader("X-Upload-ID")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Upload-ID header is required"})
		return
	}

	// Retrieve and remove upload metadata
	metadataObj, exists := h.uploads.LoadAndDelete(uploadID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload not found"})
		return
	}
	metadata := metadataObj.(*uploadMetadata)

	metadata.mu.Lock()
	if int32(len(metadata.Parts)) != metadata.TotalParts {
		metadata.mu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not all parts have been uploaded"})
		return
	}

	// Sort parts by part number for correct assembly
	sort.Slice(metadata.Parts, func(i, j int) bool {
		return metadata.Parts[i].PartNumber < metadata.Parts[j].PartNumber
	})
	parts := metadata.Parts
	metadata.mu.Unlock()

	// Complete the multipart upload
	err := h.storageProvider.CompleteMultipartUpload(c.Request.Context(), metadata.Key, uploadID, parts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete upload"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Upload completed successfully",
		"key":     metadata.Key,
	})
}
