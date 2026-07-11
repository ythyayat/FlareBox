# Simple Email Server

A lightweight email webhook server designed to receive emails from Cloudflare Email Workers and store them in JSON format. Perfect for receiving OTP codes and other transactional emails during development and testing.

## Features

- 🚀 **Webhook API** - Receives emails from Cloudflare Email Workers
- 🔐 **API Key Authentication** - Secure webhook endpoint
- 📦 **JSON Storage** - Emails organized by recipient address
- 📄 **Pagination** - Retrieve emails with page/limit parameters
- 🧹 **Auto-Cleanup** - Automatically removes inactive email files after 6 hours
- ⚡ **Fast & Lightweight** - Built with Go for high performance

## Quick Start

### Prerequisites

- Go 1.21 or higher
- A Cloudflare account with Email Routing enabled

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd simple-email-server
```

2. Install dependencies:
```bash
go mod download
```

3. Configure environment variables:
```bash
cp .env.example .env
# Edit .env and set your API_KEY
```

4. Run the server:
```bash
go run main.go
```

The server will start on port 8080 (or the port specified in your `.env` file).

## Configuration

Create a `.env` file in the project root with the following variables:

```env
# Server port
PORT=8080

# API key for webhook authentication (REQUIRED)
API_KEY=your-secret-api-key-here

# How often to check for inactive files (in minutes)
CLEANUP_INTERVAL_MINUTES=30

# How long before a file is considered inactive and deleted (in hours)
CLEANUP_INACTIVE_HOURS=6
```

## API Endpoints

### 1. Webhook Endpoint (Receive Emails)

**POST** `/api/webhook`

Receives email data from Cloudflare Email Worker.

**Authentication:** API Key required via header `X-API-Key` or query parameter `api_key`

**Request Body:**
```json
{
  "to": "testuser@example.com",
  "from": "sender@company.com",
  "subject": "Your OTP Code",
  "body": "Your verification code is: 123456",
  "html_body": "<p>Your verification code is: <strong>123456</strong></p>",
  "has_attachments": false,
  "headers": {
    "date": "2025-07-08T10:30:00Z"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Email received and stored",
  "email_id": 1
}
```

### 2. Get Emails

**GET** `/api/email/{domain}/{username}/`

Retrieves paginated emails for a specific email address.

**Parameters:**
- `domain` - Email domain (e.g., `example.com`)
- `username` - Email username (e.g., `testuser`)
- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)

**Example:**
```bash
curl -X GET "http://localhost:8080/api/email/example.com/testuser/?page=1&limit=20"
```

**Response:**
```json
{
  "emails": [
    {
      "id": 123,
      "subject": "Welcome to our service",
      "sender": "noreply@company.com",
      "date": "2025-07-08T10:30:00Z",
      "body": "Email content here...",
      "html_body": "<p>Email content here...</p>",
      "has_attachments": false
    }
  ],
  "total": 5,
  "page": 1,
  "limit": 20,
  "has_more": false
}
```

### 3. Health Check

**GET** `/health`

Returns server health status.

**Response:** `OK` (HTTP 200)

## Cloudflare Email Worker Setup

To forward emails from Cloudflare to your webhook server with properly parsed email content (no headers clutter), use the worker code provided in `cloudflare-worker.js`.

### Quick Setup

1. **Copy the worker code** from `cloudflare-worker.js` in this repository
2. **Update the configuration** at the top of the file:
   ```javascript
   const webhookUrl = "https://your-server.com/api/webhook";  // Your server URL
   const apiKey = "your-secret-api-key-here";                  // Match your .env file
   ```
3. **Deploy to Cloudflare Workers** via the dashboard
4. **Configure email routing** to use this worker

### What This Worker Does

The worker properly parses emails to extract:
- **Text body**: Clean text content from `text/plain` parts
- **HTML body**: HTML content from `text/html` parts
- **Subject, From, To**: Basic email metadata

It handles multipart/alternative emails correctly, so you'll receive clean content like:
```json
{
  "body": "kali ini",
  "html_body": "<div dir=\"ltr\">kali ini</div>",
  "subject": "baru"
}
```

Instead of the raw email with all headers and MIME boundaries.

### Alternative: Simple Version (Metadata Only)

If you only need to know that an email arrived (good for OTP notifications where you just need the subject):

```javascript
export default {
    async email(message, env, ctx) {
        const webhookUrl = "https://your-server.com/api/webhook";
        const apiKey = "your-secret-api-key-here";
        
        try {
            const emailData = {
                to: message.to,
                from: message.from,
                subject: message.headers.get('subject') || 'No Subject',
                body: `Email received from ${message.from}`,
                html_body: "",
                has_attachments: false
            };

            const response = await fetch(webhookUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': apiKey
                },
                body: JSON.stringify(emailData)
            });

            if (!response.ok) {
                throw new Error(`Webhook returned ${response.status}`);
            }

            console.log('Email forwarded successfully');
        } catch (error) {
            console.error('Error:', error.message);
        }
    }
}
```

## Storage Structure

Emails are stored in JSON files organized by recipient:

```
data/
├── example.com/
│   ├── testuser.json
│   └── admin.json
└── anotherdomain.com/
    └── user.json
```

Each JSON file contains an array of email objects with auto-incrementing IDs.

## Auto-Cleanup

The server automatically removes email files that haven't received new emails for the configured duration (default: 6 hours). This helps manage storage and ensures old test data doesn't accumulate.

- Cleanup checks run periodically (default: every 30 minutes)
- Files are deleted based on their last modification time
- Empty directories are also cleaned up automatically

## Building for Production

Build a binary:
```bash
go build -o email-server
```

Run the binary:
```bash
./email-server
```

## Docker Deployment (Optional)

Create a `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o email-server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/email-server .
EXPOSE 8080
CMD ["./email-server"]
```

Build and run:
```bash
docker build -t simple-email-server .
docker run -p 8080:8080 -e API_KEY=your-key simple-email-server
```

## Security Considerations

- Always use a strong, randomly generated API key
- Use HTTPS in production (consider using a reverse proxy like Nginx or Caddy)
- The cleanup feature ensures old emails are automatically deleted
- No email data is exposed without proper authentication

## Development

Run tests:
```bash
go test ./...
```

Format code:
```bash
go fmt ./...
```

## License

MIT License - feel free to use this project for any purpose.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
