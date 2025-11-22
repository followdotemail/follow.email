# Setup Guide - Follow Email

This guide will walk you through the process of setting up and running the Follow Email application, both for development and production environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Configuration](#environment-configuration)
3. [Setup Methods](#setup-methods)
   - [Local Development (Without Docker)](#local-development-without-docker)
   - [Docker Workflow](#docker-workflow)
4. [Running the Application](#running-the-application)
5. [Deployment](#deployment)
   - [Excloud Provisioning Script](#excloud-provisioning-script)
   - [Docker Production Deployment](#docker-production-deployment)
6. [Troubleshooting](#troubleshooting)
7. [Getting Help](#getting-help)
8. [Quick Reference](#quick-reference)

## Prerequisites

Before you begin, ensure you have the following installed:

### Required Software

- **Node.js** >= 18.0.0
- **npm** >= 9.0.0
- **Go** >= 1.23.0
- **Docker Desktop** (includes Docker Compose) — required for provisioning, CI parity, and validating container builds. Optional for day-to-day local runs but strongly recommended.

### Additional Tooling

- **Python** >= 3.9 (needed for `apps/hermes/deployments/provision-dev-instance.py`)
- **psql CLI** (optional, for inspecting the managed Neon database)

### Required Accounts & API Keys

- **Neon** (or another managed Postgres provider) for the primary database connection string
- **Clerk** account for authentication
- **Google Cloud Console** project with Gmail API enabled
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

# Database (Neon / managed Postgres)
DATABASE_URL=postgresql://username:password@your-neon-hostname/neondb?sslmode=require

# Google OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/gmail/callback
```

See `env.example` for all available configuration options.

## Setup Methods

### Local Development (Without Docker)

Use this path for day-to-day feature development on your workstation.

#### 1. Install Dependencies

- **Root workspace**
  ```bash
  npm install
  ```
- **Frontend**
  ```bash
  cd apps/frontend
  npm install
  cd ../..
  ```
- **Backend**
  ```bash
  cd apps/hermes
  go mod download
  go mod tidy
  cd ../..
  ```

#### 2. Configure Remote Database Access

- Set `DATABASE_URL` in `.env` to the Neon connection string (make sure `sslmode=require`).
- (Optional) Verify connectivity:
  ```bash
  psql "$DATABASE_URL" -c '\dt'
  ```

#### 3. Database Schema

- Backend migrations run automatically on start-up; files live in `apps/hermes/migrations/`.
- No local PostgreSQL service is provisioned—use Neon branches for isolated testing if needed.

### Docker Workflow

Docker provides environment parity with the managed infrastructure and is required for release verification.

#### 1. Install Docker

Download and install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop/).

#### 2. Configure Environment

Ensure your `.env` file is populated (see [Environment Configuration](#environment-configuration)).

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
cd apps/hermes
./follow_email
```

## Deployment

### Excloud Provisioning Script

Use `apps/hermes/deployments/provision-dev-instance.py` to create and manage ephemeral or long-lived cloud instances. The script:

1. Provisions an Excloud VM and attaches a public IP.
2. Installs Docker, nginx, and supporting packages.
3. Clones this repository, copies the root `.env`, and builds the backend Docker image.
4. Updates the `api.follow.email` DNS record via Spaceship.
5. Starts the backend container and verifies health checks.
6. Persists VM metadata to `apps/hermes/deployments/instance-info.json` for subsequent operations.

**Prerequisites**

- Python 3.9+ and `pip install -r apps/hermes/deployments/requirements.txt`
- SSH private key with access to the provisioned instance (defaults to `~/.ssh/id_ed25519`)
- Root `.env` populated with:
  - Excloud settings: `EXCLOUD_API_KEY`, `EXCLOUD_ZONE_ID`, `EXCLOUD_SUBNET_ID`, `EXCLOUD_PROJECT_ID`, `EXCLOUD_SECURITY_GROUP_ID`, `EXCLOUD_SSH_PUBKEY`
  - Spaceship credentials: `SPACESHIP_API_KEY`, `SPACESHIP_API_SECRET`
  - Application secrets including `DATABASE_URL` (Neon), Clerk, Google OAuth, etc.

**Create or Rebuild an Instance**

```bash
cd apps/hermes/deployments
python provision-dev-instance.py --action create --record-name api --ssh-key ~/.ssh/id_ed25519
```

**Other Useful Actions**

- Destroy instance:
  ```bash
  python provision-dev-instance.py --action destroy --instance-id <vm_id>
  ```
- Re-run setup against an existing VM (after manual changes):
  ```bash
  python provision-dev-instance.py --action setup-only --record-name api
  ```
- Troubleshoot failed deployments (requires `instance-info.json` and the VM ID):
  ```bash
  python provision-dev-instance.py --action create --instance-id <vm_id> --troubleshoot
  ```
  > `--action` remains mandatory even when troubleshooting; pass the last action you executed (typically `create`).

**Key Arguments**

- `--action {create,destroy,setup-only}` (required)
- `--instance-id` / `--vm-id` (required for `destroy`, used by `--troubleshoot`)
- `--ssh-key` (path to private key, default `~/.ssh/id_ed25519`)
- `--github-repo` (defaults to `https://github.com/followdotemail/follow.email.git`)
- `--record-name` (DNS prefix, default `api`)
- `--troubleshoot` (run diagnostics without provisioning a new VM)

### Docker Production Deployment

The provisioning script ultimately runs the Docker stack defined in `infra/docker-compose.yml` on the remote VM. If you need to manage it manually (for example, during debugging over SSH):

```bash
cd /opt/follow.email/infra
sudo docker compose up -d backend
sudo docker compose logs -f backend
```

The frontend container is not currently deployed in production; the backend is exposed through nginx on the host.

## Troubleshooting

### Common Issues

#### Port Already in Use

```bash
# Find and kill process on port 3000
npx kill-port 3000

# Find and kill process on port 8080
npx kill-port 8080
```

#### Neon Database Connection Failed

- Confirm the Neon branch is online via the Neon console.
- Check `DATABASE_URL` in `.env` (correct host, database name, user, and `sslmode=require`).
- Test connectivity from your machine:
  ```bash
  psql "$DATABASE_URL" -c 'select now();'
  ```
- Ensure the Neon IP allowlist includes your current location or set the project to "trusted by default".

#### Go Module Issues

```bash
cd apps/hermes
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
2. Verify migration files exist in `apps/hermes/migrations/`
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
- Review API documentation in `apps/hermes/documents/`
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

**Last Updated**: November 2025

