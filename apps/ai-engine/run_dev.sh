#!/bin/bash

# Telar AI Engine Development Runner
# This script starts the Docker Compose backend services
# It works from any directory by auto-detecting the project structure

set -e  # Exit on any error

# Default port configuration
AI_ENGINE_PORT=${AI_ENGINE_PORT:-8000}
WEAVIATE_PORT=${WEAVIATE_PORT:-8080}

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

# Function to find the project root
find_project_root() {
    # First, try to find from current directory
    local current_dir="$PWD"
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/apps/ai-engine" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir=$(dirname "$current_dir")
    done
    
    # If not found, try from script location
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    current_dir="$script_dir"
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/apps/ai-engine" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir=$(dirname "$current_dir")
    done
    
    print_error "Could not find project root (looking for apps/ai-engine directory)"
    exit 1
}

# Function to check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    print_success "Docker is running"
}

# Function to start Docker Compose services
start_docker_services() {
    local docker_compose_dir="$1/apps/ai-engine/deployments/docker-compose"
    
    if [[ ! -f "$docker_compose_dir/docker-compose.yml" ]]; then
        print_error "Docker Compose file not found at $docker_compose_dir/docker-compose.yml"
        exit 1
    fi
    
    print_status "Starting Docker Compose services..."
    cd "$docker_compose_dir"
    
    # Stop any existing services first
    docker compose down -v >/dev/null 2>&1 || true
    
    # Start services
    docker compose up -d --build
    
    if [[ $? -eq 0 ]]; then
        print_success "Docker services started successfully"
    else
        print_error "Failed to start Docker services"
        exit 1
    fi
}

# Function to wait for services to be healthy
wait_for_services() {
    print_status "Waiting for AI Engine to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s http://localhost:${AI_ENGINE_PORT}/health >/dev/null 2>&1; then
            print_success "AI Engine is healthy and ready!"
            break
        fi
        
        if [[ $attempt -eq $max_attempts ]]; then
            print_error "AI Engine failed to start within expected time"
            print_status "Check Docker logs: docker compose logs -f"
            exit 1
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    echo ""
}

# Function to check if demo UI is accessible
check_demo_ui() {
    print_status "Checking demo UI..."
    
    if curl -s http://localhost:${AI_ENGINE_PORT}/ >/dev/null 2>&1; then
        print_success "Demo UI is accessible"
    else
        print_error "Demo UI is not accessible"
        return 1
    fi
}


# Function to display service URLs and status
show_service_info() {
    local project_root="$1"
    
    echo ""
    echo "=========================================="
    echo "🚀 TELAR AI ENGINE DEVELOPMENT ENVIRONMENT"
    echo "=========================================="
    echo ""
    echo "📍 Project Root: $project_root"
    echo ""
    echo "🌐 Service URLs:"
    echo "  • Demo UI:        http://localhost:${AI_ENGINE_PORT}"
    echo "  • AI Engine API:  http://localhost:${AI_ENGINE_PORT}/api/v1"
    echo "  • Health Check:   http://localhost:${AI_ENGINE_PORT}/health"
    echo "  • Status Check:   http://localhost:${AI_ENGINE_PORT}/status"
    echo "  • Weaviate:       http://localhost:${WEAVIATE_PORT}"
    echo ""
    echo "📋 Service Status:"
    
    # Check AI Engine 
    if curl -s http://localhost:${AI_ENGINE_PORT}/health >/dev/null 2>&1; then
        echo -e "  • AI Engine:      ${GREEN}✅ Running${NC}"
    else
        echo -e "  • AI Engine:      ${RED}❌ Not responding${NC}"
    fi
    
    # Check Demo UI 
    if curl -s http://localhost:${AI_ENGINE_PORT}/ >/dev/null 2>&1; then
        echo -e "  • Demo UI:        ${GREEN}✅ Running ${NC}"
    else
        echo -e "  • Demo UI:        ${RED}❌ Not responding${NC}"
    fi
    
    # Check Weaviate
    if curl -s http://localhost:${WEAVIATE_PORT}/v1/.well-known/ready >/dev/null 2>&1; then
        echo -e "  • Weaviate:       ${GREEN}✅ Running${NC}"
    else
        echo -e "  • Weaviate:       ${RED}❌ Not responding${NC}"
    fi
    
    echo ""
    echo "🎯 Quick Commands:"
    echo "  • View logs:      docker compose logs -f"
    echo "  • Stop services:  docker compose down"
    echo "  • Restart:        docker compose restart"
    echo ""
    echo "📖 Demo Instructions:"
    echo "  1. Open http://localhost:${AI_ENGINE_PORT} in your browser"
    echo "  2. Use 'Load README.md' to ingest Telar documentation"
    echo "  3. Ask questions like 'What is Telar?' to test the RAG system"
    echo "  4. Try the Content Generation tab for conversation starters"
    echo "  5. Watch the flow diagrams animate during operations"
    echo ""
    echo "⚠️  To stop all services, run: $0 stop"
    echo "=========================================="
}

# Function to stop all services
stop_services() {
    print_status "Stopping all development services..."
    
    # Find project root
    local project_root=$(find_project_root)
    local docker_compose_dir="$project_root/apps/ai-engine/deployments/docker-compose"
    
    # Stop Docker services
    if [[ -f "$docker_compose_dir/docker-compose.yml" ]]; then
        cd "$docker_compose_dir"
        docker compose down -v
        print_success "Docker services stopped"
    fi
    
    
    print_success "All services stopped"
}

# Function to reset all data (clear Weaviate and restart fresh)
reset_services() {
    print_status "🔥 Resetting all data and services..."
    
    # Find project root
    local project_root=$(find_project_root)
    local docker_compose_dir="$project_root/apps/ai-engine/deployments/docker-compose"
    
    if [[ ! -f "$docker_compose_dir/docker-compose.yml" ]]; then
        print_error "Docker Compose file not found at $docker_compose_dir/docker-compose.yml"
        exit 1
    fi
    
    print_warning "This will completely clear all Weaviate data and restart services"
    print_warning "All ingested documents and embeddings will be lost"
    
    # Stop services and remove volumes (complete data reset)
    print_status "Clearing all data and stopping services..."
    cd "$docker_compose_dir"
    docker compose down -v
    
    
    print_success "All data cleared and services stopped"
    
    # Wait a moment for cleanup
    sleep 2
    
    # Restart everything fresh
    print_status "🚀 Starting fresh services..."
    main "start"
}


# Function to show usage
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  start    Start all development services (default)"
    echo "  stop     Stop all development services"
    echo "  restart  Restart all development services"
    echo "  reset    Clear all data and restart fresh (⚠️  destructive)"
    echo "  status   Show service status"
    echo "  help     Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  AI_ENGINE_PORT  Port for AI Engine service (default: 8000)"
    echo "  WEAVIATE_PORT   Port for Weaviate service (default: 8080)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Start all services with default ports"
    echo "  AI_ENGINE_PORT=9000 $0 start  # Start with custom AI Engine port"
    echo "  $0 stop               # Stop all services"
    echo "  $0 restart            # Restart all services"
    echo "  $0 reset              # Clear all Weaviate data and restart"
}

# Main execution
main() {
    local command="${1:-start}"
    
    case "$command" in
        "start")
            print_status "🚀 Starting Telar AI Engine Development Environment..."
            
            # Find project root
            local project_root=$(find_project_root)
            print_success "Found project root: $project_root"
            
            # Pre-flight checks
            check_docker
            
            # Start services
            start_docker_services "$project_root"
            wait_for_services
            check_demo_ui
            
            # Show status
            show_service_info "$project_root"
            ;;
        
        "stop")
            stop_services
            ;;
        
        "restart")
            print_status "🔄 Restarting all services..."
            stop_services
            sleep 2
            main "start"
            ;;
        
        "reset")
            reset_services
            ;;
        
        "status")
            local project_root=$(find_project_root)
            show_service_info "$project_root"
            ;;
        
        "help"|"-h"|"--help")
            show_usage
            ;;
        
        *)
            print_error "Unknown command: $command"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
