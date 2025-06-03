package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Provider struct {
	client *s3.Client
	bucket string
}

func NewS3Provider(client *s3.Client, bucket string) *S3Provider {
	return &S3Provider{
		client: client,
		bucket: bucket,
	}
}

func (p *S3Provider) InitiateMultipartUpload(ctx context.Context, key string) (string, error) {
	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	result, err := p.client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return "", err
	}

	return *result.UploadId, nil
}

func (p *S3Provider) UploadPart(ctx context.Context, key string, uploadID string, partNumber int32, reader io.Reader, contentLength int64) (string, error) {

	input := &s3.UploadPartInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(key),
		PartNumber:    aws.Int32(partNumber),
		UploadId:      aws.String(uploadID),
		Body:          reader,
		ContentLength: aws.Int64(contentLength),
	}
	// Configure request checksum calculation
	fmt.Printf("Uploading part with content length: %d bytes\n", contentLength)
	result, err := p.client.UploadPart(ctx, input)

	if err != nil {
		fmt.Printf("S3 Upload Error: %v\n", err)
		return "", err
	}
	// Print response details for debugging

	return *result.ETag, nil
}

func (p *S3Provider) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []PartInfo) error {
	var completedParts []types.CompletedPart
	for _, part := range parts {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       aws.String(part.ETag),
			PartNumber: aws.Int32(part.PartNumber),
		})
	}

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(p.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	fmt.Printf("S3 Complete Upload Request: %+v\n", input)

	_, err := p.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		fmt.Printf("S3 Complete Upload Error: %v\n", err)
		return err
	}

	fmt.Printf("S3 Complete Upload Response: %+v\n", err)
	return err
}

func (p *S3Provider) AbortMultipartUpload(ctx context.Context, key string, uploadID string) error {
	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(p.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	}

	_, err := p.client.AbortMultipartUpload(ctx, input)
	return err
}
