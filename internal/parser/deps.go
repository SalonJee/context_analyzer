package parser

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// Dependency represents a detected project dependency
type Dependency struct {
	Name    string
	Version string
	Source  string // which manifest file
}

// ParseDependencies extracts dependencies from known manifest files
func ParseDependencies(relPath, content string) []Dependency {
	name := filepath.Base(relPath)
	switch name {
	case "go.mod":
		return parseGoMod(content)
	case "package.json":
		return parsePackageJSON(content)
	case "requirements.txt":
		return parseRequirements(content)
	case "Cargo.toml":
		return parseCargoToml(content)
	case "pyproject.toml":
		return parsePyproject(content)
	case "composer.json":
		return parseComposerJSON(content)
	case "pom.xml":
		return parsePomXML(content)
	case "build.gradle":
		return parseGradle(content)
	case "Gemfile":
		return parseGemfile(content)
	}
	return nil
}

func parseGoMod(content string) []Dependency {
	var deps []Dependency
	lines := strings.Split(content, "\n")
	inRequire := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}
		if inRequire || strings.HasPrefix(trimmed, "require ") {
			parts := strings.Fields(strings.TrimPrefix(trimmed, "require "))
			if len(parts) >= 2 && !strings.HasPrefix(parts[0], "//") {
				version := parts[1]
				if strings.HasSuffix(version, "// indirect") {
					continue // skip indirect
				}
				deps = append(deps, Dependency{Name: parts[0], Version: version, Source: "go.mod"})
			}
		}
	}
	return deps
}

func parsePackageJSON(content string) []Dependency {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil
	}
	var deps []Dependency
	for name, ver := range pkg.Dependencies {
		deps = append(deps, Dependency{Name: name, Version: ver, Source: "package.json"})
	}
	// include a few notable devDeps
	notable := map[string]bool{
		"typescript": true, "webpack": true, "vite": true, "esbuild": true,
		"jest": true, "vitest": true, "eslint": true, "prettier": true,
	}
	for name, ver := range pkg.DevDependencies {
		if notable[name] {
			deps = append(deps, Dependency{Name: name + " (dev)", Version: ver, Source: "package.json"})
		}
	}
	return deps
}

func parseRequirements(content string) []Dependency {
	var deps []Dependency
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		// name==version or name>=version or just name
		for _, sep := range []string{"==", ">=", "<=", "~=", "!="} {
			if idx := strings.Index(line, sep); idx != -1 {
				deps = append(deps, Dependency{
					Name: strings.TrimSpace(line[:idx]),
					Version: strings.TrimSpace(line[idx:]),
					Source: "requirements.txt",
				})
				goto next
			}
		}
		deps = append(deps, Dependency{Name: line, Source: "requirements.txt"})
	next:
	}
	return deps
}

func parseCargoToml(content string) []Dependency {
	var deps []Dependency
	inDeps := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[dependencies]" || trimmed == "[dev-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inDeps = false
			continue
		}
		if inDeps && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if idx := strings.Index(trimmed, "="); idx != -1 {
				name := strings.TrimSpace(trimmed[:idx])
				ver := strings.TrimSpace(trimmed[idx+1:])
				ver = strings.Trim(ver, `"{}`)
				deps = append(deps, Dependency{Name: name, Version: ver, Source: "Cargo.toml"})
			}
		}
	}
	return deps
}

func parsePyproject(content string) []Dependency {
	var deps []Dependency
	inDeps := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[tool.poetry.dependencies]" || trimmed == "dependencies = [" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") && inDeps {
			inDeps = false
			continue
		}
		if inDeps && trimmed != "" {
			line2 := strings.Trim(trimmed, `",[]`)
			if idx := strings.IndexAny(line2, ">=<~!"); idx != -1 {
				deps = append(deps, Dependency{Name: strings.TrimSpace(line2[:idx]), Version: strings.TrimSpace(line2[idx:]), Source: "pyproject.toml"})
			} else if line2 != "" && !strings.HasPrefix(line2, "#") {
				deps = append(deps, Dependency{Name: line2, Source: "pyproject.toml"})
			}
		}
	}
	return deps
}

func parseComposerJSON(content string) []Dependency {
	var pkg struct {
		Require map[string]string `json:"require"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil
	}
	var deps []Dependency
	for name, ver := range pkg.Require {
		if name == "php" {
			continue
		}
		deps = append(deps, Dependency{Name: name, Version: ver, Source: "composer.json"})
	}
	return deps
}

func parsePomXML(content string) []Dependency {
	var deps []Dependency
	lines := strings.Split(content, "\n")
	var curGroup, curArtifact, curVersion string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if val := extractXMLTag(trimmed, "groupId"); val != "" {
			curGroup = val
		} else if val := extractXMLTag(trimmed, "artifactId"); val != "" {
			curArtifact = val
		} else if val := extractXMLTag(trimmed, "version"); val != "" {
			curVersion = val
		} else if trimmed == "</dependency>" {
			if curArtifact != "" {
				deps = append(deps, Dependency{
					Name:    curGroup + ":" + curArtifact,
					Version: curVersion,
					Source:  "pom.xml",
				})
			}
			curGroup, curArtifact, curVersion = "", "", ""
		}
	}
	return deps
}

func extractXMLTag(line, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	if idx := strings.Index(line, open); idx != -1 {
		start := idx + len(open)
		if end := strings.Index(line[start:], close); end != -1 {
			return line[start : start+end]
		}
	}
	return ""
}

func parseGradle(content string) []Dependency {
	var deps []Dependency
	re := strings.NewReplacer("'", "\"")
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(re.Replace(line))
		for _, prefix := range []string{"implementation ", "api ", "compile ", "testImplementation "} {
			if strings.HasPrefix(trimmed, prefix) {
				val := strings.TrimPrefix(trimmed, prefix)
				val = strings.Trim(val, `"`)
				parts := strings.Split(val, ":")
				if len(parts) >= 2 {
					name := parts[0] + ":" + parts[1]
					ver := ""
					if len(parts) >= 3 {
						ver = parts[2]
					}
					deps = append(deps, Dependency{Name: name, Version: ver, Source: "build.gradle"})
				}
			}
		}
	}
	return deps
}

func parseGemfile(content string) []Dependency {
	re := strings.NewReplacer("'", "\"")
	var deps []Dependency
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(re.Replace(line))
		if strings.HasPrefix(trimmed, "gem ") {
			parts := strings.Split(trimmed[4:], ",")
			if len(parts) >= 1 {
				name := strings.Trim(parts[0], `" `)
				ver := ""
				if len(parts) >= 2 {
					ver = strings.Trim(parts[1], `" `)
				}
				deps = append(deps, Dependency{Name: name, Version: ver, Source: "Gemfile"})
			}
		}
	}
	return deps
}
