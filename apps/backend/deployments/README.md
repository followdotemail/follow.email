# FollowEmail Backend Deployment

This directory contains Kubernetes deployment manifests and Docker configuration for the FollowEmail backend application.

## Prerequisites

- Docker and Docker Compose (for local development)
- Kubernetes cluster (for production deployment)
- kubectl configured to access your cluster
- NGINX Ingress Controller (if using Ingress)
- cert-manager (for TLS certificates)

## Local Development with Docker Compose

### Quick Start

1. **Build and run the application:**
   ```bash
   # From the project root
   docker-compose up --build
   ```

2. **Access the application:**
   - API: http://localhost:8080
   - RabbitMQ Management: http://localhost:15672 (admin/password)
   - MinIO Console: http://localhost:9001 (minioadmin/minioadmin)

3. **Stop the application:**
   ```bash
   docker-compose down
   ```

### Environment Variables

The Docker Compose setup uses the following default values:
- PostgreSQL: `postgres:password@localhost:5432/followemail`
- RabbitMQ: `amqp://admin:password@localhost:5672/`
- Redis: `localhost:6379`
- MinIO: `localhost:9000`

## Kubernetes Production Deployment

### Architecture

The Kubernetes deployment includes:
- **Main Application**: Go backend service with auto-scaling
- **PostgreSQL**: Database with persistent storage
- **RabbitMQ**: Message queue with management interface
- **Redis**: Caching and rate limiting
- **Ingress**: HTTPS termination and routing

### Deployment Steps

1. **Update Configuration:**
   
   **Secrets** (`secrets.yaml`):
   ```bash
   # Update these base64-encoded values:
   - DATABASE_URL
   - RABBITMQ_URL
   - AWS_ACCESS_KEY_ID
   - AWS_SECRET_ACCESS_KEY
   - OPENAI_API_KEY
   - GOOGLE_CLIENT_ID
   - GOOGLE_CLIENT_SECRET
   - JWT_SECRET
   ```
   
   **ConfigMap** (`configmap.yaml`):
   ```bash
   # Update these values as needed:
   - AWS_REGION
   - S3_BUCKET_NAME
   - SERVER_PORT
   - LOG_LEVEL
   ```
   
   **Ingress** (`ingress.yaml`):
   ```bash
   # Update the domain name:
   host: api.followemail.com  # Change to your domain
   ```

2. **Deploy using the script:**
   ```bash
   cd deployments
   ./deploy.sh
   ```

3. **Manual deployment (alternative):**
   ```bash
   # Create namespace
   kubectl apply -f namespace.yaml
   
   # Apply configuration
   kubectl apply -f configmap.yaml
   kubectl apply -f secrets.yaml
   
   # Deploy infrastructure
   kubectl apply -f postgres-deployment.yaml
   kubectl apply -f rabbitmq-deployment.yaml
   kubectl apply -f redis-deployment.yaml
   
   # Deploy application
   kubectl apply -f app-deployment.yaml
   
   # Deploy ingress (optional)
   kubectl apply -f ingress.yaml
   ```

### Monitoring and Maintenance

**Check deployment status:**
```bash
kubectl get pods -n followemail
kubectl get services -n followemail
```

**View application logs:**
```bash
kubectl logs -f deployment/followemail-app -n followemail
```

**Access application locally:**
```bash
kubectl port-forward service/followemail-app-service 8080:80 -n followemail
```

**Scale the application:**
```bash
kubectl scale deployment followemail-app --replicas=5 -n followemail
```

### Resource Requirements

**Minimum Cluster Resources:**
- CPU: 2 cores
- Memory: 4GB RAM
- Storage: 20GB

**Production Recommendations:**
- CPU: 4+ cores
- Memory: 8GB+ RAM
- Storage: 100GB+ SSD

### Security Considerations

1. **Secrets Management:**
   - Use external secret management (e.g., AWS Secrets Manager, HashiCorp Vault)
   - Rotate secrets regularly
   - Never commit actual secrets to version control

2. **Network Security:**
   - Use NetworkPolicies to restrict pod-to-pod communication
   - Enable TLS for all external communications
   - Use private container registries

3. **RBAC:**
   - Implement proper Role-Based Access Control
   - Use service accounts with minimal permissions

### Troubleshooting

**Common Issues:**

1. **Pods not starting:**
   ```bash
   kubectl describe pod <pod-name> -n followemail
   kubectl logs <pod-name> -n followemail
   ```

2. **Database connection issues:**
   - Check PostgreSQL pod status
   - Verify DATABASE_URL secret
   - Check network connectivity

3. **Ingress not working:**
   - Verify NGINX Ingress Controller is installed
   - Check cert-manager for TLS issues
   - Verify DNS configuration

**Useful Commands:**
```bash
# Get all resources in namespace
kubectl get all -n followemail

# Describe a problematic pod
kubectl describe pod <pod-name> -n followemail

# Execute into a pod
kubectl exec -it <pod-name> -n followemail -- /bin/sh

# Check events
kubectl get events -n followemail --sort-by='.lastTimestamp'
```

### Backup and Recovery

**Database Backup:**
```bash
# Create a backup job
kubectl create job --from=cronjob/postgres-backup manual-backup -n followemail

# Manual backup
kubectl exec -it deployment/postgres -n followemail -- pg_dump -U postgres followemail > backup.sql
```

**Restore Database:**
```bash
kubectl exec -i deployment/postgres -n followemail -- psql -U postgres followemail < backup.sql
```

### Performance Tuning

1. **Application Scaling:**
   - Configure Horizontal Pod Autoscaler (HPA)
   - Adjust resource requests and limits
   - Use multiple replicas for high availability

2. **Database Optimization:**
   - Tune PostgreSQL configuration
   - Use connection pooling
   - Monitor query performance

3. **Caching:**
   - Optimize Redis configuration
   - Implement application-level caching
   - Use CDN for static assets

## File Structure

```
deployments/
├── README.md                 # This file
├── deploy.sh                 # Automated deployment script
├── namespace.yaml            # Kubernetes namespace
├── configmap.yaml            # Application configuration
├── secrets.yaml              # Sensitive configuration (template)
├── app-deployment.yaml       # Main application deployment
├── postgres-deployment.yaml  # PostgreSQL database
├── rabbitmq-deployment.yaml  # RabbitMQ message queue
├── redis-deployment.yaml     # Redis cache
└── ingress.yaml             # HTTPS ingress configuration
```

## Support

For deployment issues or questions:
1. Check the troubleshooting section above
2. Review application logs
3. Consult Kubernetes documentation
4. Contact the development team