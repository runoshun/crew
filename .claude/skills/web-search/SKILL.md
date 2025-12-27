---
name: web-search
description: |
  Web search Skill. Use when you need to search the web for current information, news, documentation, or any topic requiring up-to-date knowledge. Returns JSON results with ddgr or summarized text with Gemini.
---

# Web Search

Search the web using ddgr (DuckDuckGo CLI) or Gemini CLI.

## Usage

```bash
scripts/search.sh [options] <query>
```

## Options

- `--ddgr` - Use ddgr (DuckDuckGo), returns JSON
- `--gemini` - Use Gemini CLI, returns summarized text
- (default: ddgr if available, otherwise gemini) **← recommended**

## Output

- **ddgr**: JSON array of search results (title, url, abstract)
- **gemini**: Summarized text with key points

## Examples

```bash
# Auto-detect backend (recommended)
scripts/search.sh "Rust async programming"

# Explicitly use ddgr
scripts/search.sh --ddgr "Deno 2.0 features"

# Explicitly use Gemini
scripts/search.sh --gemini "Docker networking"
```

## Configuration

Set `GEMINI_MODEL` environment variable for Gemini (default: `gemini-2.5-flash`).
