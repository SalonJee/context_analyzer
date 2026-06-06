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

const outputDir = "ctxgen-output"

func resolveOutputPath(targetDir, cmd string) (string, error) {
	folder := filepath.Join(targetDir, outputDir)
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", fmt.Errorf("cannot create output folder %s: %w", folder, err)
	}

	ext := ".md"
	if cmd == "pack" {
		ext = ".txt"
	}

	next := 1
	entries, _ := os.ReadDir(folder)
	for _, e := range entries {
		var n int
		fmt.Sscanf(e.Name(), cmd+"-%d"+ext, &n)
		if n >= next {
			next = n + 1
		}
	}

	filename := fmt.Sprintf("%s-%d%s", cmd, next, ext)
	return filepath.Join(folder, filename), nil
}

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
	resolvedOutput := *outputPath
if resolvedOutput == "" {
    resolvedOutput, err = resolveOutputPath(abs, cmd)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}


	opts := writer.Options{
		OutputPath:  resolvedOutput,
		Timestamp:   *timestamp,
		MaxFileSize: *maxSize,
	}

	var outPath string

	switch cmd {
	case "pack":
		
		fmt.Fprintln(os.Stderr, "Writing pack file...")
		outPath, err = writer.WritePack(files, contents, opts)

	case "context":
		
		fmt.Fprintln(os.Stderr, "Analyzing repository...")
		ra := analyzer.Analyze(abs, files, contents)
		fmt.Fprintf(os.Stderr, "Detected %d languages, %d dependencies, %d routes\n",
			len(ra.Languages), len(ra.Dependencies), len(ra.AllRoutes))
		outPath, err = writer.WriteContext(ra, contents, opts)

	case "hybrid":
		
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
