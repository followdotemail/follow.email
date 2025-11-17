# FollowEmail Backend

A comprehensive email management and AI-powered analysis backend service built with Go, featuring intelligent email processing, automated follow-up scheduling, and seamless OAuth integrations.

## 🚀 Features

### Core Functionality
- **Email Management**: Sync and manage emails from multiple providers (Gmail, Outlook)
- **AI-Powered Analysis**: Intelligent email sentiment analysis and priority scoring using Google Gemini
- **Automated Follow-ups**: Smart follow-up scheduling and template management
- **OAuth Integration**: Secure authentication with Google and Microsoft OAuth
- **Privacy Compliance**: GDPR-compliant data export and deletion requests

### Technical Features
- **RESTful API**: Comprehensive REST API with OpenAPI/Swagger documentation
- **Authentication**: Clerk-based user authentication with JWT tokens
- **Real-time Processing**: Asynchronous email processing with QStash
- **Scalable Storage**: AWS S3 integration for file storage and exports
- **Database**: PostgreSQL with GORM ORM and automatic migrations
- **Caching**: Redis integration for performance optimization
- **Rate Limiting**: Built-in rate limiting and security middleware

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Load Balancer │    │   API Gateway   │
│   (React/Vue)   │◄──►│   (NGINX)       │◄──►│   (Optional)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
                       ┌──────────────────────────────┼───────────────────────────┐
                       │                              ▼                           │
                       │                     ┌─────────────────┐                  │
                       │                     │   Go Backend    │                  │
                       │                     │   (Gin Router)  │                  │
                       │                     └─────────────────┘                  │
                       │                              │                           │
                       │         ┌────────────────────┼────────────────────┐      │
                       │         ▼                    ▼                    ▼      │
                       │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
                       │  │ PostgreSQL  │    │   QStash    │    │   Redis     │   │
                       │  │ Database    │    │ Message     │    │   Cache     │   │
                       │  └─────────────┘    │ Queue       │    └─────────────┘   │
                       │                     └─────────────┘                      │
                       └──────────────────────────────────────────────────────────┘
                                                        │
                       ┌────────────────────────────────┼──────────────────────────┐
                       │                                ▼                          │
                       │                    External Services                      │
                       │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
                       │  │   Clerk     │  │   Google    │  │    AWS S3   │        │
                       │  │    Auth     │  │   APIs      │  │   Storage   │        │
                       │  └─────────────┘  └─────────────┘  └─────────────┘        │
                       └───────────────────────────────────────────────────────────┘
```

## 📋 Prerequisites

Before setting up the FollowEmail Backend, ensure you have the following installed:

### Required Software
- **Go**: Version 1.24.0 or higher
- **PostgreSQL**: Version 15 or higher
- **Redis**: Version 6 or higher (optional, for caching)
- **QStash**: Upstash QStash account (for message queuing)

### Development Tools
- **Git**: For version control
- **Make**: For build automation (optional)

### External Services
You'll need accounts and API keys for:
- **Clerk**: For user authentication ([clerk.com](https://clerk.com))
- **Google Cloud**: For Gmail OAuth and Gemini AI ([console.cloud.google.com](https://console.cloud.google.com))
- **AWS**: For S3 storage ([aws.amazon.com](https://aws.amazon.com))

## 🛠️ Installation & Setup

### Manual Setup

1. **Clone and setup:**
   ```bash
   git clone https://github.com/devplaygrounds/server-follow.email.git
   cd FollowEmailBackend
   cp .env.example .env
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Setup PostgreSQL database:**
   ```bash
   # Create database
   createdb followemail
   
   # Run migrations (automatic on first startup)
   ```

4. **Setup Redis (optional):**
   ```bash
   # Install and start Redis
   redis-server
   ```

5. **Setup QStash:**
   ```bash
   # Create a QStash account at https://upstash.com/
   # Get your QStash token and signing keys from the dashboard
   ```

6. **Configure environment variables** (see below)

7. **Run the application:**
   ```bash
   go run main.go
   ```

## ⚙️ Environment Configuration

Configure the following environment variables in your `.env` file:

### Server Configuration
```bash
PORT=8080                    # Server port
ENVIRONMENT=development      # Environment (development/staging/production)
LOG_LEVEL=info              # Logging level (debug/info/warn/error)
BASE_URL=http://localhost:8080  # Base URL for OAuth callbacks
```

### Database Configuration
```bash
DATABASE_URL=postgres://username:password@localhost:5432/followemail?sslmode=disable
```

### Authentication (Clerk)
```bash
CLERK_SECRET_KEY=sk_test_your-clerk-secret-key
CLERK_PUBLISHABLE_KEY=pk_test_your-clerk-publishable-key
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

### Google Services
```bash
# OAuth for Gmail integration
GOOGLE_CLIENT_ID=your-google-client-id-here
GOOGLE_CLIENT_SECRET=your-google-client-secret-here

# AI Analysis (Gemini)
GEMINI_API_KEY=your-gemini-api-key
```

### AWS S3 Storage
```bash
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-aws-access-key-id
AWS_SECRET_ACCESS_KEY=your-aws-secret-access-key
S3_BUCKET_NAME=follow-email-storage
```

### Message Queue
```bash
# QStash Configuration
QSTASH_URL=https://qstash.upstash.io
QSTASH_TOKEN=your_qstash_token_here
QSTASH_CURRENT_SIGNING_KEY=your_current_signing_key
QSTASH_NEXT_SIGNING_KEY=your_next_signing_key
```

### Security
```bash
ENCRYPTION_KEY=your-32-byte-encryption-key-change-this-in-production
```

## 🗄️ Database Setup

The application uses PostgreSQL with automatic migrations. The database schema includes:

### Core Tables
- `users` - User profiles and authentication data
- `external_accounts` - OAuth provider connections
- `emails` - Email data and metadata
- `followup_templates` - Email templates for follow-ups
- `followup_schedules` - Scheduled follow-up tasks

### Migration Process
Migrations run automatically on application startup. Manual migration files are located in `/migrations/`:

```bash
migrations/
├── 001_initial_schema.up.sql      # Initial database schema
├── 002_clerk_integration.up.sql   # Clerk authentication integration
├── 003_convert_to_uuid.up.sql     # UUID conversion for better scalability
└── *.down.sql                     # Rollback migrations
```

To manually run migrations:
```bash
# Using migrate tool (if installed)
migrate -path ./migrations -database "postgres://localhost/followemail?sslmode=disable" up
```

## 📚 API Documentation

### Swagger UI
Access the interactive API documentation at:
- **Local**: http://localhost:8080/swagger-ui/
- **Swagger JSON**: http://localhost:8080/swagger.yaml

### API Endpoints Overview

#### Authentication
- `GET /api/v1/auth/status` - Check user authentication status (auto-creates users)
- `GET /api/v1/auth/user` - Get user information (auto-creates users if new)
- `POST /api/v1/auth/refresh` - Refresh JWT token
- `POST /api/v1/auth/logout` - Logout user

#### Email Management
- `POST /api/v1/emails/sync` - Sync emails from provider
- `GET /api/v1/emails/status` - Get sync status
- `POST /api/v1/emails/analyze` - AI analysis of emails

#### Gmail Integration
- `POST /api/v1/gmail/consent/initiate` - Start Gmail OAuth (returns properly formatted auth URL)
- `GET /api/v1/gmail/consent/callback` - OAuth callback handler (automatic)
- `GET /api/v1/gmail/consent/status` - Check Gmail connection status
- `DELETE /api/v1/gmail/consent/revoke` - Revoke Gmail access

#### Privacy & Compliance
- `POST /api/v1/privacy/consent` - Record user consent
- `POST /api/v1/privacy/export-request` - Request data export
- `POST /api/v1/privacy/delete-request` - Request data deletion

#### Health & Monitoring
- `GET /api/v1/health` - Health check endpoint
- `GET /api/v1/ping` - Simple ping endpoint

### Authentication
All protected endpoints require a Bearer token in the Authorization header:
```bash
Authorization: Bearer <your-jwt-token>
```

## 🔨 Building for Production

### Building Go Binary for Linux

Before deploying to a Linux server, you need to build the Go binary for the target environment. This section covers building the application for Linux deployment.

#### Prerequisites for Building

- Go 1.21+ installed on your development machine
- Access to the project source code
- Internet connection for downloading dependencies

#### Build Commands

**For Linux x64 (most common):**
```bash
# Set environment variables for Linux build
export GOOS=linux
export GOARCH=amd64

# Build the binary
GOOS=linux GOARCH=amd64 go build -o follow_email .

# Verify the binary
file follow_email
# Output should show: follow_email: ELF 64-bit LSB executable, x86-64
```

**For Linux ARM64 (for ARM-based servers):**
```bash
# Set environment variables for ARM64 Linux
export GOOS=linux
export GOARCH=arm64

# Build the binary
GOOS=linux GOARCH=amd64 go build -o follow_email .
```

**Build with optimizations (recommended for production):**
```bash
# Build with size and performance optimizations
export GOOS=linux
export GOARCH=amd64

GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o follow_email .

# -w: Omit the DWARF symbol table
# -s: Omit the symbol table and debug information
# This reduces binary size significantly
```

#### Cross-Platform Build Script

Create a build script for easy deployment:

```bash
#!/bin/bash
# build.sh - Build script for production deployment

echo "Building FollowEmail Backend for Linux..."

# Clean previous builds
rm -f follow_email

# Set target environment
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# Build with optimizations
GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -X main.version=$(git describe --tags --always)" -o follow_email .

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "📦 Binary: follow_email"
    echo "📏 Size: $(du -h follow_email | cut -f1)"
    echo "🏗️  Architecture: $(file follow_email)"
else
    echo "❌ Build failed!"
    exit 1
fi
```

Make the script executable:
```bash
chmod +x build.sh
./build.sh
```

#### Verification

After building, verify your binary:
```bash
# Check binary details
file follow_email
ls -lh follow_email

# Test the binary (optional - requires dependencies)
./follow_email --version  # if version flag is implemented
```

**Important Notes:**
- Always build for the target architecture (usually `linux/amd64`)
- Use `CGO_ENABLED=0` for static linking if you encounter library issues
- The binary should be built on each deployment or stored in your CI/CD pipeline
- Test the binary on a similar Linux environment before production deployment

## 🚀 Production Deployment

### Linux Server Deployment (Recommended)

This section covers deploying the FollowEmail Backend on a cloud-based Linux server using systemctl, Go binary, and nginx.

**📋 Prerequisites:** Before starting deployment, ensure you have built the Linux binary following the [Building for Production](#-building-for-production) section above.

#### Server Prerequisites

**System Requirements:**
- Ubuntu 20.04 LTS or higher (or equivalent Linux distribution)
- Minimum 512MB RAM, 1 CPU core (current deployment: 458MB RAM)
- 8GB+ storage space (current deployment: 6.8GB total)
- Root or sudo access

**Current Production Server:**
- **OS**: Ubuntu 24.04.3 LTS (Codename: noble)
- **RAM**: 458MB total
- **Disk**: 6.8GB total (4.7GB used, 2.1GB available)
- **Go**: 1.23.2 linux/amd64
- **Nginx**: 1.24.0 (Ubuntu)

**Required Software:**
```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install essential packages
sudo apt install -y git nginx postgresql postgresql-contrib redis-server

# Install Go (latest version)
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install QStash (cloud-based, no local installation needed)
# Just sign up at https://upstash.com/ and get your credentials
```

#### Application Setup

1. **Create Application Directory:**
   ```bash
   sudo mkdir -p /home/ubuntu/apps/server-follow.email
   sudo chown -R ubuntu:ubuntu /home/ubuntu/apps/
   cd /home/ubuntu/apps/server-follow.email
   ```

2. **Deploy Pre-built Binary (Recommended):**
   ```bash
   # Create application directory
   sudo mkdir -p /home/ubuntu/apps/server-follow.email
   sudo chown -R ubuntu:ubuntu /home/ubuntu/apps/
   cd /home/ubuntu/apps/server-follow.email
   
   # Copy your pre-built Linux binary to the server
   # (Upload via scp, rsync, or your deployment method)
   # Example: scp follow_email user@server:/home/ubuntu/apps/server-follow.email/
   
   # Make binary executable
   chmod +x follow_email
   
   # Copy other necessary files
   # Copy .env.example, config files, etc. if needed
   ```

   **Alternative - Build on Server:**
   ```bash
   # Clone repository (only if building on server)
   git clone <your-repository-url> .
   
   # Install dependencies
   go mod download
   
   # Build the application (see Building for Production section for optimized build)
   GOOS=linux GOARCH=amd64 go build -o follow_email . 
   
   # Make binary executable
   chmod +x follow_email
   ```

3. **Environment Configuration:**
   ```bash
   # Create production environment file
   sudo nano /etc/follow_email.env
   ```
   
   **Note**: The current production deployment uses `/etc/follow_email.env` instead of the application directory. This provides better security by keeping environment variables outside the application directory.
   
   Add your production environment variables:
   ```bash
   # Server Configuration
   PORT=8080
   ENVIRONMENT=production
   LOG_LEVEL=info
   BASE_URL=https://your-domain.com
   
   # Database (use your production PostgreSQL)
   DATABASE_URL=postgres://username:password@localhost:5432/followemail?sslmode=require
   
   # Add all other required environment variables...
   ```

#### Current Deployment Status

**✅ Currently Deployed:**
- Go application running on port 8080
- Systemd service `follow_email.service` (active and enabled)
- Nginx reverse proxy on port 80
- Environment variables in `/etc/follow_email.env`

**⚠️ Missing/Recommended Enhancements:**
- SSL/TLS certificate (currently HTTP only)
- PostgreSQL database (psql command not found)
- Rate limiting in nginx
- Security headers
- Log rotation configuration
- Monitoring and alerting setup

#### Systemctl Service Configuration

1. **Create Systemd Service File:**
   ```bash
   sudo nano /etc/systemd/system/follow_email.service
   ```

2. **Service Configuration:**
   ```ini
   [Unit]
   Description=Follow Email Go App
   After=network.target

   [Service]
   User=ubuntu
   WorkingDirectory=/home/ubuntu/apps/server-follow.email
   ExecStart=/home/ubuntu/apps/server-follow.email/follow_email
   EnvironmentFile=/etc/follow_email.env
   Restart=always
   Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

   [Install]
   WantedBy=multi-user.target
   ```

   **Note**: The current production deployment uses a simplified configuration. For enhanced security, consider adding the following optional security settings:
   ```ini
   # Optional security enhancements (not in current production)
   NoNewPrivileges=true
   PrivateTmp=true
   ProtectSystem=strict
   ProtectHome=true
   ReadWritePaths=/home/ubuntu/apps/server-follow.email
   
   # Resource limits
   LimitNOFILE=65536
   LimitNPROC=4096
   ```

3. **Enable and Start Service:**
   ```bash
   # Reload systemd configuration
   sudo systemctl daemon-reload
   
   # Enable service to start on boot
   sudo systemctl enable follow_email
   
   # Start the service
   sudo systemctl start follow_email
   
   # Check service status
   sudo systemctl status follow_email
   ```

#### Nginx Reverse Proxy Configuration

1. **Create Nginx Site Configuration:**
   ```bash
   sudo nano /etc/nginx/sites-available/follow-email
   ```

2. **Nginx Configuration:**

   **Current Production Configuration (Basic Setup):**
   ```nginx
   server {
       listen 80;
       server_name _;  # Replace with your domain

       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection keep-alive;
           proxy_set_header Host $host;
           proxy_cache_bypass $http_upgrade;
       }
   }
   ```

   **Enhanced Production Configuration (Recommended):**
   ```nginx
   # Rate limiting
   limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
   limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/s;
   
   # Upstream backend
   upstream follow_email_backend {
       server 127.0.0.1:8080;
       keepalive 32;
   }
   
   server {
       listen 80;
       server_name your-domain.com www.your-domain.com;
       
       # Redirect HTTP to HTTPS (when SSL is configured)
       # return 301 https://$server_name$request_uri;
       
       # API routes
       location /api/ {
           limit_req zone=api burst=20 nodelay;
           
           proxy_pass http://follow_email_backend;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_cache_bypass $http_upgrade;
           
           # Timeouts
           proxy_connect_timeout 60s;
           proxy_send_timeout 60s;
           proxy_read_timeout 60s;
       }
       
       # Authentication routes (stricter rate limiting)
       location /api/v1/auth/ {
           limit_req zone=auth burst=10 nodelay;
           
           proxy_pass http://follow_email_backend;
           proxy_http_version 1.1;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
       
       # Health check endpoint
       location /api/v1/health {
           proxy_pass http://follow_email_backend;
           access_log off;
       }
       
       # Default location
       location / {
           proxy_pass http://follow_email_backend;
           proxy_http_version 1.1;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

3. **Enable Nginx Site:**
   ```bash
   # Enable the site
   sudo ln -s /etc/nginx/sites-available/follow-email /etc/nginx/sites-enabled/
   
   # Remove default site (optional)
   sudo rm /etc/nginx/sites-enabled/default
   
   # Test nginx configuration
   sudo nginx -t
   
   # Restart nginx
   sudo systemctl restart nginx
   sudo systemctl enable nginx
   ```

#### SSL Certificate Setup (Let's Encrypt)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Obtain SSL certificate
sudo certbot --nginx -d your-domain.com -d www.your-domain.com

# Auto-renewal (already set up by certbot)
sudo systemctl status certbot.timer
```

#### CI/CD Deployment with GitHub Actions

The repository includes automated deployment via GitHub Actions. Set up the following secrets in your GitHub repository:

**Required GitHub Secrets:**
- `SERVER_SSH_KEY`: Your server's SSH private key
- `SERVER_HOST`: Your server's IP address or domain
- `SERVER_USER`: SSH username (usually `ubuntu`)

**Deployment Process:**
1. Push code to `master` branch
2. GitHub Actions automatically:
   - Connects to your server via SSH
   - Pulls latest code from repository
   - Verifies swagger files are present
   - Restarts the `follow_email` systemctl service
   - Restarts nginx
   - Checks service status

**Manual Deployment Commands:**
```bash
# SSH into your server
ssh ubuntu@your-server-ip

# Navigate to application directory
cd /home/ubuntu/apps/server-follow.email

# Pull latest changes
git pull origin master

# Rebuild application (if needed)
GOOS=linux GOARCH=amd64 go build -o follow_email .

# Restart services
sudo systemctl restart follow_email
sudo systemctl restart nginx

# Check status
sudo systemctl status follow_email
sudo systemctl status nginx
```

#### Database Setup

1. **PostgreSQL Configuration:**
   ```bash
   # Switch to postgres user
   sudo -u postgres psql
   
   # Create database and user
   CREATE DATABASE followemail;
   CREATE USER followemail_user WITH ENCRYPTED PASSWORD 'your_secure_password';
   GRANT ALL PRIVILEGES ON DATABASE followemail TO followemail_user;
   \q
   ```

2. **Database Migrations:**
   ```bash
   # Migrations run automatically on application startup
   # Check logs to verify successful migration
   sudo journalctl -u follow_email -f
   ```

#### Monitoring & Logging

**Service Logs:**
```bash
# View real-time logs
sudo journalctl -u follow_email -f

# View recent logs
sudo journalctl -u follow_email -n 100

# View nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

**System Monitoring:**
```bash
# Check service status
sudo systemctl status follow_email nginx postgresql redis

# Check resource usage
htop
df -h
free -h

# Check network connections
sudo netstat -tlnp | grep :8080
sudo netstat -tlnp | grep :80
```

**Log Rotation (recommended):**
```bash
# Create logrotate configuration
sudo nano /etc/logrotate.d/follow_email
```

Add the following configuration:
```
/var/log/nginx/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 www-data www-data
    postrotate
        systemctl reload nginx
    endscript
}
```

### Kubernetes Deployment (Alternative)

The application includes comprehensive Kubernetes manifests in `/deployments/`:

```bash
# Deploy to Kubernetes
cd deployments
./deploy.sh
```

### Environment-Specific Configuration
- **Development**: Uses `.env` file, debug logging enabled
- **Staging**: Environment variables from system, structured logging
- **Production**: Secure environment variables, optimized performance settings

### Health Checks
- **Liveness Probe**: `/api/v1/health`
- **Readiness Probe**: `/api/v1/ping`
- **Startup Probe**: Database connectivity check

## 🔧 Development

### Project Structure
```
├── cmd/                    # Application entry points
├── config/                 # Configuration management
├── internal/              # Private application code
│   ├── handlers/          # HTTP request handlers
│   ├── middleware/        # HTTP middleware
│   ├── models/           # Data models
│   ├── routes/           # Route definitions
│   └── services/         # Business logic
├── pkg/                  # Public packages
│   ├── ai/              # AI integration
│   ├── oauth/           # OAuth providers
│   ├── queue/           # Message queue
│   └── storage/         # File storage
├── migrations/          # Database migrations
├── deployments/        # Kubernetes manifests
└── swagger-ui/         # API documentation UI
```

### Code Style & Standards
- Follow Go conventions and `gofmt` formatting
- Use structured logging with correlation IDs
- Implement comprehensive error handling
- Write unit tests for business logic
- Document public APIs with OpenAPI/Swagger

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/handlers/
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📞 Support

For support and questions:
- Create an issue in the repository
- Check the [API Documentation](http://localhost:8080/swagger-ui/)
- Review the deployment guides in `/deployments/README.md`

---

**Built with ❤️ using Go, PostgreSQL, and modern cloud technologies.**

ssh -i ~/.ssh/id_ed25519 ubuntu@210.79.129.124