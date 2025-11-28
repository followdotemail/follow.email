# Follow Email Backend - System Design & Architecture

## Overview

The Follow Email Backend is a comprehensive email management and automation system built with Go (Gin framework) that provides OAuth-based authentication, email synchronization, AI-powered analysis, and automated follow-up capabilities.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend/     │    │   API Gateway   │    │   Backend       │
│   Client Apps   │◄──►│   (Gin Router)  │◄──►│   Services      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Middleware    │    │   External      │
                       │   - Auth        │    │   Services      │
                       │   - Rate Limit  │    │   - Gmail API   │
                       │   - Error       │    │   - Gemini AI   │
                       └─────────────────┘    │   - AWS S3      │
                                              │   - RabbitMQ    │
                                              └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │   Database      │
                                              │   (PostgreSQL)  │
                                              └─────────────────┘
```

## Core Components

### 1. Authentication & Authorization

**OAuth Service** (`pkg/oauth/`)
- Supports Google OAuth 2.0 and Microsoft OAuth
- Handles authorization code flow with PKCE
- Manages access and refresh tokens
- Provides user information retrieval

**Auth Handler** (`internal/handlers/auth.go`)
- Generates OAuth authorization URLs
- Handles OAuth callbacks
- Issues JWT tokens for authenticated sessions
- Manages token refresh and logout

**Auth Middleware** (`internal/middleware/auth.go`)
- Validates JWT tokens on protected routes
- Extracts user context from tokens
- Provides role-based access control

### 2. Database Layer

**Models** (`internal/models/`)
- `User`: User account information
- `OAuthToken`: OAuth access/refresh tokens
- `Email`: Email metadata and content
- `FollowUpTemplate`: Email template definitions
- `FollowUpSchedule`: Scheduled follow-up tasks
- `EmailAnalytics`: Email interaction metrics
- `GmailConsent`: Gmail OAuth consent and sync metadata
- `UserPrivacyMetadata`: GDPR/CCPA compliance data

**Database Service** (`internal/database/`)
- GORM-based ORM with PostgreSQL
- Automatic migrations and schema management
- Connection pooling and query optimization
- Slow query logging for performance monitoring

### 3. Email Management

**Email Sync Service** (`internal/services/email_sync.go`)
- Synchronizes emails from Gmail/Outlook APIs
- Handles incremental sync with pagination
- Manages email metadata extraction
- Processes attachments and content parsing

**Email Handler** (`internal/handlers/email.go`)
- REST API endpoints for email operations
- Sync status monitoring and reporting
- Email search and filtering capabilities
- Bulk operations support

### 4. AI Integration

**Gemini AI Service** (`pkg/ai/gemini.go`)
- Email content analysis and categorization
- Sentiment analysis and priority scoring
- Automated response generation
- Follow-up suggestion algorithms

**AI Analysis Endpoints**
- `/emails/analyze`: Analyze email content
- `/emails/generate-response`: Generate AI responses
- `/emails/:emailId/follow-up`: Schedule AI-powered follow-ups

### 5. Queue & Background Processing

**QStash Service** (`internal/queue/qstash.go`)
- Asynchronous email processing
- Follow-up scheduling and execution
- Batch operations and bulk processing
- Dead letter queue for failed operations

### 6. Storage & File Management

**S3 Storage Service** (`pkg/storage/s3.go`)
- Email attachment storage
- User data export/import
- Backup and archival operations
- CDN integration for file delivery

### 7. Privacy & Compliance

**Privacy Service** (`internal/services/privacy.go`)
- GDPR compliance features
- User consent management
- Data export and deletion
- Audit logging and tracking

## API Endpoints

### Public Endpoints
```
GET  /health                    # Health check
GET  /api/v1/ping              # Service status
```

### Authentication Endpoints
```
GET  /api/v1/auth/status        # Check user status (auto-creates users)
GET  /api/v1/auth/user          # Get user info (auto-creates users if new)
POST /api/v1/auth/logout        # Logout user
```

### Protected Endpoints
```
GET  /api/v1/profile            # User profile
POST /api/v1/emails/sync        # Sync emails
GET  /api/v1/emails/sync/status # Sync status
POST /api/v1/emails/analyze     # Analyze email
POST /api/v1/emails/generate-response # Generate AI response
POST /api/v1/emails/:id/follow-up     # Schedule follow-up
POST /api/v1/privacy/consent    # Record consent
POST /api/v1/privacy/export-request   # Request data export
```

## Data Flow

### 1. User Authentication Flow
```
1. Client requests OAuth URL → Auth Handler
2. User authorizes → OAuth Provider (Google/Microsoft)
3. Provider redirects → Callback Handler
4. Exchange code for tokens → OAuth Service
5. Generate JWT token → Auth Handler
6. Return JWT to client → Client stores token
```

### 2. Email Synchronization Flow
```
1. Client requests sync → Email Handler
2. Validate JWT token → Auth Middleware
3. Queue sync job → QStash
4. Background worker processes → Email Sync Service
5. Fetch emails from provider → Gmail/Outlook API
6. Store email metadata → Database
7. Store attachments → S3 Storage
8. Update sync status → Database
```

### 3. AI Analysis Flow
```
1. Client requests analysis → Email Handler
2. Retrieve email content → Database
3. Send to AI service → Gemini AI
4. Process AI response → AI Service
5. Store analysis results → Database
6. Return insights → Client
```

### 4. Follow-up Scheduling Flow
```
1. Client schedules follow-up → Email Handler
2. Create follow-up record → Database
3. Queue follow-up job → QStash
4. Background worker executes → Queue Service
5. Generate follow-up content → AI Service
6. Send follow-up email → Email Provider
7. Update analytics → Database
```

## Security Features

### Authentication & Authorization
- OAuth 2.0 with PKCE for secure authentication
- JWT tokens with configurable expiration
- Refresh token rotation for enhanced security
- Role-based access control (RBAC)

### API Security
- Rate limiting per endpoint and user
- Request validation and sanitization
- CORS configuration for cross-origin requests
- Comprehensive error handling without information leakage

### Data Protection
- Encrypted storage of sensitive tokens
- Secure communication with HTTPS
- Data anonymization for analytics
- Regular security audits and updates

## Performance Optimizations

### Database
- Connection pooling with configurable limits
- Query optimization with proper indexing
- Slow query monitoring and alerting
- Read replicas for analytics queries

### Caching
- Redis caching for frequently accessed data
- JWT token caching for validation
- API response caching with TTL
- CDN integration for static assets

### Background Processing
- Asynchronous job processing with QStash
- Worker pool management for scalability
- Job prioritization and retry mechanisms
- Dead letter queues for failed operations

## Monitoring & Observability

### Logging
- Structured logging with configurable levels
- Request/response logging with correlation IDs
- Error tracking and alerting
- Performance metrics collection

### Health Checks
- Database connectivity monitoring
- External service health checks
- Resource utilization tracking
- Automated failover mechanisms

## Deployment Architecture

### Container Orchestration
- Docker containerization for all services
- Kubernetes deployment with auto-scaling
- Service mesh for inter-service communication
- Blue-green deployment strategy

### Infrastructure
- Load balancers for high availability
- Database clustering with failover
- Message queue clustering
- CDN for global content delivery

## Configuration Management

### Environment Variables
```
# Server Configuration
PORT=8080
ENVIRONMENT=production

# Database
DATABASE_URL=postgresql://...

# OAuth Providers
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
REDIRECT_URL=https://api.example.com/api/v1/auth/callback

# External Services
GEMINI_API_KEY=...
AWS_ACCESS_KEY_ID=...
QSTASH_URL=...
QSTASH_TOKEN=...
QSTASH_CURRENT_SIGNING_KEY=...
QSTASH_NEXT_SIGNING_KEY=...
```

### Configuration Validation
- Startup configuration validation
- Environment-specific configurations
- Secret management with external providers
- Configuration hot-reloading support

## Error Handling & Recovery

### Error Categories
- **Client Errors (4xx)**: Bad requests, authentication failures
- **Server Errors (5xx)**: Internal errors, service unavailable
- **Integration Errors**: External API failures, timeout errors
- **Business Logic Errors**: Validation failures, constraint violations

### Recovery Mechanisms
- Automatic retry with exponential backoff
- Circuit breaker pattern for external services
- Graceful degradation for non-critical features
- Comprehensive error logging and alerting

## Future Enhancements

### Planned Features
- Multi-tenant architecture support
- Advanced email analytics and reporting
- Machine learning model training pipeline
- Real-time collaboration features
- Mobile SDK development

### Scalability Improvements
- Microservices architecture migration
- Event-driven architecture with event sourcing
- GraphQL API layer for flexible queries
- Advanced caching strategies with Redis Cluster

This system design provides a robust, scalable, and secure foundation for email management and automation, with clear separation of concerns and comprehensive error handling throughout the application stack.