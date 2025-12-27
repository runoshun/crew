#!/usr/bin/env bash
# Web search using ddgr (DuckDuckGo) or Gemini CLI
set -euo pipefail

usage() {
    echo "Usage: $0 [options] <query>"
    echo ""
    echo "Search the web using ddgr (DuckDuckGo) or Gemini CLI"
    echo ""
    echo "Options:"
    echo "  --ddgr      Use ddgr (DuckDuckGo) - returns JSON"
    echo "  --gemini    Use Gemini CLI - returns summarized text"
    echo "  (default: ddgr if available, otherwise gemini)"
    echo ""
    echo "Arguments:"
    echo "  query    Search query"
    echo ""
    echo "Environment variables:"
    echo "  GEMINI_MODEL    Model to use for Gemini (default: gemini-2.5-flash)"
    exit 1
}

search_ddgr() {
    local query="$1"
    if ! command -v ddgr &>/dev/null; then
        echo "Error: ddgr is not installed. Install with: pip install ddgr" >&2
        exit 1
    fi
    exec ddgr --json "$query"
}

search_gemini() {
    local query="$1"
    local model="${GEMINI_MODEL:-gemini-2.5-flash}"
    
    exec npx @google/gemini-cli \
        --model "$model" \
        --sandbox \
        "Search the web for \"${query}\" and provide a comprehensive summary.

Requirements:
- Provide key information and main points
- Include relevant details and context
- Format with clear sections and bullet points
- Do NOT include any URLs or source references" 2>/dev/null
}

# Parse options
backend=""
query_args=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --ddgr)
            backend="ddgr"
            shift
            ;;
        --gemini)
            backend="gemini"
            shift
            ;;
        --help|-h)
            usage
            ;;
        *)
            query_args+=("$1")
            shift
            ;;
    esac
done

if [[ ${#query_args[@]} -lt 1 ]]; then
    usage
fi

query="${query_args[*]}"

# Execute search
if [[ -n "$backend" ]]; then
    # Explicit backend specified
    case "$backend" in
        ddgr)
            search_ddgr "$query"
            ;;
        gemini)
            search_gemini "$query"
            ;;
    esac
else
    # Auto-detect: prefer ddgr if available
    if command -v ddgr &>/dev/null; then
        search_ddgr "$query"
    else
        search_gemini "$query"
    fi
fi
