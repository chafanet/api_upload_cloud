package storage

import (
	"context"
	"io"
)

// StorageProvider defines the interface that any cloud storage provider must implement
type StorageProvider interface {
	// UploadPart uploads a part of a file and returns the ETag
	UploadPart(ctx context.Context, key string, uploadID string, partNumber int32, reader io.Reader, contentLength int64) (string, error)

	// InitiateMultipartUpload starts a new multipart upload and returns the upload ID
	InitiateMultipartUpload(ctx context.Context, key string) (string, error)

	// CompleteMultipartUpload finalizes the multipart upload
	CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []PartInfo) error

	// AbortMultipartUpload aborts an in-progress multipart upload
	AbortMultipartUpload(ctx context.Context, key string, uploadID string) error
}

// PartInfo represents information about an uploaded part
type PartInfo struct {
	PartNumber int32
	ETag       string
}
