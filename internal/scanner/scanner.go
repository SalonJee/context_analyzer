package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// FileInfo holds metadata about a scanned file
type FileInfo struct {
	Path     string
	RelPath  string
	Size     int64
	Language string
	IsBinary bool
}

// Scanner walks a directory respecting ignore rules
type Scanner struct {
	Root        string
	ignoreRules []ignoreRule
}

type ignoreRule struct {
	pattern  string
	negated  bool
	dirOnly  bool
	anchored bool
}

var defaultIgnoreDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	".git":         true,
	"target":       true,
	"bin":          true,
	"vendor":       true,
	".cache":       true,
	"__pycache__":  true,
	".pytest_cache": true,
	"coverage":     true,
	".nyc_output":  true,
	"out":          true,
	".next":        true,
	".nuxt":        true,
	".svelte-kit":  true,
	"venv":         true,
	".venv":        true,
	"env":          true,
	".env":         true,
	".tox":         true,
	"htmlcov":      true,
	".gradle":      true,
	".mvn":         true,
	"pkg":          true,
	"obj":          true,
	"Debug":        true,
	"Release":      true,
}

var defaultIgnoreFiles = map[string]bool{
	".DS_Store":      true,
	"Thumbs.db":      true,
	"desktop.ini":    true,
	".gitkeep":       true,
}

var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".svg": false, // svg is text
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".7z": true, ".rar": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".a": true,
	".bin": true, ".o": true, ".obj": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
	".sqlite": true, ".db": true,
	".pyc": true, ".pyo": true, ".class": true,
	".lock": false, // lock files are text
}

// New creates a Scanner for the given root directory
func New(root string) (*Scanner, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	s := &Scanner{Root: abs}
	s.loadGitignore(abs)
	return s, nil
}

func (s *Scanner) loadGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule := ignoreRule{}
		if strings.HasPrefix(line, "!") {
			rule.negated = true
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			rule.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		if strings.HasPrefix(line, "/") {
			rule.anchored = true
			line = line[1:]
		}
		rule.pattern = line
		s.ignoreRules = append(s.ignoreRules, rule)
	}
}

func (s *Scanner) isIgnoredByGitignore(relPath string, isDir bool) bool {
	parts := strings.Split(relPath, string(os.PathSeparator))
	name := parts[len(parts)-1]

	for _, rule := range s.ignoreRules {
		if rule.dirOnly && !isDir {
			continue
		}
		matched := false
		if rule.anchored {
			matched, _ = filepath.Match(rule.pattern, relPath)
			if !matched {
				matched, _ = filepath.Match(rule.pattern, name)
			}
		} else {
			// Check against each path component
			for _, part := range parts {
				m, _ := filepath.Match(rule.pattern, part)
				if m {
					matched = true
					break
				}
			}
			if !matched {
				matched, _ = filepath.Match(rule.pattern, relPath)
			}
		}
		if matched {
			if rule.negated {
				return false
			}
			return true
		}
	}
	return false
}

// Walk traverses the directory and returns all relevant files
func (s *Scanner) Walk() ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}

		relPath, _ := filepath.Rel(s.Root, path)
		if relPath == "." {
			return nil
		}

		name := info.Name()

		// Skip hidden files/dirs (except important ones)
		if strings.HasPrefix(name, ".") {
			allowed := map[string]bool{
				".gitignore": true, ".env.example": true, ".editorconfig": true,
				".eslintrc": true, ".prettierrc": true, ".babelrc": true,
			}
			if !allowed[name] {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			if defaultIgnoreDirs[name] {
				return filepath.SkipDir
			}
			if s.isIgnoredByGitignore(relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}

		// File checks
		if defaultIgnoreFiles[name] {
			return nil
		}
		if s.isIgnoredByGitignore(relPath, false) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		isBinary := false
		if v, ok := binaryExtensions[ext]; ok {
			isBinary = v
		} else if ext == "" {
			isBinary = isBinaryFile(path)
		}

		files = append(files, FileInfo{
			Path:     path,
			RelPath:  relPath,
			Size:     info.Size(),
			Language: DetectLanguage(name, ext),
			IsBinary: isBinary,
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	return files, nil
}

func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
		r := rune(b)
		if b < 32 && !unicode.IsSpace(r) && b != 27 {
			return true
		}
	}
	return false
}

// ReadFile reads a text file's content
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
