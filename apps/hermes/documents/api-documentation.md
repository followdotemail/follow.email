# Follow Email Backend API Documentation

This document provides comprehensive information about all available API endpoints, authentication requirements, request/response formats, and usage examples.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

The API uses **Clerk Authentication** with JWT tokens. Most endpoints require authentication via the `Authorization` header.

### Authentication Header Format
```
Authorization: Bearer <jwt_token>
```

### Authentication Flow
1. User authenticates through Clerk (frontend)
2. Clerk provides JWT token
3. Include JWT token in API requests
4. Backend validates token via Clerk

---

## Public Endpoints

These endpoints do not require authentication.

### Health Check

**GET** `/health`

Returns the service health status.

**Response:**
```json
{
  "status": "healthy",
  "service": "follow-email-backend",
  "environment": "development"
}
```

### Ping

**GET** `/ping`

Simple ping endpoint for connectivity testing.

**Response:**
```json
{
  "message": "pong"
}
```

---

## Authentication Endpoints

All authentication endpoints require a valid Clerk JWT token.

### Get User Information

**GET** `/auth/user`

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "user": {
    "id": 1,
    "clerk_id": "user_2abc123def456",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "profile_image_url": "https://example.com/avatar.jpg",
    "gmail_consent": true,
    "gmail_consent_date": "2024-01-15T10:30:00Z",
    "gmail_sync_enabled": true,
    "last_gmail_sync_at": "2024-01-15T12:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T12:00:00Z"
  }
}
```

### Create User

**POST** `/auth/user`

Creates a user record in the database after Clerk authentication.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "user": {
    "id": 1,
    "clerk_id": "user_2abc123def456",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "profile_image_url": "https://example.com/avatar.jpg",
    "gmail_consent": false,
    "gmail_sync_enabled": false,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "message": "User created successfully"
}
```

### Update User

**PUT** `/auth/user`

Updates user information from Clerk data.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "user": {
    "id": 1,
    "clerk_id": "user_2abc123def456",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "profile_image_url": "https://example.com/avatar.jpg",
    "updated_at": "2024-01-15T12:00:00Z"
  },
  "message": "User updated successfully"
}
```

### Delete User

**DELETE** `/auth/user`

Deletes user account and all associated data.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "message": "User deleted successfully"
}
```

### Logout

**POST** `/auth/logout`

Logs out the user (handled by Clerk).

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

---

## Protected Endpoints

All protected endpoints require authentication and are rate-limited.

### User Profile

**GET** `/profile`

Returns basic user profile information.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "user_id": "user_2abc123def456",
  "email": "user@example.com"
}
```

---

## Gmail Integration Endpoints

All Gmail endpoints require authentication and are rate-limited.

### Initiate Gmail Consent

**POST** `/gmail/consent/initiate`

Initiates the Gmail OAuth consent flow and returns a properly formatted authorization URL.

**Important Notes:**
- The `auth_url` contains correctly formatted ampersand (`&`) characters for OAuth parameters
- Custom JSON encoding prevents HTML escaping of `&` characters (no `\u0026` encoding)
- The returned URL can be used directly for OAuth authorization without additional processing
- In development mode, uses a default user ID for testing

**Headers:**
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body (Optional):**
```json
{
  "return_url": "https://yourapp.com/gmail-connected"
}
```

**Response:**
```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=437264857039-0jjlrefj3p9v29dj2lepujhfjo0smvtu.apps.googleusercontent.com&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fapi%2Fv1%2Fgmail%2Fconsent%2Fcallback&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.readonly+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fgmail.modify+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.profile&state=dev-user-123_1759677815",
  "state": "dev-user-123_1759677815",
  "message": "Please visit the auth_url to grant Gmail access permissions"
}
```

**Response Fields:**
- `auth_url`: Google OAuth authorization URL with properly formatted `&` characters
- `state`: Security state parameter for OAuth flow validation
- `message`: User instructions for completing the OAuth flow

**Usage:**
1. Make POST request to initiate consent
2. Redirect user to the `auth_url` (URL contains proper `&` characters, not `\u0026`)
3. User grants permissions on Google's OAuth page
4. Google redirects to callback URL with authorization code
5. Backend processes the callback automatically and exchanges code for tokens

### Gmail OAuth Callback

**GET** `/gmail/consent/callback`

Handles the OAuth callback from Google (called automatically).

**Query Parameters:**
- `code` (required): Authorization code from Google
- `state` (required): State parameter for security
- `error` (optional): Error from Google if authorization failed

**Response (Success):**
```json
{
  "message": "Gmail access granted successfully",
  "user_email": "user@gmail.com",
  "consent_date": "2024-01-15T10:30:00Z",
  "sync_enabled": true
}
```

**Response (Error):**
```json
{
  "error": "OAuth authorization failed",
  "details": "access_denied"
}
```

### Get Gmail Connection Status

**GET** `/gmail/consent/status`

Returns the current Gmail connection and consent status.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "connected": true,
  "consent_given": true,
  "consent_date": "2024-01-15T10:30:00Z",
  "sync_enabled": true,
  "last_sync_at": "2024-01-15T12:00:00Z",
  "email_address": "user@gmail.com",
  "connection_status": "active"
}
```

**Connection Status Values:**
- `active`: Token is valid and connection is working
- `error`: Token exists but connection failed
- `not_connected`: No token exists

### Revoke Gmail Consent

**DELETE** `/gmail/consent/revoke`

Revokes Gmail access and deletes stored tokens.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "message": "Gmail access revoked successfully",
  "revoked_at": "2024-01-15T14:30:00Z"
}
```

---

## Privacy & Compliance Endpoints

All privacy endpoints require authentication and handle GDPR/CCPA compliance.

### Record Consent

**POST** `/privacy/consent`

Records user consent for data processing.

**Headers:**
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "consent_type": "gdpr",
  "granted": true,
  "metadata": {
    "purpose": "email_processing",
    "version": "1.0"
  }
}
```

**Consent Types:**
- `gdpr`: GDPR consent
- `ccpa`: CCPA consent
- `both`: Both GDPR and CCPA

**Response:**
```json
{
  "message": "Consent recorded successfully"
}
```

### Request Data Export

**POST** `/privacy/export-request`

Initiates a data export request for the user.

**Headers:**
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body (Optional):**
```json
{
  "reason": "GDPR data portability request",
  "metadata": {
    "format_preference": "json"
  }
}
```

**Response:**
```json
{
  "request_id": "export_123456",
  "status": "pending",
  "estimated_completion": "2024-01-16T10:30:00Z",
  "message": "Data export request submitted successfully"
}
```

### Request Data Deletion

**POST** `/privacy/delete-request`

Initiates a data deletion request for the user.

**Headers:**
```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

**Request Body (Optional):**
```json
{
  "reason": "GDPR right to be forgotten",
  "metadata": {
    "confirmation": true
  }
}
```

**Response:**
```json
{
  "request_id": "delete_123456",
  "status": "pending",
  "estimated_completion": "2024-01-16T10:30:00Z",
  "message": "Data deletion request submitted successfully"
}
```

### Get Data Export

**GET** `/privacy/export/:requestId`

Retrieves a completed data export.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `requestId`: The export request ID

**Response (Pending):**
```json
{
  "request_id": "export_123456",
  "status": "pending",
  "progress": 45,
  "estimated_completion": "2024-01-16T10:30:00Z"
}
```

**Response (Completed):**
```json
{
  "request_id": "export_123456",
  "status": "completed",
  "download_url": "https://s3.amazonaws.com/exports/user_data_123456.zip",
  "expires_at": "2024-01-22T10:30:00Z",
  "file_size": 1024000
}
```

### Get User Privacy Requests

**GET** `/privacy/requests`

Returns all privacy requests for the authenticated user.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "requests": [
    {
      "id": "export_123456",
      "type": "export",
      "status": "completed",
      "created_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T12:00:00Z"
    },
    {
      "id": "delete_789012",
      "type": "deletion",
      "status": "pending",
      "created_at": "2024-01-16T09:00:00Z",
      "estimated_completion": "2024-01-17T09:00:00Z"
    }
  ]
}
```

---

## Email Management Endpoints

All email endpoints require authentication and provide access to synchronized email data.

### Query Emails

**GET** `/emails`

Retrieves a paginated list of emails with optional filtering.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Query Parameters:**
- `page` (optional): Page number for pagination (default: 1)
- `limit` (optional): Number of emails per page (default: 20, max: 100)
- `from_email` (optional): Filter by sender email address
- `to_email` (optional): Filter by recipient email address
- `subject` (optional): Filter by subject (partial match)
- `date_from` (optional): Filter emails from this date (ISO 8601 format)
- `date_to` (optional): Filter emails to this date (ISO 8601 format)
- `is_read` (optional): Filter by read status (true/false)
- `has_attachments` (optional): Filter by attachment presence (true/false)

**Example Request:**
```
GET /api/v1/emails?page=1&limit=10&from_email=john@example.com&is_read=false
```

**Response:**
```json
{
  "emails": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "message_id": "<CABcdefg123@mail.gmail.com>",
      "thread_id": "thread_abc123",
      "subject": "Important Meeting Request",
      "from_email": "john@example.com",
      "from_name": "John Doe",
      "to_email": "user@example.com",
      "to_name": "User Name",
      "received_at": "2024-01-15T10:30:00Z",
      "is_read": false,
      "has_attachments": true,
      "labels": ["INBOX", "IMPORTANT"],
      "ai_summary": "Meeting request for project discussion",
      "ai_sentiment": "neutral",
      "ai_priority": "high",
      "ai_category": "business",
      "created_at": "2024-01-15T10:31:00Z",
      "updated_at": "2024-01-15T10:31:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "total_pages": 5,
    "total_count": 87,
    "page_size": 20,
    "has_next": true,
    "has_previous": false
  }
}
```

### Get Email by ID

**GET** `/emails/:id`

Retrieves detailed information for a specific email.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id` (required): Email UUID

**Response:**
```json
{
  "email": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "user_2abc123def456",
    "message_id": "<CABcdefg123@mail.gmail.com>",
    "thread_id": "thread_abc123",
    "subject": "Important Meeting Request",
    "from_email": "john@example.com",
    "from_name": "John Doe",
    "to_email": "user@example.com",
    "to_name": "User Name",
    "cc_emails": ["cc@example.com"],
    "bcc_emails": [],
    "received_at": "2024-01-15T10:30:00Z",
    "is_read": false,
    "is_starred": false,
    "is_important": true,
    "has_attachments": true,
    "labels": ["INBOX", "IMPORTANT"],
    "s3_key": "emails/user_2abc123def456/550e8400-e29b-41d4-a716-446655440000.json",
    "ai_processed": true,
    "ai_summary": "Meeting request for project discussion next week",
    "ai_sentiment": "neutral",
    "ai_priority": "high",
    "ai_category": "business",
    "ai_key_points": ["Meeting request", "Project discussion", "Next week"],
    "ai_suggested_actions": ["Check calendar", "Confirm availability"],
    "created_at": "2024-01-15T10:31:00Z",
    "updated_at": "2024-01-15T10:31:00Z"
  }
}
```

### Get Email Content

**GET** `/emails/:id/content`

Retrieves the full email body and attachment information from S3 storage.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id` (required): Email UUID

**Response:**
```json
{
  "email_id": "550e8400-e29b-41d4-a716-446655440000",
  "body": {
    "text": "Hi there,\n\nI would like to schedule a meeting to discuss the project...",
    "html": "<p>Hi there,</p><p>I would like to schedule a meeting to discuss the project...</p>"
  },
  "attachments": [
    {
      "filename": "project_proposal.pdf",
      "content_type": "application/pdf",
      "size": 1024000,
      "s3_key": "attachments/user_2abc123def456/550e8400-e29b-41d4-a716-446655440000/project_proposal.pdf"
    }
  ],
  "retrieved_at": "2024-01-15T14:30:00Z"
}
```

**Error Responses for Email Endpoints:**

### 400 Bad Request
```json
{
  "error": "Invalid query parameters",
  "details": "limit must be between 1 and 100"
}
```

### 404 Not Found
```json
{
  "error": "Email not found",
  "message": "Email with ID 550e8400-e29b-41d4-a716-446655440000 not found"
}
```

### 403 Forbidden
```json
{
  "error": "Access denied",
  "message": "You don't have permission to access this email"
}
```

---

## Error Responses

All endpoints may return the following error responses:

### 400 Bad Request
```json
{
  "error": "BAD_REQUEST",
  "message": "Invalid request",
  "details": "Specific validation error message"
}
```

### 401 Unauthorized
```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

### 403 Forbidden
```json
{
  "error": "FORBIDDEN",
  "message": "Access denied"
}
```

### 404 Not Found
```json
{
  "error": "NOT_FOUND",
  "message": "Resource not found"
}
```

### 429 Too Many Requests
```json
{
  "error": "RATE_LIMITED",
  "message": "Too many requests",
  "retry_after": 60
}
```

### 500 Internal Server Error
```json
{
  "error": "INTERNAL_ERROR",
  "message": "Internal server error"
}
```

---

## Rate Limiting

The API implements rate limiting on different endpoint groups:

- **Authentication endpoints**: 10 requests per minute
- **Gmail endpoints**: 5 requests per minute
- **Privacy endpoints**: 3 requests per minute
- **General protected endpoints**: 60 requests per minute

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 59
X-RateLimit-Reset: 1642248660
```

---

## Environment Variables

Required environment variables for API functionality:

### Authentication
- `CLERK_SECRET_KEY`: Clerk secret key for JWT validation
- `CLERK_PUBLISHABLE_KEY`: Clerk publishable key

### Gmail Integration
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret
- `BASE_URL`: Base URL for OAuth callbacks (e.g., http://localhost:8080)

### Database
- `DATABASE_URL`: PostgreSQL connection string

### Other Services
- `QSTASH_URL`: QStash API URL
- `QSTASH_TOKEN`: QStash authentication token
- `QSTASH_CURRENT_SIGNING_KEY`: Current webhook signing key
- `QSTASH_NEXT_SIGNING_KEY`: Next webhook signing key
- `AWS_ACCESS_KEY_ID`: AWS access key
- `AWS_SECRET_ACCESS_KEY`: AWS secret key
- `S3_BUCKET_NAME`: S3 bucket for file storage
- `GEMINI_API_KEY`: Google Gemini AI API key

---

## Usage Examples

### Complete Email Management Flow

```javascript
// 1. Query emails with filters
const emailsResponse = await fetch('/api/v1/emails?page=1&limit=10&is_read=false', {
  headers: {
    'Authorization': `Bearer ${jwtToken}`
  }
});

const { emails, pagination } = await emailsResponse.json();
console.log(`Found ${pagination.total_count} emails`);

// 2. Get specific email details
const emailId = emails[0].id;
const emailResponse = await fetch(`/api/v1/emails/${emailId}`, {
  headers: {
    'Authorization': `Bearer ${jwtToken}`
  }
});

const { email } = await emailResponse.json();
console.log('Email subject:', email.subject);

// 3. Get email content and attachments
const contentResponse = await fetch(`/api/v1/emails/${emailId}/content`, {
  headers: {
    'Authorization': `Bearer ${jwtToken}`
  }
});

const { body, attachments } = await contentResponse.json();
console.log('Email body:', body.text);
console.log('Attachments:', attachments.length);
```

### Complete Gmail Integration Flow

```javascript
// 1. Initiate Gmail consent
const initiateResponse = await fetch('/api/v1/gmail/consent/initiate', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${jwtToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    return_url: 'https://myapp.com/gmail-connected'
  })
});

const { auth_url } = await initiateResponse.json();

// 2. Redirect user to Google OAuth
window.location.href = auth_url;

// 3. After callback, check status
const statusResponse = await fetch('/api/v1/gmail/consent/status', {
  headers: {
    'Authorization': `Bearer ${jwtToken}`
  }
});

const status = await statusResponse.json();
console.log('Gmail connected:', status.connected);
```

### Privacy Compliance Flow

```javascript
// Record GDPR consent
await fetch('/api/v1/privacy/consent', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${jwtToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    consent_type: 'gdpr',
    granted: true,
    metadata: {
      purpose: 'email_processing',
      version: '1.0'
    }
  })
});

// Request data export
const exportResponse = await fetch('/api/v1/privacy/export-request', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${jwtToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    reason: 'GDPR data portability request'
  })
});

const { request_id } = await exportResponse.json();

// Check export status
const exportStatus = await fetch(`/api/v1/privacy/export/${request_id}`, {
  headers: {
    'Authorization': `Bearer ${jwtToken}`
  }
});
```

---

## Security Considerations

1. **JWT Token Security**: Store JWT tokens securely (httpOnly cookies recommended)
2. **HTTPS Only**: All API calls should use HTTPS in production
3. **Rate Limiting**: Respect rate limits to avoid being blocked
4. **Token Expiration**: Handle token expiration and refresh appropriately
5. **OAuth State**: Validate OAuth state parameters for security
6. **Data Privacy**: Follow GDPR/CCPA guidelines when handling user data

---

## Support

For API support and questions:
- Check server logs for detailed error information
- Ensure all required environment variables are set
- Verify JWT token validity with Clerk
- Test endpoints with proper authentication headers