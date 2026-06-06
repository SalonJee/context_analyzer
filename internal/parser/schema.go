package parser

import (
	"regexp"
	"strings"
)

// SchemaInfo holds detected database schema information
type SchemaInfo struct {
	Tables    []TableInfo
	Source    string
}

// TableInfo holds table name and column hints
type TableInfo struct {
	Name    string
	Columns []string
}

var (
	reSQLCreate   = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?["'\x60]?(\w+)["'\x60]?`)
	reSQLColumn   = regexp.MustCompile(`^\s+["'\x60]?(\w+)["'\x60]?\s+(INTEGER|INT|BIGINT|SMALLINT|SERIAL|TEXT|VARCHAR|CHAR|BOOLEAN|BOOL|FLOAT|DOUBLE|DECIMAL|NUMERIC|DATE|TIME|TIMESTAMP|DATETIME|JSON|JSONB|UUID|BLOB|BYTEA)`)
	reGoStruct    = regexp.MustCompile(`type\s+(\w+)\s+struct`)
	reGoField     = regexp.MustCompile(`^\s+(\w+)\s+\w+.*\x60.*(?:db|json|gorm):"([^"]+)"`)
	reMigration   = regexp.MustCompile(`(?i)(?:create_table|add_column|add_index|change_table)\s+:?(\w+)`)
	reORMModel    = regexp.MustCompile(`(?i)(?:Model|Table)\s*\(\s*["']?(\w+)["']?`)
)

// ExtractSchema detects schema information from SQL, migrations, and ORM models
func ExtractSchema(relPath, language, content string) *SchemaInfo {
	schema := &SchemaInfo{Source: relPath}
	ext := strings.ToLower(relPath[strings.LastIndex(relPath, ".")+1:])

	if ext == "sql" || language == "SQL" {
		schema.Tables = append(schema.Tables, parseSQLSchema(content)...)
	}

	// Rails/ActiveRecord migrations
	if language == "Ruby" && strings.Contains(relPath, "migrat") {
		schema.Tables = append(schema.Tables, parseRailsMigration(content)...)
	}

	// Go struct with db tags (GORM, sqlx)
	if language == "Go" {
		schema.Tables = append(schema.Tables, parseGoStructSchema(content)...)
	}

	// TypeScript/Prisma schema
	if ext == "prisma" || strings.Contains(relPath, "schema.prisma") {
		schema.Tables = append(schema.Tables, parsePrismaSchema(content)...)
	}

	// Django models.py
	if language == "Python" && (strings.Contains(relPath, "model") || strings.Contains(relPath, "models")) {
		schema.Tables = append(schema.Tables, parseDjangoModels(content)...)
	}

	return schema
}

func parseSQLSchema(content string) []TableInfo {
	var tables []TableInfo
	var current *TableInfo
	inTable := false

	for _, line := range strings.Split(content, "\n") {
		if m := reSQLCreate.FindStringSubmatch(line); m != nil {
			if current != nil {
				tables = append(tables, *current)
			}
			current = &TableInfo{Name: m[1]}
			inTable = true
			continue
		}
		if inTable && current != nil {
			if strings.TrimSpace(line) == ");" || strings.TrimSpace(line) == ")," {
				tables = append(tables, *current)
				current = nil
				inTable = false
				continue
			}
			if m := reSQLColumn.FindStringSubmatch(line); m != nil {
				current.Columns = append(current.Columns, m[1]+" "+strings.ToLower(m[2]))
			}
		}
	}
	if current != nil {
		tables = append(tables, *current)
	}
	return tables
}

func parseRailsMigration(content string) []TableInfo {
	var tables []TableInfo
	var current *TableInfo

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "create_table") {
			m := regexp.MustCompile(`create_table\s+:?["']?(\w+)["']?`).FindStringSubmatch(trimmed)
			if m != nil {
				if current != nil {
					tables = append(tables, *current)
				}
				current = &TableInfo{Name: m[1]}
			}
		} else if current != nil {
			// t.string :field_name
			if m := regexp.MustCompile(`t\.(\w+)\s+:(\w+)`).FindStringSubmatch(trimmed); m != nil {
				current.Columns = append(current.Columns, m[2]+" "+m[1])
			} else if strings.HasPrefix(trimmed, "end") {
				tables = append(tables, *current)
				current = nil
			}
		}
	}
	return tables
}

func parseGoStructSchema(content string) []TableInfo {
	var tables []TableInfo
	var current *TableInfo
	inStruct := false

	for _, line := range strings.Split(content, "\n") {
		if m := reGoStruct.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			// Only capture structs that look like DB models
			name := m[1]
			inStruct = true
			current = &TableInfo{Name: name}
			continue
		}
		if inStruct && current != nil {
			trimmed := strings.TrimSpace(line)
			if trimmed == "}" {
				if len(current.Columns) > 0 {
					tables = append(tables, *current)
				}
				current = nil
				inStruct = false
				continue
			}
			// Check for db or gorm tags
			if m := reGoField.FindStringSubmatch(line); m != nil {
				colName := strings.Split(m[2], ",")[0]
				if colName != "" && colName != "-" {
					current.Columns = append(current.Columns, colName)
				}
			}
		}
	}
	return tables
}

func parsePrismaSchema(content string) []TableInfo {
	var tables []TableInfo
	var current *TableInfo

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "model ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				current = &TableInfo{Name: parts[1]}
			}
		} else if current != nil {
			if trimmed == "}" {
				tables = append(tables, *current)
				current = nil
				continue
			}
			// field_name FieldType
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 && !strings.HasPrefix(parts[0], "@@") && !strings.HasPrefix(parts[0], "//") {
				current.Columns = append(current.Columns, parts[0]+" "+strings.ToLower(parts[1]))
			}
		}
	}
	return tables
}

func parseDjangoModels(content string) []TableInfo {
	var tables []TableInfo
	var current *TableInfo

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// class MyModel(models.Model):
		if m := regexp.MustCompile(`^class\s+(\w+)\s*\(.*[Mm]odel`).FindStringSubmatch(trimmed); m != nil {
			if current != nil && len(current.Columns) > 0 {
				tables = append(tables, *current)
			}
			current = &TableInfo{Name: m[1]}
		} else if current != nil {
			// field = models.CharField(...)
			if m := regexp.MustCompile(`^\s+(\w+)\s*=\s*models\.(\w+)`).FindStringSubmatch(line); m != nil {
				fieldName := m[1]
				if fieldName != "Meta" && fieldName != "class" {
					current.Columns = append(current.Columns, fieldName+" "+strings.ToLower(m[2]))
				}
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "def ") {
				if strings.HasPrefix(trimmed, "class ") {
					// New class
					if len(current.Columns) > 0 {
						tables = append(tables, *current)
					}
					current = nil
				}
			}
		}
	}
	if current != nil && len(current.Columns) > 0 {
		tables = append(tables, *current)
	}
	return tables
}
