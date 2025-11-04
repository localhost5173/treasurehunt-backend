# API Testing Examples

## Setup

1. Start MongoDB:
```bash
docker-compose up -d
```

2. Run the application:
```bash
go run main.go
```

## Test Sequence

### 1. Health Check
```bash
curl http://localhost:8080/health
```

Expected Response:
```json
{"status":"ok"}
```

### 2. Signup (Create New User)
```bash
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

Expected Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "671234567890abcdef123456",
    "email": "test@example.com",
    "name": "Test User",
    "createdAt": "2025-10-22T10:30:00Z",
    "updatedAt": "2025-10-22T10:30:00Z"
  }
}
```

**Save the token for the next requests!**

### 3. Login (Existing User)
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### 4. Get Current User (Protected Route)
```bash
# Replace YOUR_JWT_TOKEN with the token from signup/login
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 5. Analyze Image with URL (Protected Route)
```bash
# Replace YOUR_JWT_TOKEN with the token from signup/login
curl -X POST http://localhost:8080/api/getImageContents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "imageUrl": "https://images.unsplash.com/photo-1555041469-a586c61ea9bc",
    "prompt": "Is there a table in this image?"
  }'
```

Expected Response:
```json
{
  "result": true,
  "answer": "Yes, there is a table visible in the image. The image shows a modern dining area with a sleek wooden table..."
}
```

### 6. Analyze Image with Base64 (Protected Route)
```bash
# Replace YOUR_JWT_TOKEN with the token from signup/login
curl -X POST http://localhost:8080/api/getImageContents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "imageBase64": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD...",
    "prompt": "Is there a person in this image?"
  }'
```

## More Prompt Examples

### Check for specific objects:
```json
{
  "imageUrl": "https://example.com/image.jpg",
  "prompt": "Is there a chair in this image?"
}
```

### Check for colors:
```json
{
  "imageUrl": "https://example.com/image.jpg",
  "prompt": "Is the table red?"
}
```

### Check for people:
```json
{
  "imageUrl": "https://example.com/image.jpg",
  "prompt": "Are there any people in this image?"
}
```

### Check for text:
```json
{
  "imageUrl": "https://example.com/image.jpg",
  "prompt": "Is there any text visible in this image?"
}
```

### Check for conditions:
```json
{
  "imageUrl": "https://example.com/image.jpg",
  "prompt": "Is the room well-lit?"
}
```

## Google OAuth Testing

1. Visit: `http://localhost:8080/api/auth/google`
2. Login with your Google account
3. You'll be redirected to the frontend URL with the token as a query parameter
4. Extract the token from the URL: `http://localhost:5173?token=YOUR_JWT_TOKEN`

## Error Cases

### Invalid Token
```bash
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer invalid_token"
```

Response:
```json
{"error":"Invalid or expired token"}
```

### Missing Authorization
```bash
curl -X POST http://localhost:8080/api/getImageContents \
  -H "Content-Type: application/json" \
  -d '{"imageUrl":"https://example.com/image.jpg","prompt":"Is there a table?"}'
```

Response:
```json
{"error":"Missing authorization header"}
```

### Invalid Email/Password
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "wrong@example.com",
    "password": "wrongpassword"
  }'
```

Response:
```json
{"error":"Invalid email or password"}
```

### Missing Required Fields
```bash
curl -X POST http://localhost:8080/api/getImageContents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"prompt":"Is there a table?"}'
```

Response:
```json
{"error":"Either imageUrl or imageBase64 is required"}
```

## Using with JavaScript/Frontend

```javascript
// Signup
const signupResponse = await fetch('http://localhost:8080/api/auth/signup', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    email: 'test@example.com',
    password: 'password123',
    name: 'Test User'
  })
});
const { token, user } = await signupResponse.json();

// Store token
localStorage.setItem('token', token);

// Use token for protected routes
const imageAnalysisResponse = await fetch('http://localhost:8080/api/getImageContents', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    imageUrl: 'https://example.com/image.jpg',
    prompt: 'Is there a table in this image?'
  })
});
const result = await imageAnalysisResponse.json();
console.log(result); // { result: true, answer: "Yes, there is a table..." }
```

## Notes

- JWT tokens expire after 24 hours
- Make sure MongoDB is running before starting the application
- Ensure your OpenAI API key is valid and has sufficient credits
- The image analysis uses GPT-4 Vision Preview model
- Large images may take longer to process
