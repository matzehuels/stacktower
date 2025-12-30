#!/bin/bash
# API End-to-End Tests
#
# This script tests the stacktowerd API by:
#   1. Building the server binary
#   2. Starting it in standalone --no-auth mode
#   3. Testing all API endpoints: parse, layout, visualize, render
#   4. Downloading artifacts to organized test folders
#
# Usage:
#   ./scripts/test_api_e2e.sh         # Run all API tests
#   ./scripts/test_api_e2e.sh quick   # Run quick smoke test only
#
# Output structure:
#   output/api-e2e/
#     ├── layout/          # Layout test results
#     ├── parse/           # Parse test results  
#     ├── visualize/       # Visualize test results (from layout data)
#     └── render/          # Full render pipeline results

set -euo pipefail

readonly SERVER_BIN="./bin/stacktowerd"
readonly OUTPUT_DIR="./output/api-e2e"
readonly SERVER_PORT="${PORT:-8080}"
readonly SERVER_URL="http://localhost:${SERVER_PORT}"
readonly API_URL="${SERVER_URL}/api/v1"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly NC='\033[0m' # No Color

# Server PID (for cleanup)
SERVER_PID=""

main() {
    local mode="${1:-all}"

    echo "=== Stacktower API E2E Tests ==="
    echo ""

    setup
    trap cleanup EXIT

    case "$mode" in
        quick)
            run_quick_tests
            ;;
        all)
            run_quick_tests
            run_layout_tests
            run_visualize_tests
            if [[ "${SKIP_NETWORK:-}" != "true" ]]; then
                run_parse_tests
                run_render_tests
            else
                echo ""
                echo "--- Skipping network tests (SKIP_NETWORK=true) ---"
            fi
            ;;
        *)
            echo "Usage: $0 [quick|all]"
            exit 1
            ;;
    esac

    echo ""
    echo -e "${GREEN}=== All tests passed ===${NC}"
    echo "Artifacts saved to: $OUTPUT_DIR"
}

setup() {
    # Build server if needed
    if [[ ! -f "$SERVER_BIN" ]] || [[ "$SERVER_BIN" -ot ./cmd/stacktowerd/main.go ]]; then
        echo "Building stacktowerd..."
        go build -o "$SERVER_BIN" ./cmd/stacktowerd
    fi

    # Check prerequisites
    if ! command -v jq >/dev/null 2>&1; then
        fail "jq is required but not installed"
    fi
    if ! command -v curl >/dev/null 2>&1; then
        fail "curl is required but not installed"
    fi

    # Create output directories
    mkdir -p "$OUTPUT_DIR"/{layout,parse,visualize,render}

    # Kill any existing server on our port
    if lsof -ti:${SERVER_PORT} >/dev/null 2>&1; then
        echo "Killing existing process on port ${SERVER_PORT}..."
        kill $(lsof -ti:${SERVER_PORT}) 2>/dev/null || true
        sleep 1
    fi

    # Start server
    echo "Starting stacktowerd on port ${SERVER_PORT}..."
    $SERVER_BIN standalone --no-auth --port "$SERVER_PORT" > "$OUTPUT_DIR/server.log" 2>&1 &
    SERVER_PID=$!

    # Wait for server to be ready
    echo -n "Waiting for server..."
    for i in {1..30}; do
        if curl -s "${SERVER_URL}/health" >/dev/null 2>&1; then
            echo " ready!"
            return 0
        fi
        echo -n "."
        sleep 0.5
    done

    echo ""
    fail "Server failed to start. Check $OUTPUT_DIR/server.log"
}

cleanup() {
    if [[ -n "$SERVER_PID" ]]; then
        echo ""
        echo "Stopping server (PID: $SERVER_PID)..."
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
}

run_quick_tests() {
    echo ""
    echo "--- Quick Smoke Tests ---"

    # Health check
    test_endpoint "Health check" "GET" "/health" 200

    # Auth (should return mock local user)
    test_endpoint "Auth me" "GET" "/api/v1/auth/me" 200
    
    # History (empty initially)
    test_endpoint "History" "GET" "/api/v1/history" 200
}

run_layout_tests() {
    echo ""
    echo "--- Layout Tests (POST /api/v1/layout) ---"

    # Test inline graph layout
    test_layout "simple-chain" \
        '{"nodes":[{"id":"a"},{"id":"b"},{"id":"c"}],"edges":[{"from":"a","to":"b"},{"from":"b","to":"c"}]}'

    # Test diamond graph
    test_layout "diamond" \
        '{"nodes":[{"id":"root"},{"id":"left"},{"id":"right"},{"id":"bottom"}],"edges":[{"from":"root","to":"left"},{"from":"root","to":"right"},{"from":"left","to":"bottom"},{"from":"right","to":"bottom"}]}'

    # Test fan-out graph
    test_layout "fan-out" \
        '{"nodes":[{"id":"hub"},{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"}],"edges":[{"from":"hub","to":"a"},{"from":"hub","to":"b"},{"from":"hub","to":"c"},{"from":"hub","to":"d"}]}'
}

run_visualize_tests() {
    echo ""
    echo "--- Visualize Tests (POST /api/v1/visualize) ---"
    echo "    (recreates images from layout data)"

    # Test all formats at once for tower
    test_visualize "simple-chain" "tower" "svg,png,pdf"
    
    # Test all formats at once for nodelink
    test_visualize "simple-chain" "nodelink" "svg,png,pdf"

    # Test other graphs
    test_visualize "diamond" "tower" "svg,png"
    test_visualize "fan-out" "tower" "svg"
}

run_parse_tests() {
    echo ""
    echo "--- Parse Tests (POST /api/v1/parse) ---"

    test_parse "python" "click"
    test_parse "python" "requests"
    test_parse "rust" "serde"
}

run_render_tests() {
    echo ""
    echo "--- Render Tests (POST /api/v1/render) ---"
    echo "    (full pipeline: parse → layout → visualize)"

    # Test all formats at once
    test_render "python" "click" "svg,png,pdf"
    test_render "python" "requests" "svg,png"
    test_render "rust" "serde" "svg,png,pdf"
}

# =============================================================================
# Test Functions
# =============================================================================

test_endpoint() {
    local name=$1
    local method=$2
    local path=$3
    local expected_status=$4

    echo -n "  $name... "

    local response_file="$OUTPUT_DIR/response.json"
    local status
    status=$(curl -s -o "$response_file" -w "%{http_code}" -X "$method" "${SERVER_URL}${path}")

    if [[ "$status" != "$expected_status" ]]; then
        echo -e "${RED}FAIL${NC} (expected $expected_status, got $status)"
        cat "$response_file" 2>/dev/null || true
        fail "Endpoint test failed"
    fi

    echo -e "${GREEN}OK${NC}"
}

test_layout() {
    local name=$1
    local graph_json=$2

    echo -n "  $name... "

    local out_dir="$OUTPUT_DIR/layout/$name"
    mkdir -p "$out_dir"
    
    local response_file="$out_dir/response.json"
    # Graph must be passed as base64-encoded JSON (Go []byte expects base64 in JSON)
    local graph_base64
    graph_base64=$(echo -n "$graph_json" | base64 | tr -d '\n')
    local payload="{\"graph\":\"${graph_base64}\"}"

    local status
    status=$(curl -s -o "$response_file" -w "%{http_code}" \
        -X POST "${API_URL}/layout" \
        -H "Content-Type: application/json" \
        -d "$payload")

    # Accept both 200 (sync) and 202 (async)
    if [[ "$status" != "200" && "$status" != "202" ]]; then
        echo -e "${RED}FAIL${NC} (status $status)"
        cat "$response_file" 2>/dev/null || true
        fail "Layout test failed"
    fi

    # Handle async response
    local layout_status
    layout_status=$(jq -r '.status' "$response_file")

    if [[ "$layout_status" == "pending" ]]; then
        local job_id
        job_id=$(jq -r '.job_id' "$response_file")
        echo -n "(job: ${job_id:0:8}...) "

        # Poll for completion
        for i in {1..30}; do
            sleep 0.5
            status=$(curl -s -o "$response_file" -w "%{http_code}" "${API_URL}/jobs/${job_id}")
            layout_status=$(jq -r '.status' "$response_file")

            if [[ "$layout_status" == "completed" ]]; then
                break
            elif [[ "$layout_status" == "failed" ]]; then
                echo -e "${RED}FAIL${NC}"
                jq '.error' "$response_file"
                fail "Layout job failed"
            fi
        done

        if [[ "$layout_status" != "completed" ]]; then
            echo -e "${RED}TIMEOUT${NC}"
            fail "Layout job timed out"
        fi
    fi

    # Extract and save layout_data for use in visualize tests
    jq '.result.layout_data // .layout_data' "$response_file" > "$out_dir/layout_data.json"
    
    # Save node/edge counts
    local node_count edge_count
    node_count=$(jq -r '.result.node_count // .node_count // 0' "$response_file")
    edge_count=$(jq -r '.result.edge_count // .edge_count // 0' "$response_file")

    echo -e "${GREEN}OK${NC} ($node_count nodes, $edge_count edges)"
}

test_visualize() {
    local name=$1
    local viz_type=$2
    local formats_csv=$3  # comma-separated formats like "svg,png,pdf"

    echo -n "  $name ($viz_type: $formats_csv)... "

    local layout_file="$OUTPUT_DIR/layout/$name/layout_data.json"
    if [[ ! -f "$layout_file" ]]; then
        echo -e "${YELLOW}SKIP${NC} (no layout data from previous test)"
        return 0
    fi

    local out_dir="$OUTPUT_DIR/visualize/$name/$viz_type"
    mkdir -p "$out_dir"
    
    local response_file="$out_dir/response.json"
    
    # Read layout data and base64 encode it
    local layout_base64
    layout_base64=$(cat "$layout_file" | base64 | tr -d '\n')
    
    # Convert comma-separated formats to JSON array
    local formats_json
    formats_json=$(echo "$formats_csv" | sed 's/,/","/g' | sed 's/^/["/' | sed 's/$/"]/')
    local payload="{\"layout\":\"${layout_base64}\",\"formats\":${formats_json},\"viz_type\":\"${viz_type}\"}"

    local status
    status=$(curl -s -o "$response_file" -w "%{http_code}" \
        -X POST "${API_URL}/visualize" \
        -H "Content-Type: application/json" \
        -d "$payload")

    if [[ "$status" != "200" ]]; then
        echo -e "${RED}FAIL${NC} (status $status)"
        cat "$response_file" 2>/dev/null || true
        fail "Visualize test failed"
    fi

    # Extract and save each artifact
    local artifact_count=0
    for format in ${formats_csv//,/ }; do
        local artifact_data
        artifact_data=$(jq -r ".artifacts.${format} // empty" "$response_file")
        
        if [[ -n "$artifact_data" ]]; then
            echo "$artifact_data" | base64 -d > "$out_dir/${name}.${format}"
            if [[ -s "$out_dir/${name}.${format}" ]]; then
                artifact_count=$((artifact_count + 1))
            fi
        fi
    done

    if [[ "$artifact_count" -eq 0 ]]; then
        echo -e "${YELLOW}WARN${NC} (no artifacts)"
        return 0
    fi

    echo -e "${GREEN}OK${NC} ($artifact_count artifacts)"
}

test_parse() {
    local language=$1
    local package=$2

    echo -n "  $language/$package... "

    local out_dir="$OUTPUT_DIR/parse/$language-$package"
    mkdir -p "$out_dir"
    
    local response_file="$out_dir/response.json"
    local payload="{\"language\":\"${language}\",\"package\":\"${package}\"}"

    local status
    status=$(curl -s -o "$response_file" -w "%{http_code}" \
        -X POST "${API_URL}/parse" \
        -H "Content-Type: application/json" \
        -d "$payload")

    # Accept both 200 (sync/cached) and 202 (async)
    if [[ "$status" != "200" && "$status" != "202" ]]; then
        echo -e "${RED}FAIL${NC} (status $status)"
        cat "$response_file" 2>/dev/null || true
        fail "Parse test failed"
    fi

    # Check if async (job_id) or sync (result)
    local parse_status
    parse_status=$(jq -r '.status' "$response_file")

    if [[ "$parse_status" == "pending" ]]; then
        # Poll for completion
        local job_id
        job_id=$(jq -r '.job_id' "$response_file")
        echo -n "(job: ${job_id:0:8}...) "

        for i in {1..30}; do
            sleep 0.5
            status=$(curl -s -o "$response_file" -w "%{http_code}" "${API_URL}/jobs/${job_id}")
            parse_status=$(jq -r '.status' "$response_file")

            if [[ "$parse_status" == "completed" ]]; then
                break
            elif [[ "$parse_status" == "failed" ]]; then
                echo -e "${RED}FAIL${NC}"
                jq '.error' "$response_file"
                fail "Parse job failed"
            fi
        done

        if [[ "$parse_status" != "completed" ]]; then
            echo -e "${RED}TIMEOUT${NC}"
            fail "Parse job timed out"
        fi
    fi

    # Extract and save graph_data
    jq '.result.graph_data // .graph_data' "$response_file" > "$out_dir/graph_data.json"

    # Verify we got graph_data
    local node_count edge_count
    node_count=$(jq -r '.result.node_count // .node_count // 0' "$response_file")
    edge_count=$(jq -r '.result.edge_count // .edge_count // 0' "$response_file")
    
    if [[ "$node_count" -eq 0 ]]; then
        echo -e "${YELLOW}WARN${NC} (0 nodes)"
    else
        echo -e "${GREEN}OK${NC} ($node_count nodes, $edge_count edges)"
    fi
}

test_render() {
    local language=$1
    local package=$2
    local formats_csv=$3  # comma-separated formats like "svg,png,pdf"

    echo -n "  $language/$package ($formats_csv)... "

    local out_dir="$OUTPUT_DIR/render/$language-$package"
    mkdir -p "$out_dir"
    
    local response_file="$out_dir/response.json"
    
    # Convert comma-separated formats to JSON array
    local formats_json
    formats_json=$(echo "$formats_csv" | sed 's/,/","/g' | sed 's/^/["/' | sed 's/$/"]/')
    local payload="{\"language\":\"${language}\",\"package\":\"${package}\",\"formats\":${formats_json}}"

    local status
    status=$(curl -s -o "$response_file" -w "%{http_code}" \
        -X POST "${API_URL}/render" \
        -H "Content-Type: application/json" \
        -d "$payload")

    # Accept both 200 (sync/cached) and 202 (async)
    if [[ "$status" != "200" && "$status" != "202" ]]; then
        echo -e "${RED}FAIL${NC} (status $status)"
        cat "$response_file" 2>/dev/null || true
        fail "Render test failed"
    fi

    # Check if async (job_id) or sync (result)
    local render_status
    render_status=$(jq -r '.status' "$response_file")

    if [[ "$render_status" == "pending" ]]; then
        # Poll for completion
        local job_id
        job_id=$(jq -r '.job_id' "$response_file")
        echo -n "(job: ${job_id:0:8}...) "

        for i in {1..60}; do
            sleep 1
            status=$(curl -s -o "$response_file" -w "%{http_code}" "${API_URL}/jobs/${job_id}")
            render_status=$(jq -r '.status' "$response_file")
            
            if [[ "$render_status" == "completed" ]]; then
                break
            elif [[ "$render_status" == "failed" ]]; then
                echo -e "${RED}FAIL${NC}"
                jq '.error' "$response_file"
                fail "Render job failed"
            fi
            echo -n "."
        done

        if [[ "$render_status" != "completed" ]]; then
            echo -e "${RED}TIMEOUT${NC}"
            fail "Render job timed out"
        fi
    fi

    # Get artifact URLs from response (handle both sync and async responses)
    local result_json
    if jq -e '.result.artifacts' "$response_file" >/dev/null 2>&1; then
        result_json=$(jq '.result.artifacts' "$response_file")
    elif jq -e '.artifacts' "$response_file" >/dev/null 2>&1; then
        result_json=$(jq '.artifacts' "$response_file")
    else
        echo -e "${YELLOW}WARN${NC} (no artifacts in response)"
        return 0
    fi

    # Download each artifact
    local artifact_count=0
    for fmt in $(echo "$result_json" | jq -r 'keys[]'); do
        local artifact_url
        artifact_url=$(echo "$result_json" | jq -r ".${fmt}")
        
        if [[ "$artifact_url" == http* ]]; then
            # Direct URL
            curl -s -o "$out_dir/${package}.${fmt}" "$artifact_url"
        elif [[ "$artifact_url" == /api/* ]]; then
            # Relative URL
            curl -s -o "$out_dir/${package}.${fmt}" "${SERVER_URL}${artifact_url}"
        else
            # Base64 encoded inline
            echo "$artifact_url" | base64 -d > "$out_dir/${package}.${fmt}"
        fi
        artifact_count=$((artifact_count + 1))
    done

    echo -e "${GREEN}OK${NC} ($artifact_count artifacts)"
}

fail() {
    echo -e "${RED}ERROR: $*${NC}" >&2
    exit 1
}

main "$@"
