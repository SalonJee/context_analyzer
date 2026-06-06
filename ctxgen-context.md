# Project Context


## Overview

This repository is primarily **Go** with 2 other languages.
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
└── go.mod
```


## Languages

- **Go**: 8 files
- **Go Module**: 1 files
- **Markdown**: 1 files


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

- `GET /path` — `internal/parser/parser.go`
- `/PATH 'GET'` — `internal/parser/parser.go`
- `GET route/` — `internal/parser/parser.go`
- `GET path` — `internal/parser/parser.go`


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
Files processed:  11
Total source size: 64.3 KB
Original estimate: ~16k tokens
Context output:   ~1k tokens
Estimated reduction: 91.8%
```

