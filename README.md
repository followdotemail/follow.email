# Follow Email - Monorepo

A modern email management platform built with Next.js and Go, structured as a monorepo for better code organization and developer experience.

## 📁 Project Structure

```
follow.email/
├── .github/
│   └── workflows/
│       └── deploy.yml     # GitHub Actions deployment workflow
├── apps/
│   ├── frontend/          # Next.js Frontend Application
│   └── backend/           # Go Backend API
├── infra/                 # Infrastructure configuration
│   ├── docker-compose.yml # Docker composition
│   ├── nginx.conf         # Nginx configuration
│   ├── Dockerfile.frontend
│   └── Dockerfile.backend
├── deployment/            # Deployment documentation
├── packages/              # Shared packages (for future use)
├── env.example            # Environment variables template
├── package.json           # Root workspace configuration
├── turbo.json            # Turborepo configuration
├── Setup.md              # Detailed setup instructions
└── README.md             # This file
```

## 🚀 Quick Start

### Prerequisites

- **Node.js** >= 18.0.0
- **npm** >= 9.0.0
- **Go** >= 1.24.0
- **Docker** and **Docker Compose** (for Docker setup)
- **PostgreSQL** (for manual setup)
- **Redis** (for manual setup)

### Installation

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd follow.email
   ```

2. **Install dependencies:**
   ```bash
   npm install
   ```

3. **Configure environment:**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

4. **Start development:**

   **Using Docker (Recommended):**
   ```bash
   npm run docker:up
   ```

   **Manual:**
   ```bash
   npm run dev
   ```

📖 **For detailed setup instructions, see [Setup.md](./Setup.md)**

## 🛠️ Development

### Running Services

**All services with Turborepo:**
```bash
npm run dev
```

**Frontend only:**
```bash
npm run frontend:dev
```

**Backend only:**
```bash
npm run backend:dev
```

**Using Docker:**
```bash
npm run docker:up      # Start all services
npm run docker:logs    # View logs
npm run docker:down    # Stop services
```

### Access Points

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080/api/v1
- **Swagger UI**: http://localhost:8080/swagger-ui/
- **Nginx**: http://localhost (when using Docker with Nginx)

## 📦 Build

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
```

## 📝 Available Scripts

### Root Level Commands

| Command | Description |
|---------|-------------|
| `npm run dev` | Start all apps with Turborepo |
| `npm run build` | Build all apps |
| `npm run start` | Start all apps in production mode |
| `npm run lint` | Lint all apps |
| `npm run format` | Format code in all apps |
| `npm run frontend:dev` | Start frontend only |
| `npm run frontend:build` | Build frontend |
| `npm run backend:dev` | Start backend only |
| `npm run backend:build` | Build backend |
| `npm run backend:test` | Run backend tests |
| `npm run docker:build` | Build Docker images |
| `npm run docker:up` | Start Docker services |
| `npm run docker:down` | Stop Docker services |
| `npm run docker:logs` | View Docker logs |

## 🏗️ Applications

### Frontend (`apps/frontend`)

Modern Next.js application featuring:
- **Framework**: Next.js 15.5.3 with App Router
- **UI**: Tailwind CSS 4.0 with Radix UI components
- **Authentication**: Clerk
- **State Management**: Jotai
- **API Client**: Axios with SWR
- **Styling**: Tailwind CSS with custom animations

### Backend (`apps/backend`)

Robust Go backend with:
- **Framework**: Gin
- **Database**: PostgreSQL with GORM
- **Authentication**: Clerk integration
- **Email**: Gmail API integration
- **Storage**: AWS S3
- **AI**: Google Gemini
- **Queue**: RabbitMQ and Upstash QStash

## 🔧 Technology Stack

### Frontend
- Next.js 15.5.3
- React 19.1.0
- TypeScript 5.x
- Tailwind CSS 4.x
- Clerk Authentication
- Radix UI Components

### Backend
- Go 1.24
- Gin Web Framework
- PostgreSQL with GORM
- Redis
- OAuth2 (Google)
- AWS SDK
- Google Generative AI

### Infrastructure
- Turborepo for monorepo management
- Docker & Docker Compose
- Nginx as reverse proxy
- GitHub Actions for CI/CD

## 📚 Documentation

- **[Setup.md](./Setup.md)** - Complete setup and deployment guide
- **Backend API Documentation**: `apps/backend/documents/api-documentation.md`
- **Auth Flow**: `apps/backend/documents/auth-flow.md`
- **System Design**: `apps/backend/documents/system-design.md`
- **Swagger UI**: Available at http://localhost:8080/swagger-ui/ when running

## 🐳 Docker Support

The project includes full Docker support with:

- **docker-compose.yml**: Orchestrates frontend and backend services
- **nginx.conf**: Reverse proxy configuration
- **Dockerfile.frontend**: Multi-stage build for Next.js
- **Dockerfile.backend**: Multi-stage build for Go

All Docker configurations are in the `infra/` directory.

## 🚢 Deployment

### Automated Deployment (GitHub Actions)

The project includes a GitHub Actions workflow for automated deployment at `.github/workflows/deploy.yml`.

**Setup:**

1. Configure GitHub Secrets in your repository settings:
   - `SERVER_SSH_KEY` - Your SSH private key
   - `SERVER_HOST` - Server hostname or IP
   - `SERVER_USER` - SSH username (e.g., ubuntu)

2. Push to master/main branch to trigger automatic deployment

The workflow will automatically build and deploy both frontend and backend to your server.

### Manual Deployment

See [Setup.md](./Setup.md) for detailed deployment instructions including:
- Server setup
- Environment configuration
- Service configuration (systemd, PM2)
- Nginx setup
- SSL/HTTPS configuration

### Kubernetes Deployment

Kubernetes configurations are available in `apps/backend/deployments/`:
- Namespace, ConfigMap, Secrets
- PostgreSQL, Redis deployments
- Application deployment
- Ingress configuration

## 🔐 Environment Variables

All environment variables are configured in a single `env.example` file at the root. Copy it to `.env` and configure:

```bash
cp env.example .env
```

The file includes configuration for:
- Frontend (Clerk, API URLs)
- Backend (Database, Redis, OAuth, AWS, etc.)
- Docker (PostgreSQL, Redis for containers)

## 🧪 Testing

**Backend tests:**
```bash
npm run backend:test
```

Or directly:
```bash
cd apps/backend
go test ./...
go test -v ./...  # Verbose output
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

[Add your license here]

## 👥 Authors

[Add authors here]

## 🙏 Acknowledgments

- Built with modern monorepo best practices
- Follows microservices architecture
- Production-ready with Docker and Kubernetes support
- Comprehensive documentation and setup guides

---

For detailed setup instructions, troubleshooting, and deployment guides, please see [Setup.md](./Setup.md).
