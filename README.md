# FlareBox

A complete email management system with web dashboard, webhook API, and intelligent email generator. Designed to receive emails from Cloudflare Email Workers, manage domains, and generate unique temporary email addresses with support for Indonesian and Western names.

<video src="https://github.com/user-attachments/assets/0edece5e-02ec-4a43-ac2b-24920cab553e" autoplay loop muted playsinline></video>

## ✨ Features

### 🎨 Web Dashboard
- **Modern UI** - Clean, responsive interface built with HTMX
- **Email Inbox** - View and manage received emails by domain/username
- **Domain Management** - Add/remove active email domains
- **Email Generator** - Generate unique temporary email addresses
- **Settings Control** - Configure cleanup intervals and API keys
- **User Management** - Change password and username

### 🔐 Authentication & Security
- **Session-based auth** - Secure cookie-based authentication for web UI
- **API key auth** - Dual API keys (webhook + client) for API access
- **Auto-generated keys** - API keys generated on first run
- **Password protection** - Bcrypt-hashed passwords
- **Default credentials** - `admin / 123456` (change after first login!)

### 📧 Email Management
- **Webhook API** - Receives emails from Cloudflare Email Workers
- **JSON Storage** - Emails organized by domain and username
- **Pagination** - Retrieve emails with page/limit parameters
- **Auto-cleanup** - Configurable cleanup of inactive emails
- **Multi-domain** - Support for multiple email domains

### 🎲 Email Generator
- **Unique addresses** - Generates guaranteed unique email addresses
- **Smart patterns** - 7 different realistic username patterns
- **Bilingual names** - Mix of Indonesian (25+) and Western (30+) names
- **Random domains** - Selects from active domains
- **API access** - Generate emails programmatically

### ⚙️ Configuration
- **Database-driven** - Settings stored in SQLite database
- **UI configurable** - Change settings without restart
- **Minimal .env** - Only `PORT` needed in environment
- **Auto-migration** - Database schema managed automatically

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- (Optional) Docker & Docker Compose

### Installation

1. **Clone the repository:**
```bash
git clone <repository-url>
cd flarebox
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Run the server:**
```bash
go run main.go
```

The server will start on port **2525** by default.

```
🚀 FlareBox
📍 Server running on http://localhost:2525
🔑 Default login: admin / 123456 (change after first login)
```

4. **Access the dashboard:**

Open your browser and go to: **http://localhost:2525/dashboard**

Login with:
- **Username:** `admin`
- **Password:** `123456`

⚠️ **Important:** Change the default password after first login!

## 📝 Configuration

Create a `.env` file (optional) to customize the port:

```env
# Server port (default: 2525)
PORT=2525
```

That's it! All other settings (API keys, cleanup intervals, domains) are managed through the web dashboard or stored in the database.

### Database

The application uses SQLite (`settings.db`) to store:
- User credentials
- API keys (webhook & client)
- Active domains
- Cleanup settings

The database is created automatically on first run with secure default values.

## 🎯 Web Dashboard Features

### Dashboard Home
- View all email addresses that have received emails
- Click to see emails for specific address
- Real-time email display

### Domain Management
- Add new email domains
- Remove domains
- View active domains list

### Email Generator
- Generate random unique email addresses
- Uses active domains
- Mix of Indonesian and Western names
- Guaranteed uniqueness check

### Settings
- **API Keys Management**
  - View current API keys
  - Regenerate webhook key
  - Regenerate client key
  
- **Cleanup Configuration**
  - Set cleanup interval (minutes)
  - Set inactive threshold (hours)
  
- **Account Management**
  - Change password
  - Change username

## 🔌 API Endpoints

### Webhook Endpoint (Receive Emails)

**Authentication:** Webhook API Key (via `X-API-Key` header or `api_key` query parameter)

```bash
POST /api/webhook
X-API-Key: your-webhook-api-key
Content-Type: application/json

{
  "to": "testuser@example.com",
  "from": "sender@company.com",
  "subject": "Your OTP Code",
  "body": "Your verification code is: 123456",
  "html_body": "<p>Your verification code is: <strong>123456</strong></p>",
  "has_attachments": false,
  "headers": {
    "date": "2026-07-13T10:30:00Z"
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

### Get Emails

**Authentication:** Client API Key (via `X-API-Key` header or `api_key` query parameter)

```bash
GET /api/email/{domain}/{username}/?page=1&limit=20
X-API-Key: your-client-api-key
```

**Example:**
```bash
curl -H "X-API-Key: your-client-api-key" \
  "http://localhost:2525/api/email/example.com/testuser/?page=1&limit=20"
```

**Response:**
```json
{
  "emails": [
    {
      "id": 123,
      "subject": "Welcome to our service",
      "sender": "noreply@company.com",
      "date": "2026-07-13T10:30:00Z",
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

### Random Email Generator

**Authentication:** Client API Key

#### Get Random Domains
```bash
GET /api/random-domains/?limit=10
X-API-Key: your-client-api-key
```

**Response:**
```json
{
  "domains": ["example.com", "test.com", "demo.com"],
  "total": 3
}
```

#### Generate Random Email
```bash
GET /api/random-email
X-API-Key: your-client-api-key
```

**Response:**
```json
{
  "email": "budi.santoso@example.com",
  "username": "budi.santoso",
  "domain": "example.com"
}
```

**Email Generation Features:**
- **7 different patterns:** firstname.lastname, firstname_lastname, firstname123, etc.
- **55 first names:** Mix of Indonesian (budi, siti, andi...) and Western (john, sarah, michael...)
- **45 last names:** Mix of Indonesian (santoso, wijaya...) and Western (smith, johnson...)
- **Uniqueness guaranteed:** Checks existing emails and retries up to 10 times
- **Fallback:** Adds timestamp if all attempts fail

### Health Check

```bash
GET /health
```

**Response:** `OK` (HTTP 200)

## ☁️ Cloudflare Email Worker Setup

To forward emails from Cloudflare to your webhook server with properly parsed email content, use the worker code provided in `cloudflare-worker.js`.

### Quick Setup

1. **Copy the worker code** from `cloudflare-worker.js` in this repository
2. **Update the configuration** at the top of the file:
   ```javascript
   const webhookUrl = "https://your-server.com/api/webhook";  // Your server URL
   const apiKey = "your-webhook-api-key-here";                 // From dashboard
   ```
3. **Deploy to Cloudflare Workers** via the dashboard
4. **Configure email routing** to use this worker

### What This Worker Does

The worker properly parses emails to extract:
- **Text body**: Clean text content from `text/plain` parts
- **HTML body**: HTML content from `text/html` parts
- **Subject, From, To**: Basic email metadata

It handles multipart/alternative emails correctly, so you'll receive clean content instead of raw email with headers and MIME boundaries.

## 📦 Storage Structure

Emails are stored in JSON files organized by domain and username:

```
data/
├── example.com/
│   ├── testuser.json
│   ├── admin.json
│   └── budi.santoso.json
└── anotherdomain.com/
    └── user.json
```

Each JSON file contains an array of email objects with auto-incrementing IDs.

## 🧹 Auto-Cleanup

The server automatically removes email files that haven't received new emails for the configured duration (configurable via dashboard, default: 6 hours).

- Cleanup checks run periodically (default: every 30 minutes)
- Files are deleted based on their last modification time
- Empty directories are also cleaned up automatically
- **All configurable via Settings page** in the dashboard

## 🐳 Docker Deployment

### Using Docker Compose (Recommended)

```bash
docker-compose up -d
```

The `docker-compose.yml` file is already configured with:
- Port 2525
- Volume mounts for data and database
- Health checks
- Auto-restart policy

### Manual Docker Build

```bash
# Build
docker build -t flarebox .

# Run
docker run -d \
  -p 2525:2525 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/settings.db:/app/settings.db \
  --name flarebox \
  flarebox
```

## 🔧 Building for Production

Build a binary:
```bash
go build -o flarebox
```

Run the binary:
```bash
./flarebox
```

For production deployment:
1. Use a reverse proxy (Nginx, Caddy, Traefik) with HTTPS
2. Change default password immediately
3. Regenerate API keys from the dashboard
4. Configure firewall to restrict access
5. Set up regular database backups

## 🔒 Security Considerations

- ✅ **Strong API keys** - Auto-generated secure random keys
- ✅ **HTTPS recommended** - Use reverse proxy in production
- ✅ **Password hashing** - Bcrypt with salt
- ✅ **Session security** - HTTP-only cookies with SameSite
- ✅ **Auto-cleanup** - Old emails deleted automatically
- ✅ **API key rotation** - Regenerate keys anytime from UI
- ⚠️ **Change defaults** - Update admin password on first login

## 📚 Documentation

- **Cloudflare Worker**: See `cloudflare-worker.js` for email forwarding setup

## 🛠️ Development

Format code:
```bash
go fmt ./...
```

Build:
```bash
go build -o email-server
```

## 📄 License

MIT License - feel free to use this project for any purpose.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 💡 Use Cases

- **Development & Testing** - Receive OTP codes and transactional emails during development
- **Temporary Emails** - Generate disposable email addresses for testing
- **Email Testing** - Test email sending functionality without real inboxes
- **CI/CD Pipelines** - Automated testing of email workflows
- **Demo Applications** - Show email functionality without real email infrastructure
