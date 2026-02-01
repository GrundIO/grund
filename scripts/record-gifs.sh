#!/bin/bash
# record-gifs.sh - Record all GIFs with automatic setup/teardown
#
# Usage:
#   ./scripts/record-gifs.sh          # Record all GIFs
#   ./scripts/record-gifs.sh status   # Record single GIF
#
# Prerequisites:
#   - Docker running
#   - grund installed (make install)
#   - VHS installed (brew install charmbracelet/tap/vhs)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RECORDINGS_DIR="$SCRIPT_DIR/recordings"
OUTPUT_DIR="$PROJECT_ROOT/docs/assets"
DEMO_CONFIG="$PROJECT_ROOT/docs/demo/services.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Docker
    if ! docker info &>/dev/null; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    # Check grund
    if ! command -v grund &>/dev/null; then
        log_error "grund not found. Run 'make install' first."
        exit 1
    fi

    # Check VHS
    if ! command -v vhs &>/dev/null; then
        log_error "VHS not found. Install with: brew install charmbracelet/tap/vhs"
        exit 1
    fi

    log_info "All prerequisites met."
}

# Clean up any running services
cleanup() {
    log_info "Cleaning up services..."
    export GRUND_CONFIG="$DEMO_CONFIG"
    grund down &>/dev/null || true
}

# Start services needed for certain recordings
start_services() {
    log_info "Starting demo services..."
    export GRUND_CONFIG="$DEMO_CONFIG"
    grund up user-service --detach &>/dev/null || true
    sleep 5  # Wait for services to be ready
}

# Record a single tape
record_tape() {
    local tape_name=$1
    local tape_file="$RECORDINGS_DIR/${tape_name}.tape"

    if [[ ! -f "$tape_file" ]]; then
        log_error "Tape file not found: $tape_file"
        return 1
    fi

    log_info "Recording $tape_name..."

    # Run VHS from project root
    cd "$PROJECT_ROOT"
    if vhs "$tape_file"; then
        log_info "✓ $tape_name recorded successfully"
    else
        log_warn "✗ $tape_name failed"
        return 1
    fi
}

# Define recording order and dependencies
# Format: "tape_name:needs_services"
TAPES=(
    # No services needed
    "grund-init:no"
    "grund-service-init:no"
    "grund-service-add:no"
    "grund-config-show:no"

    # Services needed
    "grund-status:yes"
    "grund-logs:yes"
    "grund-up:yes"
    "grund-infra-only:yes"
    "grund-full-workflow:yes"
)

record_all() {
    local services_started=false

    mkdir -p "$OUTPUT_DIR"

    for entry in "${TAPES[@]}"; do
        local tape_name="${entry%%:*}"
        local needs_services="${entry##*:}"

        # Start services if needed and not already started
        if [[ "$needs_services" == "yes" && "$services_started" == "false" ]]; then
            cleanup
            start_services
            services_started=true
        fi

        record_tape "$tape_name" || true
    done

    # Final cleanup
    cleanup

    log_info "All recordings complete! GIFs saved to $OUTPUT_DIR"
    ls -la "$OUTPUT_DIR"/*.gif 2>/dev/null || true
}

record_single() {
    local tape_name=$1

    mkdir -p "$OUTPUT_DIR"

    # Check if this tape needs services
    local needs_services="no"
    for entry in "${TAPES[@]}"; do
        if [[ "${entry%%:*}" == "$tape_name" ]]; then
            needs_services="${entry##*:}"
            break
        fi
    done

    if [[ "$needs_services" == "yes" ]]; then
        cleanup
        start_services
    fi

    record_tape "$tape_name"

    if [[ "$needs_services" == "yes" ]]; then
        cleanup
    fi
}

# Main
main() {
    check_prerequisites

    if [[ $# -eq 0 ]]; then
        # Record all
        record_all
    else
        # Record single tape
        record_single "$1"
    fi
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

main "$@"
