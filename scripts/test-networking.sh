#!/bin/bash
# Comprehensive networking test script for Spitfire
# Tests various networking scenarios and validates functionality

set -e

SPITFIRE_BIN="./bin/spitfire"
FAILED_TESTS=0
TOTAL_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

run_test() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_pattern="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log "Running: $test_name"
    
    if output=$(eval "$test_cmd" 2>&1); then
        if [[ -n "$expected_pattern" ]] && ! echo "$output" | grep -q "$expected_pattern"; then
            fail "$test_name - Expected pattern '$expected_pattern' not found"
            echo "Output: $output"
        else
            success "$test_name"
        fi
    else
        fail "$test_name - Command failed"
        echo "Output: $output"
    fi
}

cleanup_vms() {
    log "Cleaning up any existing VMs..."
    $SPITFIRE_BIN vm down -a 2>/dev/null || true
    sleep 2
}

# Ensure we have a clean state
cleanup_vms

echo "=== Spitfire Networking Test Suite ==="
echo

# Test 1: Basic single VM start
log "=== Test 1: Basic Single VM ==="
run_test "Single VM startup" \
    "$SPITFIRE_BIN vm up --debug -f examples/networking-test/spitfire.yaml" \
    "Firecracker VM started"

# Check if VM is running
run_test "VM status check" \
    "$SPITFIRE_BIN vm ps" \
    "web-with-net"

# Check processes
log "Checking for networking processes..."
if ps aux | grep -v grep | grep -q "pasta\|slirp4netns"; then
    success "Networking processes found"
else
    fail "No networking processes found"
fi

# Test 2: VM cleanup
log "=== Test 2: VM Cleanup ==="
run_test "VM shutdown" \
    "$SPITFIRE_BIN vm down" \
    ""

sleep 2

# Verify cleanup
if ps aux | grep -v grep | grep -q "firecracker.*web-with-net"; then
    fail "VM process still running after shutdown"
else
    success "VM process properly cleaned up"
fi

# Test 3: Multi-VM scenario
log "=== Test 3: Multi-VM Scenario ==="
run_test "Multi-VM startup" \
    "$SPITFIRE_BIN vm up --debug -f examples/multi-vm-test/spitfire.yaml" \
    "Firecracker VM started"

# Check all VMs are running
sleep 3
vm_count=$(ps aux | grep -c "firecracker.*\.sock" || true)
if [[ $vm_count -ge 3 ]]; then
    success "Multiple VMs started (found $vm_count)"
else
    fail "Expected 3+ VMs, found $vm_count"
fi

# Test 4: Port conflicts (should be handled gracefully)
log "=== Test 4: Resource Management ==="
run_test "VM process listing" \
    "$SPITFIRE_BIN vm ps" \
    "web-1"

# Test 5: Cleanup multi-VM
run_test "Multi-VM shutdown" \
    "$SPITFIRE_BIN vm down -a" \
    ""

sleep 3

# Test 6: Backend-specific testing (if tools are available)
if command -v pasta >/dev/null 2>&1; then
    log "=== Test 6: Backend-Specific Testing ==="
    run_test "Backend-specific VM startup" \
        "$SPITFIRE_BIN vm up --debug -f examples/backend-test/spitfire.yaml" \
        "Firecracker VM started"
    
    $SPITFIRE_BIN vm down -a 2>/dev/null || true
    sleep 2
else
    log "Skipping backend-specific tests - pasta not available"
fi

# Test 7: Error handling
log "=== Test 7: Error Handling ==="
# This should fail gracefully with invalid config
if $SPITFIRE_BIN vm up -f /nonexistent/file.yaml 2>/dev/null; then
    fail "Should have failed with non-existent config file"
else
    success "Properly handled invalid config file"
fi

# Final cleanup
cleanup_vms

echo
echo "=== Test Results ==="
echo "Total tests: $TOTAL_TESTS"
echo "Failed tests: $FAILED_TESTS"
echo "Passed tests: $((TOTAL_TESTS - FAILED_TESTS))"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi