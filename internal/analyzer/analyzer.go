package analyzer

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/ctxgen/ctxgen/internal/parser"
	"github.com/ctxgen/ctxgen/internal/scanner"
)

// RepoAnalysis is the complete analysis of a repository
type RepoAnalysis struct {
	Root         string
	Files        []scanner.FileInfo
	Languages    map[string]int   // language -> file count
	TotalSize    int64
	TotalChars   int64
	Dependencies []parser.Dependency
	FileTree     string
	Modules      []Module
	EntryPoints  []string
	AllRoutes    []RouteEntry
	AllSchemas   []parser.SchemaInfo
	ImportGraph  map[string][]string
	ConfigFiles  []string
	IgnoredCount int
}

// Module represents a logical grouping of related files
type Module struct {
	Name    string
	Path    string
	Files   []ModuleFile
	IsEntry bool
}

// ModuleFile is a file within a module with its analysis
type ModuleFile struct {
	RelPath  string
	Language string
	Symbols  []parser.Symbol
	Imports  []string
	Summary  string
}

// RouteEntry is a route with its source file
type RouteEntry struct {
	parser.Route
	File string
}

// Analyze builds the full repository analysis
func Analyze(root string, files []scanner.FileInfo, contents map[string]string) *RepoAnalysis {
	ra := &RepoAnalysis{
		Root:      root,
		Files:     files,
		Languages: make(map[string]int),
		ImportGraph: make(map[string][]string),
	}

	// Config/manifest file names
	manifestNames := map[string]bool{
		"go.mod": true, "go.sum": true,
		"package.json": true, "package-lock.json": true, "yarn.lock": true,
		"requirements.txt": true, "Pipfile": true, "pyproject.toml": true,
		"Cargo.toml": true,
		"pom.xml": true, "build.gradle": true,
		"composer.json": true,
		"Makefile": true, "makefile": true,
		"Dockerfile": true,
		".gitignore": true,
		"docker-compose.yml": true, "docker-compose.yaml": true,
		".env.example": true,
	}

	moduleMap := make(map[string]*Module)

	for _, f := range files {
		if f.IsBinary {
			ra.IgnoredCount++
			continue
		}

		ra.Languages[f.Language]++
		ra.TotalSize += f.Size

		content := contents[f.RelPath]
		ra.TotalChars += int64(len(content))

		// Config file detection
		base := filepath.Base(f.RelPath)
		if manifestNames[base] {
			ra.ConfigFiles = append(ra.ConfigFiles, f.RelPath)
		}

		// Entry point detection
		if isEntryPoint(f.RelPath, f.Language) {
			ra.EntryPoints = append(ra.EntryPoints, f.RelPath)
		}

		// Parse dependencies from manifests
		if manifestNames[base] && content != "" {
			deps := parser.ParseDependencies(f.RelPath, content)
			ra.Dependencies = append(ra.Dependencies, deps...)
		}

		// Parse file symbols
		if content != "" && !f.IsBinary {
			fa := parser.Parse(f.RelPath, f.Language, content)

			// Schema extraction
			schema := parser.ExtractSchema(f.RelPath, f.Language, content)
			if len(schema.Tables) > 0 {
				ra.AllSchemas = append(ra.AllSchemas, *schema)
			}

			// Route collection
			for _, route := range fa.Routes {
				ra.AllRoutes = append(ra.AllRoutes, RouteEntry{Route: route, File: f.RelPath})
			}

			// Import graph
			if len(fa.Imports) > 0 {
				ra.ImportGraph[f.RelPath] = fa.Imports
			}

			// Build module map based on directory
			dir := filepath.Dir(f.RelPath)
			if dir == "." {
				dir = "/"
			}
			if _, ok := moduleMap[dir]; !ok {
				moduleMap[dir] = &Module{
					Path: dir,
					Name: filepath.Base(dir),
				}
			}
			mod := moduleMap[dir]
			mf := ModuleFile{
				RelPath:  f.RelPath,
				Language: f.Language,
				Symbols:  fa.Symbols,
				Imports:  fa.Imports,
				Summary:  fa.Summary,
			}
			if isEntryPoint(f.RelPath, f.Language) {
				mod.IsEntry = true
			}
			mod.Files = append(mod.Files, mf)
		}
	}

	// Sort modules by path depth then name
	for _, mod := range moduleMap {
		sort.Slice(mod.Files, func(i, j int) bool {
			return mod.Files[i].RelPath < mod.Files[j].RelPath
		})
		ra.Modules = append(ra.Modules, *mod)
	}
	sort.Slice(ra.Modules, func(i, j int) bool {
		return ra.Modules[i].Path < ra.Modules[j].Path
	})

	ra.FileTree = buildFileTree(files, root)
	return ra
}

func isEntryPoint(relPath, language string) bool {
	base := filepath.Base(relPath)
	dir := filepath.Dir(relPath)

	entryFiles := map[string]bool{
		"main.go": true, "main.py": true, "main.rs": true, "main.ts": true,
		"main.js": true, "main.tsx": true,
		"index.ts": true, "index.js": true, "index.tsx": true,
		"app.ts": true, "app.js": true, "app.tsx": true, "app.py": true,
		"server.ts": true, "server.js": true,
		"manage.py": true, "wsgi.py": true, "asgi.py": true,
		"Program.cs": true,
	}
	if !entryFiles[base] {
		return false
	}
	// Prefer root-level or src-level entries
	depth := strings.Count(relPath, string(filepath.Separator))
	return depth <= 2 || strings.HasPrefix(dir, "src") || strings.HasPrefix(dir, "cmd")
}

// buildFileTree generates a tree representation of the files
func buildFileTree(files []scanner.FileInfo, root string) string {
	type node struct {
		name     string
		children map[string]*node
		files    []string
		isDir    bool
	}

	rootNode := &node{name: ".", children: make(map[string]*node), isDir: true}

	for _, f := range files {
		if f.IsBinary {
			continue
		}
		parts := strings.Split(f.RelPath, string(filepath.Separator))
		cur := rootNode
		for i, part := range parts {
			if i == len(parts)-1 {
				cur.files = append(cur.files, part)
			} else {
				if _, ok := cur.children[part]; !ok {
					cur.children[part] = &node{name: part, children: make(map[string]*node), isDir: true}
				}
				cur = cur.children[part]
			}
		}
	}

	var sb strings.Builder
	var walk func(n *node, prefix string, last bool)
	walk = func(n *node, prefix string, last bool) {
		// Sort children and files
		dirs := make([]string, 0, len(n.children))
		for k := range n.children {
			dirs = append(dirs, k)
		}
		sort.Strings(dirs)
		sort.Strings(n.files)

		all := make([]string, 0, len(dirs)+len(n.files))
		all = append(all, dirs...)
		all = append(all, n.files...)

		for i, name := range all {
			isLast := i == len(all)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}
			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}

			if child, ok := n.children[name]; ok {
				sb.WriteString(prefix + connector + name + "/\n")
				walk(child, childPrefix, isLast)
			} else {
				sb.WriteString(prefix + connector + name + "\n")
			}
		}
	}

	sb.WriteString(".\n")
	walk(rootNode, "", true)
	return sb.String()
}

// TopLanguages returns languages sorted by file count
func (ra *RepoAnalysis) TopLanguages() []LangCount {
	var langs []LangCount
	for lang, count := range ra.Languages {
		langs = append(langs, LangCount{Language: lang, Count: count})
	}
	sort.Slice(langs, func(i, j int) bool {
		if langs[i].Count != langs[j].Count {
			return langs[i].Count > langs[j].Count
		}
		return langs[i].Language < langs[j].Language
	})
	return langs
}

// LangCount is a language with file count
type LangCount struct {
	Language string
	Count    int
}

// DetectFrameworks infers frameworks from dependencies and file patterns
func (ra *RepoAnalysis) DetectFrameworks() []string {
	fwMap := map[string]string{
		// Go
		"github.com/gin-gonic/gin": "Gin (Go)",
		"github.com/labstack/echo": "Echo (Go)",
		"github.com/gofiber/fiber": "Fiber (Go)",
		"github.com/gorilla/mux":   "Gorilla Mux (Go)",
		"github.com/go-chi/chi":    "Chi (Go)",
		// JS/TS
		"react":        "React",
		"next":         "Next.js",
		"vue":          "Vue.js",
		"nuxt":         "Nuxt.js",
		"angular":      "@angular/core -> Angular",
		"svelte":       "Svelte",
		"express":      "Express.js",
		"fastify":      "Fastify",
		"nestjs":       "NestJS",
		"@nestjs/core": "NestJS",
		"koa":          "Koa.js",
		// Python
		"django":      "Django",
		"flask":       "Flask",
		"fastapi":     "FastAPI",
		"tornado":     "Tornado",
		"starlette":   "Starlette",
		// Java/Kotlin
		"springframework": "Spring Framework",
		"spring-boot":     "Spring Boot",
		// Rust
		"actix-web": "Actix Web",
		"axum":      "Axum",
		"warp":      "Warp",
		"rocket":    "Rocket",
		// PHP
		"laravel/framework": "Laravel",
		"symfony/symfony":   "Symfony",
		// Ruby
		"rails":  "Ruby on Rails",
		"sinatra": "Sinatra",
	}

	seen := make(map[string]bool)
	var frameworks []string

	for _, dep := range ra.Dependencies {
		depLower := strings.ToLower(dep.Name)
		for key, fw := range fwMap {
			if strings.Contains(depLower, key) && !seen[fw] {
				seen[fw] = true
				frameworks = append(frameworks, fw)
			}
		}
	}

	sort.Strings(frameworks)
	return frameworks
}

// ImportantFiles returns files that are likely most important
func (ra *RepoAnalysis) ImportantFiles() []scanner.FileInfo {
	var result []scanner.FileInfo
	priority := map[string]int{
		"main.go": 10, "main.py": 10, "main.rs": 10, "main.ts": 10, "main.js": 10,
		"app.go": 9, "app.py": 9, "app.ts": 9, "app.js": 9,
		"server.go": 9, "server.ts": 9, "server.js": 9,
		"router.go": 8, "routes.go": 8, "router.ts": 8, "routes.ts": 8,
		"config.go": 7, "config.py": 7, "config.ts": 7,
		"database.go": 7, "db.go": 7, "models.py": 7,
		"Dockerfile": 6, "docker-compose.yml": 6, "Makefile": 5,
		"go.mod": 5, "package.json": 5, "pyproject.toml": 5,
		"README.md": 4, "readme.md": 4,
	}

	for _, f := range ra.Files {
		if f.IsBinary {
			continue
		}
		base := filepath.Base(f.RelPath)
		if p, ok := priority[base]; ok && p >= 7 {
			result = append(result, f)
		}
	}

	// Limit to top 10
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}
