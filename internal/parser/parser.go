package parser

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"
)

// Symbol represents an extracted code symbol
type Symbol struct {
	Kind    string // func, method, class, struct, interface, const, var, type, route
	Name    string
	Sig     string // signature or brief context
	Line    int
}

// FileAnalysis holds parsed information from a single file
type FileAnalysis struct {
	RelPath  string
	Language string
	Symbols  []Symbol
	Imports  []string
	Routes   []Route
	Summary  string // first meaningful comment block
}

// Route represents a detected HTTP route
type Route struct {
	Method string
	Path   string
	Line   int
}

// Parse analyzes a file's content based on its language
func Parse(relPath, language, content string) *FileAnalysis {
	fa := &FileAnalysis{
		RelPath:  relPath,
		Language: language,
	}
	fa.Summary = extractSummaryComment(content)

	switch language {
	case "Go":
		parseGo(fa, content)
	case "Python":
		parsePython(fa, content)
	case "TypeScript", "TypeScript (TSX)", "JavaScript", "JavaScript (JSX)":
		parseTypeScript(fa, content)
	case "Java":
		parseJava(fa, content)
	case "C#":
		parseCSharp(fa, content)
	case "Rust":
		parseRust(fa, content)
	case "PHP":
		parsePHP(fa, content)
	case "Ruby":
		parseRuby(fa, content)
	default:
		parseGeneric(fa, content)
	}

	fa.Routes = append(fa.Routes, extractRoutes(content, relPath)...)
	return fa
}

// --- Go parser ---
var (
	reFuncGo      = regexp.MustCompile(`^func\s+(\([^)]+\)\s+)?(\w+)\s*(\([^)]*\)[^{]*)`)
	reTypeGo      = regexp.MustCompile(`^type\s+(\w+)\s+(struct|interface|func|map|\[)`)
	reImportGo    = regexp.MustCompile(`"([^"]+)"`)
	reConstGo     = regexp.MustCompile(`^\s*(const|var)\s+(\w+)`)
)

func parseGo(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	inImportBlock := false
	inConstBlock := false
	_ = inConstBlock

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "import (" {
			inImportBlock = true
			continue
		}
		if inImportBlock {
			if trimmed == ")" {
				inImportBlock = false
				continue
			}
			if m := reImportGo.FindStringSubmatch(trimmed); m != nil {
				fa.Imports = append(fa.Imports, m[1])
			}
			continue
		}
		if strings.HasPrefix(trimmed, "import \"") {
			if m := reImportGo.FindStringSubmatch(trimmed); m != nil {
				fa.Imports = append(fa.Imports, m[1])
			}
		}

		if m := reFuncGo.FindStringSubmatch(trimmed); m != nil {
			receiver := strings.TrimSpace(m[1])
			name := m[2]
			sig := strings.TrimSpace(m[3])
			kind := "func"
			if receiver != "" {
				kind = "method"
				receiver = strings.Trim(receiver, "() ")
				// extract type name
				parts := strings.Fields(receiver)
				if len(parts) > 1 {
					receiver = strings.Trim(parts[len(parts)-1], "*")
				}
				fa.Symbols = append(fa.Symbols, Symbol{Kind: kind, Name: receiver + "." + name, Sig: sig, Line: i + 1})
			} else {
				fa.Symbols = append(fa.Symbols, Symbol{Kind: kind, Name: name, Sig: sig, Line: i + 1})
			}
		} else if m := reTypeGo.FindStringSubmatch(trimmed); m != nil {
			kind := m[2]
			if kind == "struct" || kind == "interface" {
				fa.Symbols = append(fa.Symbols, Symbol{Kind: kind, Name: m[1], Line: i + 1})
			} else {
				fa.Symbols = append(fa.Symbols, Symbol{Kind: "type", Name: m[1], Line: i + 1})
			}
		}
	}
}

// --- Python parser ---
var (
	rePyFunc   = regexp.MustCompile(`^(\s*)def\s+(\w+)\s*\(([^)]*)\)`)
	rePyClass  = regexp.MustCompile(`^class\s+(\w+)`)
	rePyImport = regexp.MustCompile(`^(?:from\s+(\S+)\s+import|import\s+(\S+))`)
)

func parsePython(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if m := rePyFunc.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			kind := "func"
			if indent > 0 {
				kind = "method"
			}
			fa.Symbols = append(fa.Symbols, Symbol{Kind: kind, Name: m[2], Sig: "(" + m[3] + ")", Line: i + 1})
		} else if m := rePyClass.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "class", Name: m[1], Line: i + 1})
		} else if m := rePyImport.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			if m[1] != "" {
				fa.Imports = append(fa.Imports, m[1])
			} else {
				fa.Imports = append(fa.Imports, m[2])
			}
		}
	}
}

// --- TypeScript/JavaScript parser ---
var (
	reTSFunc      = regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*[(<]`)
	reTSClass     = regexp.MustCompile(`(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`)
	reTSInterface = regexp.MustCompile(`(?:export\s+)?interface\s+(\w+)`)
	reTSType      = regexp.MustCompile(`(?:export\s+)?type\s+(\w+)\s*=`)
	reTSConst     = regexp.MustCompile(`(?:export\s+)?const\s+(\w+)\s*[:=]`)
	reTSImport    = regexp.MustCompile(`^import\s+.*?from\s+['"]([^'"]+)['"]`)
	reTSArrow     = regexp.MustCompile(`(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s+)?\(`)
)

func parseTypeScript(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := reTSImport.FindStringSubmatch(trimmed); m != nil {
			fa.Imports = append(fa.Imports, m[1])
			continue
		}
		if m := reTSClass.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "class", Name: m[1], Line: i + 1})
		} else if m := reTSInterface.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "interface", Name: m[1], Line: i + 1})
		} else if m := reTSType.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "type", Name: m[1], Line: i + 1})
		} else if m := reTSFunc.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "func", Name: m[1], Line: i + 1})
		} else if m := reTSArrow.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "func", Name: m[1], Line: i + 1})
		} else if m := reTSConst.FindStringSubmatch(trimmed); m != nil {
			// Only capture exported consts as they may be important
			if strings.HasPrefix(trimmed, "export") {
				fa.Symbols = append(fa.Symbols, Symbol{Kind: "const", Name: m[1], Line: i + 1})
			}
		}
	}
}

// --- Java parser ---
var (
	reJavaClass  = regexp.MustCompile(`(?:public\s+)?(?:abstract\s+)?(?:final\s+)?class\s+(\w+)`)
	reJavaIface  = regexp.MustCompile(`(?:public\s+)?interface\s+(\w+)`)
	reJavaMethod = regexp.MustCompile(`(?:public|private|protected)\s+(?:static\s+)?(?:\w+\s+)+(\w+)\s*\(`)
	reJavaImport = regexp.MustCompile(`^import\s+([\w.]+);`)
)

func parseJava(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := reJavaImport.FindStringSubmatch(trimmed); m != nil {
			fa.Imports = append(fa.Imports, m[1])
		} else if m := reJavaClass.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "class", Name: m[1], Line: i + 1})
		} else if m := reJavaIface.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "interface", Name: m[1], Line: i + 1})
		} else if m := reJavaMethod.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "method", Name: m[1], Line: i + 1})
		}
	}
}

// --- C# parser ---
var (
	reCsClass  = regexp.MustCompile(`(?:public|private|internal|protected)?\s*(?:abstract|sealed|static)?\s*class\s+(\w+)`)
	reCsIface  = regexp.MustCompile(`(?:public|private|internal)?\s*interface\s+(\w+)`)
	reCsMethod = regexp.MustCompile(`(?:public|private|protected|internal)\s+(?:static\s+)?(?:async\s+)?(?:\w+\s+)+(\w+)\s*\(`)
	reCsUsing  = regexp.MustCompile(`^using\s+([\w.]+);`)
)

func parseCSharp(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := reCsUsing.FindStringSubmatch(trimmed); m != nil {
			fa.Imports = append(fa.Imports, m[1])
		} else if m := reCsClass.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "class", Name: m[1], Line: i + 1})
		} else if m := reCsIface.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "interface", Name: m[1], Line: i + 1})
		} else if m := reCsMethod.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "method", Name: m[1], Line: i + 1})
		}
	}
}

// --- Rust parser ---
var (
	reRustFn     = regexp.MustCompile(`(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?fn\s+(\w+)`)
	reRustStruct = regexp.MustCompile(`(?:pub(?:\([^)]*\))?\s+)?struct\s+(\w+)`)
	reRustEnum   = regexp.MustCompile(`(?:pub(?:\([^)]*\))?\s+)?enum\s+(\w+)`)
	reRustTrait  = regexp.MustCompile(`(?:pub(?:\([^)]*\))?\s+)?trait\s+(\w+)`)
	reRustUse    = regexp.MustCompile(`^use\s+([\w:{}*]+);`)
)

func parseRust(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := reRustUse.FindStringSubmatch(trimmed); m != nil {
			fa.Imports = append(fa.Imports, m[1])
		} else if m := reRustFn.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "func", Name: m[1], Line: i + 1})
		} else if m := reRustStruct.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "struct", Name: m[1], Line: i + 1})
		} else if m := reRustEnum.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "enum", Name: m[1], Line: i + 1})
		} else if m := reRustTrait.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "trait", Name: m[1], Line: i + 1})
		}
	}
}

// --- PHP parser ---
var (
	rePHPClass  = regexp.MustCompile(`(?:abstract\s+)?class\s+(\w+)`)
	rePHPFunc   = regexp.MustCompile(`(?:public|private|protected)?\s*(?:static\s+)?function\s+(\w+)`)
	rePHPUse    = regexp.MustCompile(`^use\s+([\w\\]+)`)
)

func parsePHP(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := rePHPUse.FindStringSubmatch(trimmed); m != nil {
			fa.Imports = append(fa.Imports, m[1])
		} else if m := rePHPClass.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "class", Name: m[1], Line: i + 1})
		} else if m := rePHPFunc.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "func", Name: m[1], Line: i + 1})
		}
	}
}

// --- Ruby parser ---
var (
	reRubyClass  = regexp.MustCompile(`^class\s+(\w+)`)
	reRubyModule = regexp.MustCompile(`^module\s+(\w+)`)
	reRubyDef    = regexp.MustCompile(`^\s*def\s+(\w+)`)
	reRubyReq    = regexp.MustCompile(`^require(?:_relative)?\s+['"]([^'"]+)['"]`)
)

func parseRuby(fa *FileAnalysis, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := reRubyReq.FindStringSubmatch(trimmed); m != nil {
			fa.Imports = append(fa.Imports, m[1])
		} else if m := reRubyClass.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "class", Name: m[1], Line: i + 1})
		} else if m := reRubyModule.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "module", Name: m[1], Line: i + 1})
		} else if m := reRubyDef.FindStringSubmatch(trimmed); m != nil {
			fa.Symbols = append(fa.Symbols, Symbol{Kind: "func", Name: m[1], Line: i + 1})
		}
	}
}

// --- Generic fallback ---
func parseGeneric(fa *FileAnalysis, content string) {
	// No-op for unsupported languages
}

// --- Route extraction ---
var routePatterns = []*regexp.Regexp{
	// Go: router.GET("/path", handler)
	regexp.MustCompile(`(?i)(?:router|r|mux|app|e|g)\s*\.\s*(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|Handle|HandleFunc|Any)\s*\(\s*["` + "`" + `]([^"` + "`" + `]+)["` + "`" + `]`),
	// Express/Fastify: app.get('/path', ...)
	regexp.MustCompile(`(?i)(?:app|router|server)\s*\.\s*(get|post|put|patch|delete|head|options|use|all)\s*\(\s*['"]([^'"]+)['"]`),
	// Flask: @app.route('/path', methods=['GET'])
	regexp.MustCompile(`@\w+\.route\s*\(\s*['"]([^'"]+)['"](?:[^)]*methods\s*=\s*\[([^\]]*)\])?`),
	// Django: path('route/', view)
	regexp.MustCompile(`(?:path|re_path|url)\s*\(\s*r?['"]([^'"]+)['"]`),
	// Spring: @RequestMapping/@GetMapping etc
	regexp.MustCompile(`@(?:RequestMapping|GetMapping|PostMapping|PutMapping|DeleteMapping|PatchMapping)\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`),
	// ASP.NET: [Route("path")]
	regexp.MustCompile(`\[(?:Route|HttpGet|HttpPost|HttpPut|HttpDelete|HttpPatch)\s*\(\s*["']([^"']+)["']`),
}

func extractRoutes(content, relPath string) []Route {
	var routes []Route
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, re := range routePatterns {
			m := re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			route := Route{Line: i + 1}
			if len(m) >= 3 {
				route.Method = strings.ToUpper(m[1])
				route.Path = m[2]
			} else if len(m) >= 2 {
				route.Path = m[1]
				route.Method = inferMethodFromContext(trimmed)
			}
			if route.Path != "" && !strings.Contains(route.Path, "{{") {
				routes = append(routes, route)
			}
		}
	}
	return routes
}

func inferMethodFromContext(line string) string {
	lower := strings.ToLower(line)
	for _, m := range []string{"get", "post", "put", "patch", "delete"} {
		if strings.Contains(lower, m) {
			return strings.ToUpper(m)
		}
	}
	return "GET"
}

// extractSummaryComment grabs the first comment block from a file
func extractSummaryComment(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	inBlock := false
	count := 0

	for scanner.Scan() && count < 30 {
		count++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") {
			text := strings.TrimPrefix(trimmed, "//")
			text = strings.TrimPrefix(text, " ")
			if text != "" && !strings.HasPrefix(strings.ToLower(text), "generated") &&
				!strings.HasPrefix(strings.ToLower(text), "copyright") {
				lines = append(lines, text)
				inBlock = true
			}
		} else if strings.HasPrefix(trimmed, "/*") {
			inBlock = true
		} else if strings.HasSuffix(trimmed, "*/") {
			inBlock = false
		} else if inBlock && trimmed != "" && !strings.HasPrefix(trimmed, "*") {
			break
		} else if inBlock && strings.HasPrefix(trimmed, "*") {
			text := strings.TrimPrefix(trimmed, "*")
			text = strings.TrimSpace(text)
			if text != "" {
				lines = append(lines, text)
			}
		} else if !inBlock && trimmed != "" && !strings.HasPrefix(trimmed, "#!") &&
			!strings.HasPrefix(trimmed, "package ") &&
			!strings.HasPrefix(trimmed, "import ") {
			break
		}
	}

	if len(lines) > 3 {
		lines = lines[:3]
	}
	return strings.Join(lines, " ")
}

// filepath import needed
var _ = filepath.Ext
