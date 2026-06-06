# Project Context


## Overview

This repository is primarily **Go** with 3 other languages.
Entry points: `cmd/ctxgen/main.go`.


## File Tree

```
.
├── cmd/
│   └── ctxgen/
│       └── main.go
├── internal/
│   ├── analyzer/
│   │   └── analyzer.go
│   ├── parser/
│   │   ├── deps.go
│   │   ├── parser.go
│   │   └── schema.go
│   ├── scanner/
│   │   ├── language.go
│   │   └── scanner.go
│   └── writer/
│       └── writer.go
├── README.md
├── ctxgen-context.md
├── ctxgen-pack.txt
└── go.mod
```


## Languages

- **Go**: 8 files
- **Markdown**: 2 files
- **Go Module**: 1 files
- **Text**: 1 files


## Entry Points

- `cmd/ctxgen/main.go`


## Configuration Files

- `go.mod`


## Core Modules

### cmd/ctxgen

**`cmd/ctxgen/main.go`** (Go)
  - *funcs*: `main()`, `humanSize(bytes int64) string`

### internal/analyzer

**`internal/analyzer/analyzer.go`** (Go)
  - *structs*: `RepoAnalysis`, `Module`, `ModuleFile`, `RouteEntry`, `node`, `LangCount`
  - *funcs*: `Analyze(root string, files []scanner.FileInfo, contents map[string]string) *RepoAnalysis`, `isEntryPoint(relPath, language string) bool`, `buildFileTree(files []scanner.FileInfo, root string) string`
  - *methods*: `RepoAnalysis.TopLanguages() []LangCount`, `RepoAnalysis.DetectFrameworks() []string`, `RepoAnalysis.ImportantFiles() []scanner.FileInfo`

### internal/parser

**`internal/parser/deps.go`** (Go)
  - *structs*: `Dependency`
  - *funcs*: `ParseDependencies(relPath, content string) []Dependency`, `parseGoMod(content string) []Dependency`, `parsePackageJSON(content string) []Dependency`, `parseRequirements(content string) []Dependency`, `parseCargoToml(content string) []Dependency`, `parsePyproject(content string) []Dependency`, `parseComposerJSON(content string) []Dependency`, `parsePomXML(content string) []Dependency`, `extractXMLTag(line, tag string) string`, `parseGradle(content string) []Dependency`, `parseGemfile(content string) []Dependency`

**`internal/parser/parser.go`** (Go)
  - *structs*: `Symbol`, `FileAnalysis`, `Route`
  - *funcs*: `Parse(relPath, language, content string) *FileAnalysis`, `parseGo(fa *FileAnalysis, content string)`, `parsePython(fa *FileAnalysis, content string)`, `parseTypeScript(fa *FileAnalysis, content string)`, `parseJava(fa *FileAnalysis, content string)`, `parseCSharp(fa *FileAnalysis, content string)`, `parseRust(fa *FileAnalysis, content string)`, `parsePHP(fa *FileAnalysis, content string)`, `parseRuby(fa *FileAnalysis, content string)`, `parseGeneric(fa *FileAnalysis, content string)`, `extractRoutes(content, relPath string) []Route`, `inferMethodFromContext(line string) string`, `extractSummaryComment(content string) string`

**`internal/parser/schema.go`** (Go)
  - *structs*: `SchemaInfo`, `TableInfo`
  - *funcs*: `ExtractSchema(relPath, language, content string) *SchemaInfo`, `parseSQLSchema(content string) []TableInfo`, `parseRailsMigration(content string) []TableInfo`, `parseGoStructSchema(content string) []TableInfo`, `parsePrismaSchema(content string) []TableInfo`, `parseDjangoModels(content string) []TableInfo`

### internal/scanner

**`internal/scanner/language.go`** (Go)
  > DetectLanguage maps file extensions/names to language names
  - *funcs*: `DetectLanguage(name, ext string) string`

**`internal/scanner/scanner.go`** (Go)
  - *structs*: `FileInfo`, `Scanner`, `ignoreRule`
  - *funcs*: `New(root string) (*Scanner, error)`, `isBinaryFile(path string) bool`, `ReadFile(path string) (string, error)`
  - *methods*: `Scanner.loadGitignore(dir string)`, `Scanner.isIgnoredByGitignore(relPath string, isDir bool) bool`, `Scanner.Walk() ([]FileInfo, error)`

### internal/writer

**`internal/writer/writer.go`** (Go)
  - *structs*: `Options`, `mdWriter`
  - *funcs*: `WritePack(files []scanner.FileInfo, contents map[string]string, opts Options) (string, error)`, `WriteContext(ra *analyzer.RepoAnalysis, contents map[string]string, opts Options) (string, error)`, `WriteHybrid(ra *analyzer.RepoAnalysis, contents map[string]string, opts Options) (string, error)`, `humanSize(bytes int64) string`, `humanNum(n int) string`, `EstimateTokens(s string) int`
  - *methods*: `mdWriter.write(s string)`, `mdWriter.line(s string)`, `mdWriter.nl()`, `mdWriter.h1(s string)`, `mdWriter.h2(s string)`, `mdWriter.h3(s string)`, `mdWriter.code(lang, content string)`


## API Routes

- `GET /path` — `ctxgen-pack.txt`
- `/PATH 'GET'` — `ctxgen-pack.txt`
- `GET route/` — `ctxgen-pack.txt`
- `GET path` — `ctxgen-pack.txt`


## Import Relationships

Key module dependencies (top-level files):

- `cmd/ctxgen/main.go` → `flag`, `fmt`, `os`, `path/filepath`, `strings`, `time`
- `internal/analyzer/analyzer.go` → `path/filepath`, `sort`, `strings`
- `internal/parser/deps.go` → `encoding/json`, `path/filepath`, `strings`
- `internal/parser/parser.go` → `bufio`, `path/filepath`, `regexp`, `strings`
- `internal/parser/schema.go` → `regexp`, `strings`
- `internal/scanner/language.go` → `strings`
- `internal/scanner/scanner.go` → `bufio`, `os`, `path/filepath`, `sort`, `strings`, `unicode`
- `internal/writer/writer.go` → `fmt`, `os`, `path/filepath`, `sort`, `strings`, `time`, `unicode/utf8`


## Notes

- 1 binary/non-text files were excluded from analysis.


## Token Estimate

```
Files processed:  13
Total source size: 139.7 KB
Original estimate: ~35k tokens
Context output:   ~1k tokens
Estimated reduction: 96.2%
```


## Important File Contents

*Full contents of 1 key files:*

### cmd/ctxgen/main.go

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ctxgen/ctxgen/internal/analyzer"
	"github.com/ctxgen/ctxgen/internal/scanner"
	"github.com/ctxgen/ctxgen/internal/writer"
)

const version = "1.0.0"

const usage = `ctxgen - Repository Context Generator for AI

USAGE:
  ctxgen <command> [options] <directory>

COMMANDS:
  pack      Generate a single text file with all source files concatenated
  context   Generate a compact Markdown summary (recommended)
  hybrid    Generate context summary + full contents of key files

OPTIONS:
  -o <path>         Output file path (default: ctxgen-pack.txt / ctxgen-context.md / ctxgen-hybrid.md)
  -t                Include timestamp in output
  -max-size <bytes> Skip files larger than this size (default: 500000)
  -v                Show version

EXAMPLES:
  ctxgen context .
  ctxgen pack ./myproject -o output.txt
  ctxgen hybrid . -o context.md -t
  ctxgen context /path/to/repo -o ai-context.md
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "-v" || cmd == "--version" || cmd == "version" {
		fmt.Printf("ctxgen version %s\n", version)
		os.Exit(0)
	}
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		fmt.Print(usage)
		os.Exit(0)
	}

	switch cmd {
	case "pack", "context", "hybrid":
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'ctxgen help' for usage.\n", cmd)
		os.Exit(1)
	}

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	outputPath := fs.String("o", "", "Output file path")
	timestamp := fs.Bool("t", false, "Include timestamp in output")
	maxSize := fs.Int64("max-size", 500000, "Maximum file size in bytes")
	_ = fs.Bool("v", false, "Show version") // consumed above

	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	// Validate directory
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access directory '%s': %v\n", dir, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: '%s' is not a directory\n", dir)
		os.Exit(1)
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	start := time.Now()
	fmt.Fprintf(os.Stderr, "ctxgen %s: scanning %s...\n", cmd, abs)

	// Scan
	s, err := scanner.New(abs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating scanner: %v\n", err)
		os.Exit(1)
	}

	files, err := s.Walk()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Read file contents
	contents := make(map[string]string, len(files))
	skipped := 0
	for _, f := range files {
		if f.IsBinary {
			continue
		}
		if *maxSize > 0 && f.Size > *maxSize {
			skipped++
			continue
		}
		content, err := scanner.ReadFile(f.Path)
		if err != nil {
			continue
		}
		contents[f.RelPath] = content
	}

	fmt.Fprintf(os.Stderr, "Found %d files (%d skipped: binary or too large)\n",
		len(files), skipped)

	opts := writer.Options{
		OutputPath:  *outputPath,
		Timestamp:   *timestamp,
		MaxFileSize: *maxSize,
	}

	var outPath string

	switch cmd {
	case "pack":
		if opts.OutputPath == "" {
			opts.OutputPath = "ctxgen-pack.txt"
		}
		fmt.Fprintln(os.Stderr, "Writing pack file...")
		outPath, err = writer.WritePack(files, contents, opts)

	case "context":
		if opts.OutputPath == "" {
			opts.OutputPath = "ctxgen-context.md"
		}
		fmt.Fprintln(os.Stderr, "Analyzing repository...")
		ra := analyzer.Analyze(abs, files, contents)
		fmt.Fprintf(os.Stderr, "Detected %d languages, %d dependencies, %d routes\n",
			len(ra.Languages), len(ra.Dependencies), len(ra.AllRoutes))
		outPath, err = writer.WriteContext(ra, contents, opts)

	case "hybrid":
		if opts.OutputPath == "" {
			opts.OutputPath = "ctxgen-hybrid.md"
		}
		fmt.Fprintln(os.Stderr, "Analyzing repository...")
		ra := analyzer.Analyze(abs, files, contents)
		outPath, err = writer.WriteHybrid(ra, contents, opts)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start).Round(time.Millisecond)

	// Print summary
	outputInfo, _ := os.Stat(outPath)
	outputSize := int64(0)
	if outputInfo != nil {
		outputSize = outputInfo.Size()
	}

	fmt.Fprintln(os.Stderr, strings.Repeat("─", 50))
	fmt.Fprintf(os.Stderr, "✓ Done in %s\n", elapsed)
	fmt.Fprintf(os.Stderr, "  Output:  %s\n", outPath)
	fmt.Fprintf(os.Stderr, "  Size:    %s\n", humanSize(outputSize))
	fmt.Fprintf(os.Stderr, "  ~Tokens: %d\n", int(float64(outputSize)*0.25))
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

