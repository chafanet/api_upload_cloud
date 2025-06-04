# Upload API Client Example

This is a Go client example that demonstrates how to use the Upload API for multipart file uploads. The client supports uploading files in chunks and handles the complete upload process including initiation, part uploads, and completion.

## Prerequisites

- Go 1.16 or higher
- A file to upload (default: `document.pdf` in the same directory as the program)

## Running the Example

The program accepts two optional command-line arguments:
1. `baseURL`: The base URL of the upload API server
2. `filename`: The path to the file you want to upload

### Usage Options

1. **Using all defaults**:
   ```bash
   go run main.go
   ```
   This will use:
   - Default API URL: https://crzqgyd49x.us-east-1.awsapprunner.com
   - Default file: document.pdf

2. **Custom API URL only**:
   ```bash
   go run main.go http://localhost:8080
   ```
   This will use:
   - Your specified API URL
   - Default file: document.pdf

3. **Custom API URL and custom file**:
   ```bash
   go run main.go http://localhost:8080 path/to/your/file
   ```
   This will use:
   - Your specified API URL
   - Your specified file path

## How it Works

1. The client first initiates the upload by calling the `/upload/initiate` endpoint
2. It then splits the file into 5MB chunks and uploads each part using `/upload/part`
3. Finally, it completes the upload by calling `/upload/complete`

The program will show progress information and any errors that occur during the upload process.

## Error Handling

The client includes comprehensive error handling and will display detailed information if something goes wrong during:
- File opening
- Upload initiation
- Part uploads
- Upload completion

If any error occurs, the program will exit with an appropriate error message.
