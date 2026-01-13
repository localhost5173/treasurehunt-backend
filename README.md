# TreasureHunt AI API

A Fiber-based REST API with authentication (email/password and Google OAuth) and OpenAI Vision API integration for image analysis.

## Project Status

✅ **Completed Features:**
- User authentication (email/password + JWT)
- Google OAuth integration
- MongoDB database integration
- OpenAI Vision API integration (updated to gpt-4o-mini)
- Image analysis with simplified boolean responses
- Comprehensive API testing suite
- Docker containerization support

🚧 **Current Stage:** Production-ready backend API with full authentication and image analysis capabilities

## Features

- **Authentication**
  - Email/Password signup and login
  - Google OAuth integration
  - JWT-based authentication
  - Email whitelist for controlled access (optional)
  
- **Image Analysis**
  - OpenAI Vision API integration (using gpt-4o-mini)
  - Analyze images with custom prompts
  - Returns simple true/false boolean responses

## Prerequisites

- Go 1.24.6 or higher
- MongoDB (via Docker Compose)
- OpenAI API key
- Google OAuth credentials (optional)

## Setup

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Configure environment variables**:
   - Update the `.env` file with your credentials
   - Required: `JWT_SECRET`, `OPENAI_API_KEY`
   - Optional: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
   - Optional: `ENABLE_EMAIL_WHITELIST` - Set to `true` to restrict signups to whitelisted emails, `false` (default) to allow anyone
   - Optional: `EMAIL_WHITELIST` - Comma-separated list of allowed emails (only used if `ENABLE_EMAIL_WHITELIST=true`)
     - Example: `EMAIL_WHITELIST=user1@example.com,user2@example.com`

3. **Start MongoDB**:
   ```bash
   docker-compose up -d
   ```

4. **Run the application**:
   ```bash
   go run main.go
   ```

The server will start on `http://localhost:8080`

## API Endpoints

### Public Endpoints

#### Health Check
```
GET /health
```

### Authentication Endpoints

#### Signup
```
POST /api/auth/signup
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe"
}

Response:
{
  "token": "jwt_token_here",
  "user": {
    "id": "user_id",
    "email": "user@example.com",
    "name": "John Doe",
    "createdAt": "2025-10-22T...",
    "updatedAt": "2025-10-22T..."
  }
}
```

#### Login
```
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}

Response:
{
  "token": "jwt_token_here",
  "user": { ... }
}
```

#### Google OAuth Login
```
GET /api/auth/google
```
Redirects to Google OAuth consent screen.

#### Google OAuth Callback
```
GET /api/auth/google/callback?code=<auth_code>
```
Handles OAuth callback and redirects to frontend with token.

### Protected Endpoints

All protected endpoints require an `Authorization` header:
```
Authorization: Bearer <jwt_token>
```

#### Get Current User
```
GET /api/auth/me
Authorization: Bearer <jwt_token>

Response:
{
  "id": "user_id",
  "email": "user@example.com",
  "name": "John Doe",
  ...
}
```

#### Analyze Image Contents
```
POST /api/getImageContents
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "imageUrl": "https://example.com/image.jpg",
  "prompt": "Is there a table in this image?"
}

OR with base64 image:

{
  "imageBase64": "data:image/jpeg;base64,/9j/4AAQ...",
  "prompt": "Is there a table in this image?"
}

Response:
{
  "result": true
}
```

## Project Structure

```
treasurehuntAI/
├── config/           # Configuration management
├── database/         # MongoDB connection
├── handlers/         # HTTP request handlers
│   ├── auth_handler.go
│   └── image_handler.go
├── middleware/       # HTTP middleware (JWT auth)
├── models/           # Data models
├── repository/       # Database operations
├── utils/            # Utility functions (JWT generation)
├── .env             # Environment variables
├── docker-compose.yml
├── go.mod
├── go.sum
└── main.go          # Application entry point
```

## OpenAI Vision API Usage

The `/api/getImageContents` endpoint uses OpenAI's gpt-4o-mini model to analyze images. It:

1. Accepts either an image URL or base64-encoded image
2. Takes a natural language prompt (e.g., "Is there a table in the image?")
3. Enhances the prompt to request a yes/no answer first
4. Returns a simple boolean `result` (true/false) for easy programmatic use

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `MONGO_USERNAME` | MongoDB username | Yes |
| `MONGO_PASSWORD` | MongoDB password | Yes |
| `DATABASE_NAME` | Database name | Yes |
| `SERVER_PORT` | Server port (default: 8080) | No |
| `JWT_SECRET` | Secret key for JWT signing | Yes |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | No |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret | No |
| `GOOGLE_REDIRECT_URL` | Google OAuth callback URL | No |
| `FRONTEND_URL` | Frontend URL for CORS | Yes |
| `OPENAI_API_KEY` | OpenAI API key | Yes |

## Security Notes

- Store JWT secret securely and use a strong random value in production
- Use HTTPS in production
- Validate and sanitize all user inputs
- Keep your OpenAI API key secure
- Review and restrict CORS settings for production

## Development

To run in development mode with auto-reload, you can use:
```bash
go install github.com/cosmtrek/air@latest
air
```

## Testing

### Automated Testing
Run the included test script to verify all endpoints:

```bash
./test_api.sh
```

This script tests:
- Health check endpoint
- User signup and authentication
- Protected route access
- Image analysis functionality
- Unauthorized access rejection

### Manual Testing
Example curl requests:

```bash
# Signup
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Analyze Image
curl -X POST http://localhost:8080/api/getImageContents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"imageUrl":"https://example.com/image.jpg","prompt":"Is there a table in this image?"}'
```

## License

MIT
