package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type UploadClient struct {
	baseURL string
}

type InitiateResponse struct {
	UploadID string `json:"upload_id"`
	Key      string `json:"key"`
}

type UploadPartResponse struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

func NewUploadClient(baseURL string) *UploadClient {
	return &UploadClient{
		baseURL: baseURL,
	}
}

func (c *UploadClient) InitiateUpload(fileName string, totalParts int) (*InitiateResponse, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/upload/initiate", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-File-Name", fileName)
	req.Header.Set("X-Total-Parts", fmt.Sprintf("%d", totalParts))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Request: %s %s\n", req.Method, req.URL.String())
		fmt.Printf("Request Headers: %v\n", req.Header)

		return nil, fmt.Errorf("server returned error: %s, status: %d", string(body), resp.StatusCode)
	}

	var result InitiateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *UploadClient) UploadPart(uploadID string, partNumber int, data []byte) (*UploadPartResponse, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/upload/part", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Upload-ID", uploadID)
	req.Header.Set("X-Part-Number", fmt.Sprintf("%d", partNumber))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	// Print request details
	fmt.Printf("Uploading part %d for upload ID: %s\n", partNumber, uploadID)
	fmt.Printf("Request URL: %s\n", req.URL.String())
	fmt.Printf("Request Headers: %v\n", req.Header)
	fmt.Printf("Request Body Length: %d bytes\n", len(data))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Print response details
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Headers: %v\n", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response Body: %s\n", string(body))

	// Create a new reader with the response body for further processing
	resp.Body = io.NopCloser(bytes.NewReader(body))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned error: %s, status: %d", string(body), resp.StatusCode)
	}

	var result UploadPartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *UploadClient) CompleteUpload(uploadID string) error {
	req, err := http.NewRequest("POST", c.baseURL+"/upload/complete", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Upload-ID", uploadID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned error: %s, status: %d", string(body), resp.StatusCode)
	}

	return nil
}

func main() {
	// Set default values
	baseURL := "https://crzqgyd49x.us-east-1.awsapprunner.com"
	filePath := "document.pdf"

	// Handle command line arguments
	switch len(os.Args) {
	case 3: // Both baseURL and filename provided
		baseURL = os.Args[1]
		filePath = os.Args[2]
	case 2: // Only baseURL provided
		baseURL = os.Args[1]
	}

	// Create a new client with the determined base URL
	client := NewUploadClient(baseURL)

	// Open the file to upload
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}

	// Calculate number of parts (5MB per part)
	const partSize = 5 * 1024 * 1024 // 5MB
	totalParts := (int(fileInfo.Size()) + partSize - 1) / partSize
	if totalParts == 0 {
		totalParts = 1
	}

	// Step 1: Initiate upload
	log.Println("Initiating upload...")
	initResp, err := client.InitiateUpload(filepath.Base(filePath), totalParts)
	if err != nil {
		log.Fatalf("Failed to initiate upload: %v", err)
	}
	log.Printf("Upload initiated with ID: %s\n", initResp.UploadID)

	// Step 2: Upload parts
	for partNumber := 1; partNumber <= totalParts; partNumber++ {
		// Read part data
		partData := make([]byte, partSize)
		n, err := file.Read(partData)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to read file part: %v", err)
		}
		if n == 0 {
			break
		}

		// Trim buffer to actual data size
		partData = partData[:n]

		log.Printf("Uploading part %d of %d (size: %d bytes)...\n", partNumber, totalParts, len(partData))
		partResp, err := client.UploadPart(initResp.UploadID, partNumber, partData)
		if err != nil {
			log.Fatalf("Failed to upload part %d: %v", partNumber, err)
		}
		log.Printf("Part %d uploaded successfully, ETag: %s\n", partResp.PartNumber, partResp.ETag)
	}

	// Step 3: Complete upload
	log.Println("Completing upload...")
	if err := client.CompleteUpload(initResp.UploadID); err != nil {
		log.Fatalf("Failed to complete upload: %v", err)
	}
	log.Printf("Upload completed successfully! File key: %s\n", initResp.Key)
}
