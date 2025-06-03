package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	"upload_api_cloud/internal/handler"
	"upload_api_cloud/internal/middleware"
	"upload_api_cloud/internal/storage"
)

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Initialize storage provider
	bucket := os.Getenv("AWS_BUCKET_NAME")
	if bucket == "" {
		log.Fatal("AWS_BUCKET_NAME environment variable is required")
	}
	storageProvider := storage.NewS3Provider(s3Client, bucket)

	// Initialize upload handler
	uploadHandler := handler.NewUploadHandler(storageProvider)

	// Set up Gin router
	router := gin.Default()

	// Configure rate limiter (10 requests per second per IP, burst of 20)
	rateLimiter := middleware.NewRateLimiter(10, 20, 1*time.Hour)
	router.Use(rateLimiter.Middleware())

	// Configure port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure server for better concurrent handling
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		IdleTimeout:    120 * time.Second,
	}

	// Configure upload endpoints
	router.POST("/upload/initiate", uploadHandler.InitiateUpload)
	router.POST("/upload/part", uploadHandler.UploadPart)
	router.POST("/upload/complete", uploadHandler.CompleteUpload)

	// Add health check endpoint
	router.GET("/health-check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
