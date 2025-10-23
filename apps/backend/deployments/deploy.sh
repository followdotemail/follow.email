#!/bin/bash

# FollowEmail Backend Kubernetes Deployment Script
# This script deploys the FollowEmail backend application to Kubernetes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Check if we can connect to Kubernetes cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

print_status "Starting FollowEmail Backend deployment..."

# Create namespace
print_status "Creating namespace..."
kubectl apply -f namespace.yaml

# Apply ConfigMaps and Secrets
print_status "Applying ConfigMaps..."
kubectl apply -f configmap.yaml

print_warning "Please update the secrets in secrets.yaml with your actual values before proceeding!"
read -p "Have you updated the secrets? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_error "Please update the secrets in secrets.yaml and run the script again."
    exit 1
fi

print_status "Applying Secrets..."
kubectl apply -f secrets.yaml

# Deploy infrastructure services
print_status "Deploying PostgreSQL..."
kubectl apply -f postgres-deployment.yaml

print_status "Deploying RabbitMQ..."
kubectl apply -f rabbitmq-deployment.yaml

print_status "Deploying Redis..."
kubectl apply -f redis-deployment.yaml

# Wait for infrastructure services to be ready
print_status "Waiting for infrastructure services to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/postgres -n followemail
kubectl wait --for=condition=available --timeout=300s deployment/rabbitmq -n followemail
kubectl wait --for=condition=available --timeout=300s deployment/redis -n followemail

print_success "Infrastructure services are ready!"

# Deploy the main application
print_status "Deploying FollowEmail application..."
kubectl apply -f app-deployment.yaml

# Wait for application to be ready
print_status "Waiting for application to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/followemail-app -n followemail

print_success "Application deployed successfully!"

# Apply Ingress (optional)
read -p "Do you want to deploy the Ingress? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_warning "Please update the domain in ingress.yaml before applying!"
    read -p "Have you updated the domain? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Applying Ingress..."
        kubectl apply -f ingress.yaml
        print_success "Ingress applied successfully!"
    else
        print_warning "Skipping Ingress deployment. You can apply it later with: kubectl apply -f ingress.yaml"
    fi
else
    print_warning "Skipping Ingress deployment."
fi

# Display deployment status
print_status "Deployment Summary:"
echo "=================="
kubectl get pods -n followemail
echo
kubectl get services -n followemail
echo

print_success "FollowEmail Backend deployment completed!"
print_status "You can check the application logs with: kubectl logs -f deployment/followemail-app -n followemail"
print_status "To access the application locally: kubectl port-forward service/followemail-app-service 8080:80 -n followemail"

# Check if application is responding
print_status "Testing application health..."
kubectl port-forward service/followemail-app-service 8080:80 -n followemail &
PORT_FORWARD_PID=$!
sleep 5

if curl -s http://localhost:8080/health > /dev/null; then
    print_success "Application is responding to health checks!"
else
    print_warning "Application health check failed. Please check the logs."
fi

kill $PORT_FORWARD_PID 2>/dev/null || true

print_success "Deployment script completed!"