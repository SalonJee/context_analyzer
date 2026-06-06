# ctxgen — Repository Context Generator for AI

`ctxgen` is a Go CLI tool that analyzes a software repository and generates a compact, AI-friendly context file. Instead of pasting raw source files into an AI chat (expensive in tokens), `ctxgen` extracts what matters: structure, symbols, routes, schemas, and dependencies — then wraps it in a clean Markdown document.

## Install

```bash
# Build from source (requires Go 1.21+)
git clone <repo>
cd ctxgen
go build -o ctxgen ./cmd/ctxgen

# Or install globally
go install ./cmd/ctxgen
```

## Usage

### Context mode (recommended)

```bash
ctxgen context .
```

Generates `ctxgen-context.md` — a compact Markdown summary of the entire repository.

**Example output sections:**
- Project overview (primary language, frameworks)
- File tree
- Language breakdown
- Dependencies (from `go.mod`, `package.json`, `requirements.txt`, etc.)
- Entry points
- Core modules with extracted symbols (functions, structs, classes, interfaces)
- API routes (REST endpoints detected from code)
- Database schema (SQL, migrations, ORM models, Prisma)
- Import relationships
- Token reduction estimate

### Pack mode

```bash
ctxgen pack .
```

Generates `ctxgen-pack.txt` — all source files concatenated into one file with headers. Use when you need full source content.

### Hybrid mode

```bash
ctxgen hybrid .
```

Generates `ctxgen-hybrid.md` — the full context summary **plus** inline content of the most important files (entry points, main config, routers, etc.).

## Options

```
-o <path>         Output file path
-t                Include timestamp in output header
-max-size <bytes> Skip files larger than this (default: 500000)
-v                Show version
```

## Examples

```bash
# Analyze current directory, save to custom path
ctxgen context . -o ai-context.md

# Pack a specific project
ctxgen pack ~/projects/myapp -o myapp-pack.txt

# Hybrid with timestamp
ctxgen hybrid . -o context.md -t

# Skip files larger than 100KB
ctxgen context . -max-size 100000
```

## What it analyzes

| Feature | Supported |
|---|---|
| `.gitignore` respecting | ✓ |
| Binary file detection | ✓ |
| Language detection | 40+ languages |
| **Symbol extraction** | Go, Python, TypeScript, JavaScript, Java, C#, Rust, PHP, Ruby |
| **Dependency parsing** | go.mod, package.json, requirements.txt, Cargo.toml, pom.xml, build.gradle, pyproject.toml, composer.json, Gemfile |
| **Route detection** | Gin, Echo, Express, Flask, Django, Spring, ASP.NET |
| **Schema detection** | SQL CREATE TABLE, Rails migrations, GORM structs, Prisma schema, Django models |
| Framework detection | React, Next.js, Vue, Express, Django, Flask, FastAPI, Spring, Rails, Gin, and more |

## Architecture

```
ctxgen/
├── cmd/ctxgen/         # CLI entrypoint
└── internal/
    ├── scanner/        # Directory walking, gitignore, language detection
    ├── parser/         # Symbol extraction, dependency parsing, schema detection
    ├── analyzer/       # Repository-wide aggregation, module grouping
    └── writer/         # Markdown/text output formatting
```

## Token reduction

Running `ctxgen context` on a typical project:

```
Files processed:  248
Total source size: 1.6 MB
Original estimate: ~420k tokens
Context output:   ~8.9k tokens
Estimated reduction: 97.9%
```

The goal is not compression — it's **repository intelligence extraction**: surfacing the information an AI needs to understand a codebase with far fewer tokens than raw source.

## Ignored by default

- `node_modules/`, `dist/`, `build/`, `target/`, `vendor/`, `.git/`
- `__pycache__/`, `.venv/`, `venv/`, `.cache/`
- `.next/`, `.nuxt/`, `.svelte-kit/`
- Binary files (images, archives, executables, compiled files)
- Files matched by `.gitignore`
