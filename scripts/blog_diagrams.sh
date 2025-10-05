#!/bin/bash
set -euo pipefail

readonly BIN="./bin/stacktower"
readonly OUTPUT_DIR="./blogpost/plots"

# Consistent dimensions for all diagrams
readonly WIDTH=200
readonly HEIGHT=200

main() {
    check_prerequisites
    mkdir -p "$OUTPUT_DIR"

    echo "=== Generating Blog Diagrams ==="

    generate_simple_comparison
    generate_transformation_examples
    generate_pqtree_examples
    generate_hero_image

    echo ""
    echo "=== Done ==="
}

check_prerequisites() {
    if [[ ! -f "$BIN" ]]; then
        echo "Binary not found at $BIN. Run 'make build' first" >&2
        exit 1
    fi
}

generate_simple_comparison() {
    echo ""
    echo "--- Simple DAG Comparison ---"

    # Create simple DAG JSON: App → Auth, Cache → Core
    local json='{"nodes":[{"id":"App"},{"id":"Auth"},{"id":"Cache"},{"id":"Core"}],"edges":[{"from":"App","to":"Auth"},{"from":"App","to":"Cache"},{"from":"Auth","to":"Core"},{"from":"Cache","to":"Core"}]}'
    local tmp=$(mktemp)
    echo "$json" > "$tmp"

    echo -n "  simple_dag.svg... "
    $BIN render "$tmp" -t nodelink --width $WIDTH --height $HEIGHT -o "$OUTPUT_DIR/simple_dag.svg" 2>/dev/null
    echo "OK"

    echo -n "  simple_tower.svg... "
    $BIN render "$tmp" -t tower --width $WIDTH --height $HEIGHT -o "$OUTPUT_DIR/simple_tower.svg" 2>/dev/null
    echo "OK"

    rm "$tmp"
}

generate_transformation_examples() {
    echo ""
    echo "--- Transformation Examples ---"

    # Transitive before (with redundant edge): App→Auth→Core, plus redundant App→Core
    render_example "transitive_before" \
        '{"nodes":[{"id":"App"},{"id":"Auth"},{"id":"Core"}],"edges":[{"from":"App","to":"Auth"},{"from":"Auth","to":"Core"},{"from":"App","to":"Core"}]}' \
        "false" "false"

    # Transitive after: App→Auth→Core (redundant edge removed)
    render_example "transitive_after" \
        '{"nodes":[{"id":"App"},{"id":"Auth"},{"id":"Core"}],"edges":[{"from":"App","to":"Auth"},{"from":"Auth","to":"Core"}]}' \
        "true" "true"

    # Dummies before - API→Auth→DB chain, plus API→Cache long edge to DB level
    # Use --merge to show the gap (Cache's support merged, showing skip)
    render_example_merged "dummies_before" \
        '{"nodes":[{"id":"API"},{"id":"Auth"},{"id":"DB"},{"id":"Cache"}],"edges":[{"from":"API","to":"Auth"},{"from":"Auth","to":"DB"},{"from":"API","to":"Cache"}]}'

    # Dummies after - explicit dummy node bridges the gap for Cache
    render_example "dummies_after" \
        '{"nodes":[{"id":"API","row":0},{"id":"Auth","row":1},{"id":"Cache_1","row":1,"kind":"subdivider"},{"id":"DB","row":2},{"id":"Cache","row":2}],"edges":[{"from":"API","to":"Auth"},{"from":"Auth","to":"DB"},{"from":"API","to":"Cache_1"},{"from":"Cache_1","to":"Cache"}]}' \
        "false" "true"

    # Separators before - Auth and API both depend on Logging and Metrics
    render_example "separators_before" \
        '{"nodes":[{"id":"Auth"},{"id":"API"},{"id":"Logging"},{"id":"Metrics"}],"edges":[{"from":"Auth","to":"Logging"},{"from":"Auth","to":"Metrics"},{"from":"API","to":"Logging"},{"from":"API","to":"Metrics"}]}' \
        "false" "false"

    # Separators after (with separator plate)
    render_example "separators_after" \
        '{"nodes":[{"id":"Auth"},{"id":"API"},{"id":"Logging"},{"id":"Metrics"}],"edges":[{"from":"Auth","to":"Logging"},{"from":"Auth","to":"Metrics"},{"from":"API","to":"Logging"},{"from":"API","to":"Metrics"}]}' \
        "true" "true"
}

render_example() {
    local name=$1
    local json=$2
    local normalize=$3
    local with_tower=${4:-false}
    local tmp=$(mktemp)
    echo "$json" > "$tmp"

    echo -n "  ${name}.svg... "
    $BIN render "$tmp" -t nodelink --width $WIDTH --height $HEIGHT --normalize="$normalize" -o "$OUTPUT_DIR/${name}.svg" 2>/dev/null
    echo "OK"

    if [[ "$with_tower" == "true" ]]; then
        echo -n "  ${name}_tower.svg... "
        $BIN render "$tmp" -t tower --width $WIDTH --height $HEIGHT --normalize="$normalize" -o "$OUTPUT_DIR/${name}_tower.svg" 2>/dev/null
        echo "OK"
    fi

    rm "$tmp"
}

# Render with --merge flag (for showing gap in "before" examples)
render_example_merged() {
    local name=$1
    local json=$2
    local tmp=$(mktemp)
    echo "$json" > "$tmp"

    echo -n "  ${name}.svg... "
    $BIN render "$tmp" -t nodelink --width $WIDTH --height $HEIGHT --normalize=false -o "$OUTPUT_DIR/${name}.svg" 2>/dev/null
    echo "OK"

    echo -n "  ${name}_tower.svg... "
    $BIN render "$tmp" -t tower --width $WIDTH --height $HEIGHT --normalize=true --merge -o "$OUTPUT_DIR/${name}_tower.svg" 2>/dev/null
    echo "OK"

    rm "$tmp"
}

generate_pqtree_examples() {
    echo ""
    echo "--- PQ-Tree Examples ---"

    # Before constraint (universal)
    echo -n "  pq_tree_before.svg... "
    $BIN pqtree --labels Logging,Auth,Caching -o "$OUTPUT_DIR/pq_tree_before.svg" 2>/dev/null
    echo "OK"

    # After constraint (Auth and Caching adjacent)
    echo -n "  pq_tree_after.svg... "
    $BIN pqtree --labels Logging,Auth,Caching -o "$OUTPUT_DIR/pq_tree_after.svg" 1,2 2>/dev/null
    echo "OK"
}

generate_hero_image() {
    echo ""
    echo "--- Hero Image ---"

    local cache_file="./blogpost/cache/python_fastapi.json"
    local output_file="$OUTPUT_DIR/hero_tower.svg"

    if [[ ! -f "$cache_file" ]]; then
        echo "  Cache file not found: $cache_file"
        echo "  Run blog_showcase.sh first to generate cache"
        return 1
    fi

    echo -n "  hero_tower.svg (fastapi, no nebraska)... "
    $BIN render "$cache_file" \
        -t tower \
        --style handdrawn \
        --width 350 \
        --height 500 \
        --ordering optimal \
        --merge \
        --randomize \
        -o "$output_file" 2>/dev/null
    echo "OK"
}

main "$@"

