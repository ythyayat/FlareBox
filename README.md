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

To forward emails from Cloudflare to your webhook server, create an Email Worker with the following code:

```javascript
export default {
  async email(message, env, ctx) {
    const webhookUrl = "https://your-server.com/api/webhook";
    const apiKey = "your-secret-api-key-here";
    
    try {
      // Extract email data
      const emailData = {
        to: message.to,
        from: message.from,
        subject: message.headers.get('subject') || '',
        body: await streamToString(message.raw),
        html_body: await streamToString(message.raw), // Parse as needed
        has_attachments: false, // Detect based on your needs
        headers: {
          date: message.headers.get('date'),
          'message-id': message.headers.get('message-id'),
        }
      };

      // Send to webhook
      const response = await fetch(webhookUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': apiKey
        },
        body: JSON.stringify(emailData)
      });

      if (response.ok) {
        console.log(`Email forwarded successfully to webhook`);
      } else {
        console.error(`Failed to forward email: ${response.status} ${response.statusText}`);
      }
    } catch (error) {
      console.error(`Error processing email: ${error.message}`);
    }
  }
}

// Helper function to convert stream to string
async function streamToString(stream) {
  const chunks = [];
  const reader = stream.getReader();
  
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  
  const uint8Array = new Uint8Array(
    chunks.reduce((acc, chunk) => acc + chunk.length, 0)
  );
  
  let offset = 0;
  for (const chunk of chunks) {
    uint8Array.set(chunk, offset);
    offset += chunk.length;
  }
  
  return new TextDecoder().decode(uint8Array);
}
```

### Simplified Worker Example (Basic Text Only)

```javascript
export default {
  async email(message, env, ctx) {
    const webhookUrl = "https://your-server.com/api/webhook";
    const apiKey = "your-secret-api-key-here";
    
    // Create email data object
    const emailData = {
      to: message.to,
      from: message.from,
      subject: message.headers.get('subject') || 'No Subject',
      body: `Email from ${message.from} to ${message.to}`,
      html_body: "",
      has_attachments: false
    };

    // Send to webhook
    await fetch(webhookUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': apiKey
      },
      body: JSON.stringify(emailData)
    });
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
