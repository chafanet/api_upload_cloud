# File Upload API with Resume Capability

This is a Go-based API that handles large file uploads with resume capability, using cloud storage (currently AWS S3) with a provider-agnostic design.

## Features

- Multipart upload support for large files
- Resume capability for interrupted uploads
- Cloud storage provider abstraction (currently supports AWS S3)
- Clean architecture design for easy provider switching

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Set up environment variables:
```bash
export AWS_BUCKET_NAME=your-bucket-name
export AWS_REGION=your-aws-region
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export PORT=8080  # Optional, defaults to 8080
```

## API Endpoints

### 1. Initiate Upload

Starts a new multipart upload session.

```
POST /upload/initiate
Headers:
- X-File-Name: string (required)
- X-Total-Parts: number (required)

Response:
{
    "upload_id": "string",
    "key": "string"
}
```

### 2. Upload Part

Upload a part of the file.

```
POST /upload/part
Headers:
- X-Upload-ID: string (required, from initiate response)
- X-Part-Number: number (required, 1-based)

Body: Binary file chunk

Response:
{
    "part_number": number,
    "etag": "string"
}
```

### 3. Complete Upload

Finalize the multipart upload.

```
POST /upload/complete
Headers:
- X-Upload-ID: string (required, from initiate response)

Response:
{
    "message": "Upload completed successfully",
    "key": "string"
}
```

## Example Usage

Here's an example of how to use the API with cURL:

1. Initiate upload:
```bash
curl -X POST \
  -H "X-File-Name: large-file.zip" \
  -H "X-Total-Parts: 3" \
  http://localhost:8080/upload/initiate
```

2. Upload parts:
```bash
curl -X POST \
  -H "X-Upload-ID: {upload_id}" \
  -H "X-Part-Number: 1" \
  --data-binary "@part1" \
  http://localhost:8080/upload/part
```

3. Complete upload:
```bash
curl -X POST \
  -H "X-Upload-ID: {upload_id}" \
  http://localhost:8080/upload/complete
```

## Architecture

The API follows a clean architecture pattern with the following components:

- Storage Provider Interface: Abstracts cloud storage operations
- S3 Provider: Implements the storage interface for AWS S3
- Upload Handler: Manages the upload workflow and HTTP endpoints

To add support for a different cloud provider:

1. Implement the `StorageProvider` interface in `internal/storage/storage.go`
2. Update the main application to use the new provider

## Error Handling

The API returns appropriate HTTP status codes and error messages:

- 400: Bad Request (missing or invalid headers)
- 404: Not Found (upload ID not found)
- 500: Internal Server Error (storage provider errors)

## Development

To run the server locally:

```bash
go run cmd/api/main.go
```

## Dependencies

- github.com/aws/aws-sdk-go-v2: AWS SDK for Go
- github.com/gin-gonic/gin: HTTP web framework
- github.com/google/uuid: UUID generation 