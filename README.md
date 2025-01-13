# Video Processing API

A Go-based REST API for video processing operations including upload, trimming, merging, and sharing functionality.

## Requirements

- Go 1.23
- FFmpeg
- SQLite3
- Docker (optional)

## Features
As per assignment requirements:
- Authenticated API calls using Bearer token
- Video upload with configurable size (25MB) and duration (5-25 secs) limits
- Video trimming functionality
- Video merging capability
- Share links with time-based expiry
- SQLite as database
- API documentation via Swagger

## Setup and Installation

### System Dependencies (Ubuntu)
```bash
# Install FFmpeg
sudo apt update
sudo apt install ffmpeg

# Install SQLite3
sudo apt install sqlite3

# Install Go 1.23
# First remove any existing Go installation
sudo rm -rf /usr/local/go

# Download and install Go 1.23
wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz

# Add Go to PATH in ~/.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### Local Development

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd vidproc-go
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up environment:
   - Copy `.env.example` to `.env`
   - Modify values as needed
   - Default server will run at `http://localhost:8080`

4. Start the server:
   ```bash
   go run cmd/server/main.go
   ```

### Docker Deployment

1. Build and run using Docker Compose:
   ```bash
   docker-compose up --build
   ```

Environment variables for Docker are configured in `.env.docker`
The API will be available at `http://localhost:8080`

## Testing

The project includes unit tests and end-to-end tests. Run them using:

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/api
go test ./internal/storage
go test ./internal/video

# Run end-to-end tests
go test ./tests/e2e
```

For testing API endpoints manually, example requests are available in the Swagger documentation.

## API Documentation

Access the complete API documentation through:
- Swagger UI: `http://localhost:8080/api/swagger/`
- Swagger Spec: `http://localhost:8080/api/swagger.yaml`