package scanner

import "strings"

// DetectLanguage maps file extensions/names to language names
func DetectLanguage(name, ext string) string {
	nameMap := map[string]string{
		"Makefile": "Makefile", "makefile": "Makefile",
		"Dockerfile": "Dockerfile", "dockerfile": "Dockerfile",
		"Jenkinsfile": "Groovy",
		"Vagrantfile": "Ruby",
		"Gemfile": "Ruby",
		"Rakefile": "Ruby",
		"go.mod": "Go Module", "go.sum": "Go Module",
		"package.json": "JSON", "package-lock.json": "JSON",
		"tsconfig.json": "JSON",
		"requirements.txt": "Text", "requirements.in": "Text",
		"pyproject.toml": "TOML",
		"Cargo.toml": "TOML", "Cargo.lock": "TOML",
		"composer.json": "JSON",
		"pom.xml": "XML",
		"build.gradle": "Groovy", "build.gradle.kts": "Kotlin",
		".gitignore": "GitIgnore",
		".editorconfig": "INI",
		".env.example": "ENV",
	}
	if lang, ok := nameMap[name]; ok {
		return lang
	}

	extMap := map[string]string{
		".go": "Go",
		".rs": "Rust",
		".c": "C", ".h": "C",
		".cpp": "C++", ".cc": "C++", ".cxx": "C++", ".hpp": "C++",
		".cs": "C#",
		".java": "Java",
		".kt": "Kotlin", ".kts": "Kotlin",
		".swift": "Swift",
		".py": "Python",
		".rb": "Ruby",
		".php": "PHP",
		".pl": "Perl",
		".lua": "Lua",
		".ex": "Elixir", ".exs": "Elixir",
		".erl": "Erlang",
		".hs": "Haskell",
		".clj": "Clojure",
		".scala": "Scala",
		".groovy": "Groovy",
		".js": "JavaScript", ".mjs": "JavaScript", ".cjs": "JavaScript",
		".jsx": "JavaScript (JSX)",
		".ts": "TypeScript", ".mts": "TypeScript",
		".tsx": "TypeScript (TSX)",
		".html": "HTML", ".htm": "HTML",
		".css": "CSS",
		".scss": "SCSS", ".sass": "SASS",
		".less": "LESS",
		".vue": "Vue",
		".svelte": "Svelte",
		".sh": "Shell", ".bash": "Shell", ".zsh": "Shell",
		".ps1": "PowerShell",
		".bat": "Batch", ".cmd": "Batch",
		".json": "JSON",
		".yaml": "YAML", ".yml": "YAML",
		".toml": "TOML",
		".xml": "XML",
		".csv": "CSV",
		".sql": "SQL",
		".graphql": "GraphQL", ".gql": "GraphQL",
		".proto": "Protobuf",
		".md": "Markdown", ".mdx": "Markdown",
		".rst": "reStructuredText",
		".txt": "Text",
		".ini": "INI", ".cfg": "INI", ".conf": "Config",
		".env": "ENV",
		".tf": "Terraform", ".tfvars": "Terraform",
		".hcl": "HCL",
		".tmpl": "Template", ".tpl": "Template",
		".j2": "Jinja2",
		".lock": "Lockfile",
	}

	if lang, ok := extMap[ext]; ok {
		return lang
	}
	if ext != "" {
		return strings.ToUpper(strings.TrimPrefix(ext, "."))
	}
	return "Unknown"
}
