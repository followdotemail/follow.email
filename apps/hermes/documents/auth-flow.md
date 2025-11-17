# Authentication Flow Documentation

## Overview

This document describes the complete authentication flow between the Frontend, Backend, and Clerk authentication service in the Follow Email Backend system.

## Architecture Components

### 1. **Frontend (HTML/JavaScript)**
- Clerk JavaScript SDK integration
- Social login UI components
- JWT token management
- API request handling

### 2. **Backend (Go/Gin)**
- Clerk Go SDK integration
- JWT token validation middleware
- Protected API endpoints
- User session management

### 3. **Clerk Authentication Service**
- OAuth provider integration (Google, GitHub, Facebook)
- JWT token generation and validation
- User identity management
- Session handling

## Authentication Flow Sequence

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant C as Clerk
    participant O as OAuth Provider
    participant B as Backend
    participant DB as Database

    %% Initial Setup
    U->>F: Opens application
    F->>C: Initialize Clerk SDK
    C-->>F: SDK ready with publishable key
    F->>U: Display login options

    %% Authentication Process (Registration & Sign-in)
    U->>F: Click social login (Google/GitHub/Facebook)
    F->>C: Trigger social sign-in/sign-up
    C->>O: Redirect to OAuth provider
    O->>U: Show consent screen
    U->>O: Grant permissions
    O->>C: Return authorization code
    C->>C: Exchange code for tokens
    
    alt New User (Registration)
        C->>C: Create new Clerk user account
        C->>C: Generate JWT session token
        C-->>F: Return new user + session
        F->>B: First API call triggers user creation
        B->>DB: Create user record in database
    else Existing User (Sign-in)
        C->>C: Authenticate existing user
        C->>C: Generate JWT session token
        C-->>F: Return existing user + session
    end
    
    F->>F: Store session in memory
    F->>U: Show authenticated UI

    %% API Request Flow (Stateless Authentication)
    U->>F: Click API test button
    F->>C: Call clerk.session.getToken()
    C-->>F: Return JWT token from memory
    F->>B: HTTP request with "Authorization: Bearer <JWT>" header
    B->>B: Extract JWT from "Authorization" header
    B->>C: Send token to Clerk servers for validation
    C-->>B: Return token claims (user_id, email, etc.)
    B->>B: Set user context for THIS REQUEST ONLY
    B->>DB: Execute database operation using user_id
    DB-->>B: Return user-specific data
    B-->>F: Return API response
    B->>B: Discard user context (stateless)
    F->>U: Display result
    
    Note over F,B: Token flows via Authorization header
    Note over B: Backend extracts token from header
    Note over B: No session storage - each request validated independently
```

## Backend Authentication Verification

### How Backend Verifies User Authentication

The backend uses a **stateless JWT-based authentication** system. Here's how it works:

1. **No Server-Side Sessions**: The backend doesn't maintain user sessions in memory or database
2. **JWT Token Validation**: Each API request must include a valid JWT token in the Authorization header
3. **Clerk Token Verification**: The backend validates tokens directly with Clerk's servers
4. **User Context Extraction**: Valid tokens provide user information (user_id, email, etc.)

### Token Extraction and Verification Process

The backend gets the JWT token from the **Authorization header** of incoming HTTP requests:

```go
// Step 1: Extract JWT token from Authorization header
func extractBearerToken(c *gin.Context) string {
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        return ""
    }
    
    // Expected format: "Bearer eyJhbGciOiJSUzI1NiIs..."
    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        return ""
    }
    
    return parts[1] // Return the actual JWT token
}

// Step 2: Backend middleware validates token with Clerk
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract JWT from Authorization header
        token := extractBearerToken(c)
        if token == "" {
            c.JSON(401, gin.H{"error": "Missing or invalid Authorization header"})
            return
        }
        
        // Validate with Clerk servers (not local validation)
        claims, err := clerk.VerifyToken(context.Background(), &clerk.VerifyTokenParams{
            Token: token,
        })
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid or expired token"})
            return
        }
        
        // Set user context for this request only
        c.Set("user_id", claims.Subject)
        c.Set("user_email", claims.Email)
        c.Next()
    }
}
```

### Where the Token Comes From

1. **Frontend gets token**: `const token = await clerk.session.getToken()`
2. **Frontend sends token**: Includes it in Authorization header
3. **Backend extracts token**: From `Authorization: Bearer <token>` header
4. **Backend verifies token**: Calls Clerk's servers to validate

### HTTP Request Example

```http
GET /api/v1/auth/user HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

### Authentication Flow Per Request

1. **Frontend**: Sends JWT token with each API call
2. **Backend Middleware**: Intercepts request and validates token
3. **Clerk Validation**: Backend calls Clerk to verify token authenticity
4. **User Context**: If valid, user info is available for that request
5. **Request Processing**: API handler can access user context
6. **Response**: No session state is maintained after response

### Benefits of Stateless Authentication

- **Scalability**: No server memory used for sessions
- **Horizontal Scaling**: Any server instance can handle any request
- **Security**: Tokens expire automatically, no session cleanup needed
- **Simplicity**: No session storage or synchronization required

## Detailed Flow Steps

### Phase 1: Initialization

1. **Frontend Loads**
   ```javascript
   // Clerk SDK auto-initializes with publishable key
   <script data-clerk-publishable-key="pk_test_..." src="clerk.browser.js">
   ```

2. **Clerk SDK Ready**
   ```javascript
   window.addEventListener('load', async () => {
       await window.Clerk.load();
       // Mount sign-in components
   });
   ```

### Phase 2: User Authentication (Registration & Sign-in)

#### Registration Flow (New Users)

1. **User Initiates Registration**
   - New user clicks social provider button (Google/GitHub/Facebook)
   - Frontend calls `clerk.openSignIn()` (handles both sign-in and sign-up)

2. **OAuth Registration Flow**
   - Clerk redirects to OAuth provider
   - User grants permissions for the first time
   - Provider returns user profile data to Clerk

3. **Clerk User Creation**
   - Clerk creates new user account with OAuth profile data
   - Generates unique Clerk user ID
   - Creates JWT session token with user claims
   - Returns new user object to frontend

4. **Backend User Creation**
   - Frontend makes first authenticated API call
   - Backend middleware validates JWT token
   - Backend creates user record in database using Clerk user ID
   - Links user preferences and data to Clerk identity

#### Sign-in Flow (Existing Users)

1. **User Initiates Sign-in**
   - Existing user clicks social provider button
   - Frontend calls `clerk.openSignIn()`

2. **OAuth Sign-in Flow**
   - Clerk redirects to OAuth provider
   - User authenticates with existing account
   - Provider confirms user identity to Clerk

3. **Session Restoration**
   - Clerk authenticates existing user account
   - Generates fresh JWT session token
   - Returns existing user object to frontend
   - No database user creation needed

### Phase 3: API Authentication

1. **Token Retrieval**
   ```javascript
   const token = await clerk.session.getToken();
   ```

2. **API Request**
   ```javascript
   fetch('/api/v1/auth/user', {
       headers: {
           'Authorization': `Bearer ${token}`,
           'Content-Type': 'application/json'
       }
   });
   ```

3. **Backend Validation**
   ```go
   // Middleware extracts and validates token
   func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
       return func(c *gin.Context) {
           token := extractBearerToken(c)
           claims, err := jwt.Verify(context.Background(), &jwt.VerifyParams{
               Token: token,
           })
           if err != nil {
               c.JSON(401, gin.H{"error": "Invalid token"})
               return
           }
           c.Set("user_id", claims.Subject)
           c.Next()
       }
   }
   ```

## API Endpoints

### Protected Endpoints (Require Authentication)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/auth/user` | Get user information (creates user if new) |
| POST | `/api/v1/auth/user` | Create user in database |
| PUT | `/api/v1/auth/user` | Update user information |
| DELETE | `/api/v1/auth/user` | Delete user account |
| POST | `/api/v1/auth/logout` | Logout user session |
| GET | `/api/v1/profile` | Get user profile |
| POST | `/api/v1/user/onboard` | Complete user onboarding (new users) |

### Registration vs Sign-in API Behavior

#### New User Registration Flow
- **First API Call**: Automatically creates user record in database
- **Response**: Includes `"is_new_user": true` flag
- **Frontend Action**: Can redirect to onboarding flow
- **Database**: Creates new user with Clerk ID as primary key

#### Existing User Sign-in Flow
- **API Calls**: Fetch existing user data from database
- **Response**: Includes `"is_new_user": false` flag
- **Frontend Action**: Direct to main application
- **Database**: Queries existing user record by Clerk ID

### Public Endpoints (No Authentication Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/ping` | Health check |
| GET | `/health` | Server health status |

## Token Structure

### JWT Token Claims
```json
{
  "iss": "https://clerk.accounts.dev",
  "sub": "user_2abc123def456",
  "aud": "your-app-id",
  "exp": 1640995200,
  "iat": 1640991600,
  "azp": "your-app-id"
}
```

### Backend Token Processing
```go
type JWTClaims struct {
    UserID   string `json:"sub"`
    Email    string `json:"email"`
    Provider string `json:"provider"`
}

// How backend extracts user info per request
func getUserFromContext(c *gin.Context) (string, bool) {
    userID, exists := c.Get("user_id")
    if !exists {
        return "", false
    }
    return userID.(string), true
}

// Example API handler with user creation for new users
func GetUserProfile(c *gin.Context) {
    userID, exists := getUserFromContext(c)
    if !exists {
        c.JSON(401, gin.H{"error": "User not authenticated"})
        return
    }
    
    userEmail, _ := getUserEmailFromContext(c)
    
    // Try to fetch existing user from database
    user, err := database.GetUserByClerkID(userID)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // New user - create database record
            newUser := &models.User{
                ClerkID: userID,
                Email:   userEmail,
                CreatedAt: time.Now(),
            }
            
            if err := database.CreateUser(newUser); err != nil {
                c.JSON(500, gin.H{"error": "Failed to create user"})
                return
            }
            
            c.JSON(200, gin.H{
                "user": newUser,
                "is_new_user": true,
            })
            return
        }
        
        c.JSON(500, gin.H{"error": "Failed to fetch user"})
        return
    }
    
    // Existing user
    c.JSON(200, gin.H{
        "user": user,
        "is_new_user": false,
    })
}
```

## Error Handling

### Authentication Error Scenarios

1. **Invalid Token**
   ```json
   {
     "error": "Invalid token",
     "code": "INVALID_TOKEN",
     "status": 401
   }
   ```

2. **Expired Token**
   ```json
   {
     "error": "Token expired",
     "code": "TOKEN_EXPIRED", 
     "status": 401
   }
   ```

3. **Missing Authorization Header**
   ```json
   {
     "error": "Authorization header required",
     "code": "MISSING_AUTH_HEADER",
     "status": 401
   }
   ```

### Registration-Specific Errors

4. **User Creation Failed**
   ```json
   {
     "error": "Failed to create user account",
     "code": "USER_CREATION_FAILED",
     "status": 500
   }
   ```

5. **Duplicate User**
   ```json
   {
     "error": "User already exists",
     "code": "USER_EXISTS",
     "status": 409
   }
   ```

### Sign-in Specific Errors

6. **User Not Found**
   ```json
   {
     "error": "User account not found",
     "code": "USER_NOT_FOUND",
     "status": 404
   }
   ```

### Common Authentication Errors

| Error Code | Description | Frontend Action |
|------------|-------------|------------------|
| 401 | Invalid/expired token | Redirect to login |
| 403 | Insufficient permissions | Show access denied |
| 429 | Rate limit exceeded | Show retry message |
| 500 | Server error | Show error message |

### Error Response Format
```json
{
  "error": "Invalid or expired token",
  "code": "AUTH_TOKEN_INVALID",
  "timestamp": "2025-01-18T08:30:00Z"
}
```

## Security Considerations

### Token Security
- JWT tokens are short-lived (1 hour default)
- Tokens are stored in memory, not localStorage
- HTTPS required for production
- CORS properly configured

### Backend Security
- Rate limiting on authentication endpoints
- Input validation and sanitization
- SQL injection prevention with GORM
- Secure headers middleware

## Configuration

### Environment Variables
```bash
# Clerk Configuration
CLERK_SECRET_KEY=sk_test_...
CLERK_PUBLISHABLE_KEY=pk_test_...

# Database
DATABASE_URL=postgres://...

# Server
PORT=8080
ENVIRONMENT=development
```

### Frontend Configuration
```javascript
// Embedded in HTML
const BACKEND_URL = 'http://localhost:8080';
const CLERK_PUBLISHABLE_KEY = 'pk_test_...';
```

## Testing the Flow

### Manual Testing Steps

1. **Start Backend Server**
   ```bash
   go run main.go
   ```

2. **Open Frontend**
   - Navigate to `http://localhost:8080`
   - Verify Clerk initialization

3. **Test Registration Flow (New Users)**
   - Use incognito/private browser window
   - Click social login button (Google/GitHub/Facebook)
   - Complete OAuth consent for first time
   - Verify user creation in database
   - Check for `"is_new_user": true` in API response

4. **Test Sign-in Flow (Existing Users)**
   - Use regular browser window with existing account
   - Click social login button
   - Complete OAuth authentication
   - Verify existing user data retrieval
   - Check for `"is_new_user": false` in API response

5. **Test API Calls**
   - Click "Get User Info" button
   - Check server logs for requests
   - Verify API responses
   - Test both new and existing user scenarios

### Expected Server Logs
```bash
# Successful authentication
[GIN] 2025/01/18 - 08:30:15 | 200 | GET "/api/v1/auth/user"

# Failed authentication
[GIN] 2025/01/18 - 08:30:20 | 401 | GET "/api/v1/auth/user"
```

## Troubleshooting

### Common Issues

1. **"Missing publishableKey" Error**
   - Verify Clerk script tag has correct key
   - Check key format and validity

2. **"Not authenticated" Error**
   - Ensure user completed login flow
   - Check if session exists: `clerk.session`

3. **401 Unauthorized Errors**
   - Verify JWT token is being sent
   - Check Clerk secret key configuration
   - Ensure middleware is applied to routes

4. **CORS Errors**
   - Verify CORS headers in backend
   - Check allowed origins configuration

### Registration Flow Issues

5. **New user not created in database**
   - Check database connection
   - Verify user model schema
   - Ensure Clerk ID is properly extracted
   - Check for database constraint violations

6. **Duplicate user creation attempts**
   - Implement proper error handling for existing users
   - Use database constraints on Clerk ID
   - Check for race conditions in concurrent requests

### Sign-in Flow Issues

7. **Existing user not found**
   - Verify Clerk ID consistency
   - Check database user records
   - Ensure proper user lookup logic

8. **User data inconsistency**
   - Sync user data between Clerk and database
   - Handle profile updates from OAuth providers
   - Implement data migration if needed

### Debug Commands

```javascript
// Check Clerk status
console.log('Clerk loaded:', !!window.Clerk);
console.log('User:', clerk.user);
console.log('Session:', clerk.session);

// Get token manually
const token = await clerk.session?.getToken();
console.log('Token:', token);
```

## Integration Points

### Database Integration
- User records created/updated on first login
- **No session data stored** in database (stateless JWT architecture)
- User preferences and data linked by Clerk user ID
- Each request independently fetches user data using JWT claims

### Session vs Stateless Comparison

| Aspect | Traditional Sessions | JWT Stateless (Current) |
|--------|---------------------|-------------------------|
| **Storage** | Server memory/database | No server storage |
| **Scalability** | Limited by session store | Unlimited horizontal scaling |
| **Security** | Server controls expiry | Token self-expires |
| **Performance** | Session lookup required | Direct token validation |
| **Complexity** | Session management needed | Simpler architecture |
| **User State** | Maintained between requests | Reconstructed per request |

### Email Service Integration
- OAuth tokens stored for email access
- User consent tracked for privacy compliance
- Email sync triggered post-authentication

## Registration vs Sign-in Flow Summary

### Key Differences

| Aspect | Registration (New Users) | Sign-in (Existing Users) |
|--------|-------------------------|-------------------------|
| **Clerk Action** | Creates new user account | Authenticates existing account |
| **Database Action** | Creates new user record | Queries existing user record |
| **API Response** | `"is_new_user": true` | `"is_new_user": false` |
| **Frontend Flow** | Redirect to onboarding | Direct to main app |
| **First API Call** | Triggers user creation | Fetches existing data |
| **Error Handling** | Handle creation failures | Handle lookup failures |

### Benefits of Unified Flow

1. **Seamless User Experience**
   - Single login button handles both scenarios
   - No separate registration/login forms needed
   - OAuth providers handle user verification

2. **Simplified Frontend Logic**
   - Same authentication flow for all users
   - Backend determines new vs existing users
   - Frontend responds based on `is_new_user` flag

3. **Robust Backend Handling**
   - Automatic user creation for new users
   - Consistent JWT validation for all requests
   - Proper error handling for both scenarios

### Future Enhancements
- Multi-factor authentication (MFA)
- Role-based access control (RBAC)
- Session management dashboard
- Advanced security monitoring
- User onboarding wizard for new registrations
- Profile completion tracking
- Welcome email automation

---

*Last updated: January 18, 2025*
*Version: 1.0*