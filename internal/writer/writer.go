package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ctxgen/ctxgen/internal/analyzer"
	"github.com/ctxgen/ctxgen/internal/parser"
	"github.com/ctxgen/ctxgen/internal/scanner"
)

const tokensPerChar = 0.25 // rough approximation: ~4 chars per token

// Options controls output behavior
type Options struct {
	OutputPath  string
	Timestamp   bool
	MaxFileSize int64  // skip files larger than this in bytes (0 = no limit)
}

// WritePack writes all file contents into a single text file
func WritePack(files []scanner.FileInfo, contents map[string]string, opts Options) (string, error) {
	outPath := opts.OutputPath
	if outPath == "" {
		outPath = "ctxgen-pack.txt"
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var total int
	for _, fi := range files {
		if fi.IsBinary {
			continue
		}
		content, ok := contents[fi.RelPath]
		if !ok {
			continue
		}
		fmt.Fprintf(f, "=== FILE: %s ===\n", fi.RelPath)
		fmt.Fprintln(f, content)
		fmt.Fprintln(f)
		total++
	}

	absPath, _ := filepath.Abs(outPath)
	return absPath, nil
}

// WriteContext writes the compact Markdown context file
func WriteContext(ra *analyzer.RepoAnalysis, contents map[string]string, opts Options) (string, error) {
	outPath := opts.OutputPath
	if outPath == "" {
		outPath = "ctxgen-context.md"
	}

	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := &mdWriter{f: f}

	w.h1("Project Context")

	if opts.Timestamp {
		w.line(fmt.Sprintf("> Generated: %s", time.Now().Format(time.RFC3339)))
		w.nl()
	}

	// 1. Overview
	w.h2("Overview")
	langs := ra.TopLanguages()
	frameworks := ra.DetectFrameworks()
	if len(langs) > 0 {
		primary := langs[0].Language
		if len(langs) > 1 {
			w.line(fmt.Sprintf("This repository is primarily **%s** with %d other languages.", primary, len(langs)-1))
		} else {
			w.line(fmt.Sprintf("This is a **%s** project.", primary))
		}
	}
	if len(frameworks) > 0 {
		w.line(fmt.Sprintf("Frameworks detected: %s.", strings.Join(frameworks, ", ")))
	}
	if len(ra.EntryPoints) > 0 {
		w.line(fmt.Sprintf("Entry points: `%s`.", strings.Join(ra.EntryPoints, "`, `")))
	}
	w.nl()

	// 2. File Tree
	w.h2("File Tree")
	w.code("", ra.FileTree)

	// 3. Languages
	w.h2("Languages")
	for _, lc := range langs {
		w.line(fmt.Sprintf("- **%s**: %d files", lc.Language, lc.Count))
	}
	w.nl()

	// 4. Dependencies
	if len(ra.Dependencies) > 0 {
		w.h2("Dependencies")
		// Group by source
		bySource := make(map[string][]parser.Dependency)
		for _, d := range ra.Dependencies {
			bySource[d.Source] = append(bySource[d.Source], d)
		}
		sources := make([]string, 0, len(bySource))
		for s := range bySource {
			sources = append(sources, s)
		}
		sort.Strings(sources)

		for _, src := range sources {
			deps := bySource[src]
			w.h3(src)
			// Limit to 30 most important deps per source
			if len(deps) > 30 {
				deps = deps[:30]
			}
			for _, d := range deps {
				if d.Version != "" {
					w.line(fmt.Sprintf("- `%s` %s", d.Name, d.Version))
				} else {
					w.line(fmt.Sprintf("- `%s`", d.Name))
				}
			}
			w.nl()
		}
	}

	// 5. Entry Points
	if len(ra.EntryPoints) > 0 {
		w.h2("Entry Points")
		for _, ep := range ra.EntryPoints {
			w.line(fmt.Sprintf("- `%s`", ep))
		}
		w.nl()
	}

	// 6. Configuration Files
	if len(ra.ConfigFiles) > 0 {
		w.h2("Configuration Files")
		for _, cf := range ra.ConfigFiles {
			w.line(fmt.Sprintf("- `%s`", cf))
		}
		w.nl()
	}

	// 7. Core Modules
	w.h2("Core Modules")
	symbolCount := 0
	for _, mod := range ra.Modules {
		if len(mod.Files) == 0 {
			continue
		}
		// Only show modules with actual symbols
		totalSymbols := 0
		for _, mf := range mod.Files {
			totalSymbols += len(mf.Symbols)
		}
		if totalSymbols == 0 && !mod.IsEntry {
			continue
		}

		modTitle := mod.Path
		if modTitle == "/" || modTitle == "." {
			modTitle = "(root)"
		}
		w.h3(modTitle)

		for _, mf := range mod.Files {
			if len(mf.Symbols) == 0 && mf.Summary == "" {
				continue
			}
			w.line(fmt.Sprintf("**`%s`** (%s)", mf.RelPath, mf.Language))
			if mf.Summary != "" {
				w.line(fmt.Sprintf("  > %s", mf.Summary))
			}
			if len(mf.Symbols) > 0 {
				// Group by kind
				byKind := make(map[string][]string)
				for _, sym := range mf.Symbols {
					name := sym.Name
					if sym.Sig != "" && sym.Kind == "func" || sym.Kind == "method" {
						name = name + sym.Sig
					}
					byKind[sym.Kind] = append(byKind[sym.Kind], name)
					symbolCount++
				}
				for _, kind := range []string{"struct", "class", "interface", "trait", "enum", "type", "func", "method", "module", "const"} {
					names := byKind[kind]
					if len(names) == 0 {
						continue
					}
					if len(names) > 15 {
						names = names[:15]
					}
					w.line(fmt.Sprintf("  - *%ss*: `%s`", kind, strings.Join(names, "`, `")))
				}
			}
			w.nl()
		}
	}

	// 8. API Routes
	if len(ra.AllRoutes) > 0 {
		w.h2("API Routes")
		// Deduplicate routes
		seen := make(map[string]bool)
		for _, r := range ra.AllRoutes {
			key := r.Method + " " + r.Path
			if seen[key] {
				continue
			}
			seen[key] = true
			w.line(fmt.Sprintf("- `%s %s` — `%s`", r.Method, r.Path, r.File))
		}
		w.nl()
	}

	// 9. Database Schema
	if len(ra.AllSchemas) > 0 {
		w.h2("Database Schema")
		for _, schema := range ra.AllSchemas {
			if len(schema.Tables) == 0 {
				continue
			}
			w.h3(schema.Source)
			for _, table := range schema.Tables {
				if len(table.Columns) > 0 {
					cols := table.Columns
					if len(cols) > 10 {
						cols = append(cols[:10], "...")
					}
					w.line(fmt.Sprintf("- **%s** (%s)", table.Name, strings.Join(cols, ", ")))
				} else {
					w.line(fmt.Sprintf("- **%s**", table.Name))
				}
			}
			w.nl()
		}
	}

	// 10. Import Graph (top-level only)
	if len(ra.ImportGraph) > 0 {
		w.h2("Import Relationships")
		w.line("Key module dependencies (top-level files):")
		w.nl()
		keys := make([]string, 0, len(ra.ImportGraph))
		for k := range ra.ImportGraph {
			if strings.Count(k, string(filepath.Separator)) <= 2 {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			imports := ra.ImportGraph[k]
			// Filter to local/project imports only
			var local []string
			for _, imp := range imports {
				if !strings.Contains(imp, ".") || strings.HasPrefix(imp, ".") || strings.HasPrefix(imp, "..") {
					local = append(local, "`"+imp+"`")
				}
			}
			if len(local) > 0 {
				if len(local) > 8 {
					local = local[:8]
				}
				w.line(fmt.Sprintf("- `%s` → %s", k, strings.Join(local, ", ")))
			}
		}
		w.nl()
	}

	// 11. Notes on ignored/binary files
	if ra.IgnoredCount > 0 {
		w.h2("Notes")
		w.line(fmt.Sprintf("- %d binary/non-text files were excluded from analysis.", ra.IgnoredCount))
		w.nl()
	}

	// 12. Token estimate
	w.h2("Token Estimate")
	outputStat, _ := f.Stat()
	outputSize := int64(0)
	if outputStat != nil {
		outputSize = outputStat.Size()
	}
	origTokens := int(float64(ra.TotalChars) * tokensPerChar)
	outTokens := int(float64(outputSize) * tokensPerChar)
	reduction := 0.0
	if origTokens > 0 {
		reduction = float64(origTokens-outTokens) / float64(origTokens) * 100
	}
	w.code("", fmt.Sprintf(
		"Files processed:  %d\nTotal source size: %s\nOriginal estimate: ~%s tokens\nContext output:   ~%s tokens\nEstimated reduction: %.1f%%",
		len(ra.Files),
		humanSize(ra.TotalSize),
		humanNum(origTokens),
		humanNum(outTokens),
		reduction,
	))

	absPath, _ := filepath.Abs(outPath)
	return absPath, nil
}

// WriteHybrid writes context + full content of important files
func WriteHybrid(ra *analyzer.RepoAnalysis, contents map[string]string, opts Options) (string, error) {
	outPath := opts.OutputPath
	if outPath == "" {
		outPath = "ctxgen-hybrid.md"
	}

	// First write context
	tmpOpts := opts
	tmpOpts.OutputPath = outPath
	path, err := WriteContext(ra, contents, tmpOpts)
	if err != nil {
		return "", err
	}

	// Append important files
	f, err := os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := &mdWriter{f: f}
	important := ra.ImportantFiles()

	if len(important) > 0 {
		w.h2("Important File Contents")
		w.line(fmt.Sprintf("*Full contents of %d key files:*", len(important)))
		w.nl()

		for _, fi := range important {
			content, ok := contents[fi.RelPath]
			if !ok || content == "" {
				continue
			}
			// Skip very large files
			if opts.MaxFileSize > 0 && fi.Size > opts.MaxFileSize {
				w.line(fmt.Sprintf("*`%s` — skipped (too large: %s)*", fi.RelPath, humanSize(fi.Size)))
				w.nl()
				continue
			}

			w.h3(fi.RelPath)
			lang := strings.ToLower(fi.Language)
			// Normalize lang for code fence
			langMap := map[string]string{
				"go": "go", "python": "python", "javascript": "javascript",
				"typescript": "typescript", "rust": "rust", "java": "java",
				"c#": "csharp", "c++": "cpp", "c": "c", "ruby": "ruby",
				"shell": "bash", "yaml": "yaml", "json": "json",
				"markdown": "markdown", "sql": "sql", "toml": "toml",
			}
			fence := langMap[lang]
			w.code(fence, content)
		}
	}

	return path, nil
}

// --- Markdown writer helper ---

type mdWriter struct {
	f *os.File
}

func (w *mdWriter) write(s string) {
	w.f.WriteString(s)
}

func (w *mdWriter) line(s string) {
	w.f.WriteString(s + "\n")
}

func (w *mdWriter) nl() {
	w.f.WriteString("\n")
}

func (w *mdWriter) h1(s string) {
	w.f.WriteString("# " + s + "\n\n")
}

func (w *mdWriter) h2(s string) {
	w.f.WriteString("\n## " + s + "\n\n")
}

func (w *mdWriter) h3(s string) {
	w.f.WriteString("### " + s + "\n\n")
}

func (w *mdWriter) code(lang, content string) {
	w.f.WriteString("```" + lang + "\n")
	w.f.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		w.f.WriteString("\n")
	}
	w.f.WriteString("```\n\n")
}

// --- Utilities ---

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

func humanNum(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

// EstimateTokens returns a rough token count for a string
func EstimateTokens(s string) int {
	charCount := utf8.RuneCountInString(s)
	return int(float64(charCount) * tokensPerChar)
}
