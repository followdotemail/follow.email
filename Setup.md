# Setup Guide - Follow Email

This guide will walk you through the process of setting up and running the Follow Email application, both for development and production environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Configuration](#environment-configuration)
3. [Setup Methods](#setup-methods)
   - [Docker Setup (Recommended)](#docker-setup-recommended)
   - [Manual Setup](#manual-setup)
4. [Running the Application](#running-the-application)
5. [Deployment](#deployment)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

Before you begin, ensure you have the following installed:

### Required Software

- **Node.js** >= 18.0.0
- **npm** >= 9.0.0
- **Go** >= 1.24.0
- **Docker** and **Docker Compose** (for Docker setup)
- **PostgreSQL** >= 13 (for manual setup)
- **Redis** >= 6 (for manual setup)

### Required Accounts & API Keys

- **Clerk** account for authentication
- **Google Cloud Console** account for Gmail API
- **AWS** account for S3 storage (optional)
- **Google AI** account for Gemini API (optional)
- **Upstash** account for QStash (optional)

## Environment Configuration

### 1. Copy Environment Template

```bash
cp env.example .env
```

### 2. Configure Environment Variables

Edit the `.env` file and fill in your configuration:

#### Frontend Variables

```env
# Clerk Authentication
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_your_key
CLERK_SECRET_KEY=sk_test_your_key

# API Configuration
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_URL=http://localhost:3000
```

#### Backend Variables

```env
# Server
PORT=8080
GIN_MODE=debug

# Database
DATABASE_URL=postgresql://username:password@localhost:5432/follow_email?sslmode=disable

# Redis
REDIS_URL=localhost:6379

# Google OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/gmail/callback
```

See `env.example` for all available configuration options.

## Setup Methods

### Docker Setup (Recommended)

Docker provides an isolated and consistent environment for development and production.

#### 1. Install Docker

Download and install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop/)

#### 2. Configure Environment

Make sure your `.env` file is properly configured (see above).

#### 3. Build and Start Services

```bash
# Build Docker images
npm run docker:build

# Start all services
npm run docker:up

# View logs
npm run docker:logs
```

#### 4. Verify Services

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080/api/v1
- Swagger UI: http://localhost:8080/swagger-ui/
- Nginx (if configured): http://localhost

#### 5. Stop Services

```bash
# Stop all services
npm run docker:down
```

### Manual Setup

If you prefer not to use Docker, follow these steps:

#### 1. Install Dependencies

**Root dependencies:**
```bash
npm install
```

**Frontend dependencies:**
```bash
cd apps/frontend
npm install
cd ../..
```

**Backend dependencies:**
```bash
cd apps/backend
go mod download
go mod tidy
cd ../..
```

#### 2. Set Up Database

**Create PostgreSQL database:**
```sql
CREATE DATABASE follow_email;
```

**Set up Redis:**
```bash
# Install Redis (Ubuntu/Debian)
sudo apt-get install redis-server

# Start Redis
sudo systemctl start redis

# Verify Redis is running
redis-cli ping
```

#### 3. Configure Database URL

Update your `.env` file with the correct database connection string:
```env
DATABASE_URL=postgresql://your_user:your_password@localhost:5432/follow_email?sslmode=disable
```

#### 4. Run Database Migrations

Migrations run automatically when the backend starts. The migration files are in `apps/backend/migrations/`.

## Running the Application

### Using Turborepo (All Services)

```bash
# Start both frontend and backend
npm run dev
```

This command uses Turborepo to run both applications concurrently.

### Individual Services

**Frontend only:**
```bash
npm run frontend:dev
```

**Backend only:**
```bash
npm run backend:dev
```

### Access Points

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080/api/v1
- **Swagger Documentation**: http://localhost:8080/swagger-ui/

## Production Build

### Build All Applications

```bash
npm run build
```

### Build Individual Applications

**Frontend:**
```bash
npm run frontend:build
npm run frontend:start
```

**Backend:**
```bash
npm run backend:build
cd apps/backend
./follow_email
```

## Deployment

### GitHub Actions Deployment

The project includes a GitHub Actions workflow for automated deployment located at `.github/workflows/deploy.yml`.

#### 1. Configure GitHub Secrets

Go to your repository settings and add these secrets:

- `SERVER_SSH_KEY` - Your SSH private key for the server
- `SERVER_HOST` - Your server hostname or IP
- `SERVER_USER` - SSH username (e.g., ubuntu)

#### 2. Push to Master Branch

The workflow automatically deploys when you push to the `master` or `main` branch.

#### 3. How It Works

The workflow will:
1. SSH into your server
2. Pull the latest code
3. Build both backend (Go) and frontend (Next.js)
4. Restart services (systemd for backend, PM2 for frontend)
5. Restart Nginx
6. Verify deployment status

### Manual Deployment

#### On Your Server

1. **Clone the repository:**
```bash
cd /home/ubuntu/apps
git clone <your-repo-url> follow.email
cd follow.email
```

2. **Install dependencies:**
```bash
npm install
cd apps/frontend && npm install && cd ../..
cd apps/backend && go mod download && cd ../..
```

3. **Configure environment:**
```bash
cp env.example .env
# Edit .env with your production values
nano .env
```

4. **Build applications:**
```bash
npm run frontend:build
npm run backend:build
```

5. **Set up systemd service (Backend):**

Create `/etc/systemd/system/follow-email-backend.service`:

```ini
[Unit]
Description=Follow Email Backend
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/apps/follow.email/apps/backend
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"
EnvironmentFile=/home/ubuntu/apps/follow.email/.env
ExecStart=/home/ubuntu/apps/follow.email/apps/backend/follow_email
Restart=always

[Install]
WantedBy=multi-user.target
```

6. **Set up PM2 (Frontend):**

```bash
npm install -g pm2
cd apps/frontend
pm2 start npm --name "follow-email-frontend" -- start
pm2 save
pm2 startup
```

7. **Configure Nginx:**

Use the nginx configuration from `infra/nginx.conf` as a template.

```bash
sudo cp infra/nginx.conf /etc/nginx/sites-available/follow-email
sudo ln -s /etc/nginx/sites-available/follow-email /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### Docker Production Deployment

For production with Docker:

```bash
# Use docker-compose
cd infra
docker-compose -f docker-compose.yml up -d
```

## Troubleshooting

### Common Issues

#### Port Already in Use

```bash
# Find and kill process on port 3000
npx kill-port 3000

# Find and kill process on port 8080
npx kill-port 8080
```

#### Database Connection Failed

- Verify PostgreSQL is running: `sudo systemctl status postgresql`
- Check database credentials in `.env`
- Test connection: `psql -U username -d follow_email`

#### Redis Connection Failed

- Verify Redis is running: `sudo systemctl status redis`
- Test connection: `redis-cli ping`

#### Go Module Issues

```bash
cd apps/backend
go clean -modcache
go mod tidy
go mod download
```

#### Node Module Issues

```bash
# Clean root
rm -rf node_modules package-lock.json
npm install

# Clean frontend
cd apps/frontend
rm -rf node_modules package-lock.json
npm install
```

#### Docker Issues

```bash
# Clean Docker cache
docker system prune -a

# Rebuild images
npm run docker:build

# View logs for specific service
docker-compose -f infra/docker-compose.yml logs frontend
docker-compose -f infra/docker-compose.yml logs backend
```

### Environment Variable Issues

If environment variables are not being read:

1. Ensure `.env` file exists at the root
2. Check file permissions: `chmod 644 .env`
3. Verify variable names match exactly
4. Restart the services after changing `.env`

### Migration Issues

If database migrations fail:

1. Check database connectivity
2. Verify migration files exist in `apps/backend/migrations/`
3. Check migration logs in backend console

### SSL/HTTPS Setup

For production with HTTPS:

1. Obtain SSL certificates (Let's Encrypt recommended)
2. Update nginx configuration in `infra/nginx.conf`
3. Uncomment HTTPS server block
4. Update certificate paths
5. Restart nginx

## Getting Help

- Check the main [README.md](./README.md) for project overview
- Review API documentation in `apps/backend/documents/`
- Check Swagger UI at http://localhost:8080/swagger-ui/ when running

## Quick Reference

### Development Commands

```bash
npm run dev                 # Start all services
npm run frontend:dev        # Start frontend only
npm run backend:dev         # Start backend only
npm run backend:test        # Run backend tests
```

### Build Commands

```bash
npm run build               # Build all
npm run frontend:build      # Build frontend
npm run backend:build       # Build backend
```

### Docker Commands

```bash
npm run docker:build        # Build Docker images
npm run docker:up           # Start containers
npm run docker:down         # Stop containers
npm run docker:logs         # View logs
```

---

**Last Updated**: October 2025

