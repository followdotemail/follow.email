# Deployment Directory

This directory contains deployment-related configurations and documentation.

## GitHub Actions Workflow

The GitHub Actions workflow for automated deployment is located at:
```
.github/workflows/deploy.yml
```

This workflow automatically deploys to your server when you push to the `master` or `main` branch.

## Configuration

To use the automated deployment:

1. **Configure GitHub Secrets** in your repository settings:
   - `SERVER_SSH_KEY` - Your SSH private key
   - `SERVER_HOST` - Server hostname or IP
   - `SERVER_USER` - SSH username

2. **Push to master/main branch** - The workflow will trigger automatically

## Kubernetes Deployment

For Kubernetes deployment, see the configurations in:
```
apps/hermes/deployments/
```

This includes:
- Namespace configuration
- PostgreSQL and Redis deployments
- Application deployment
- Ingress configuration
- ConfigMaps and Secrets

## Docker Deployment

For Docker-based deployment, see:
```
infra/docker-compose.yml
infra/nginx.conf
```

Run with:
```bash
npm run docker:up
```

## Manual Deployment

For manual deployment instructions, see the main [Setup.md](../Setup.md) guide.

