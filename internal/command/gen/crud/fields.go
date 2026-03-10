package crud

import (
	"fmt"
	"strings"
)

// FieldDef holds all derived information about a single entity field.
type FieldDef struct {
	Name        string // PascalCase: "ProductName"
	NameLC      string // camelCase:  "productName"
	NameSnake   string // snake_case: "product_name"
	GoType      string // "string", "float64", "int", "bool", "*string"
	DBType      string // PostgreSQL: "VARCHAR(255)", "INTEGER", "BYTEA"
	DBTypeMySQL string // MySQL:      "VARCHAR(255)", "INT", "BLOB"
	GormTag     string // "column:product_name;type:varchar(255);not null"
	JSONTag     string // "product_name"
	ValidTag    string // "required,max=255"
	ColumnName  string // snake_case, same as NameSnake
	Required    bool
}

// goTypeToDBType maps Go primitive types to PostgreSQL column types.
var goTypeToDBType = map[string]string{
	"string":  "VARCHAR(255)",
	"int":     "INTEGER",
	"int64":   "BIGINT",
	"uint":    "INTEGER",
	"uint64":  "BIGINT",
	"float32": "DECIMAL(10,4)",
	"float64": "DECIMAL(10,4)",
	"bool":    "BOOLEAN",
	"[]byte":  "BYTEA",
}

// goTypeToDBTypeMySQL maps Go primitive types to MySQL column types.
var goTypeToDBTypeMySQL = map[string]string{
	"string":  "VARCHAR(255)",
	"int":     "INT",
	"int64":   "BIGINT",
	"uint":    "INT UNSIGNED",
	"uint64":  "BIGINT UNSIGNED",
	"float32": "DECIMAL(10,4)",
	"float64": "DECIMAL(10,4)",
	"bool":    "TINYINT(1)",
	"[]byte":  "BLOB",
}

// ParseFields parses a comma-separated field definition string.
// Format: "name:type,price:float64,active:bool"
// An optional "?" suffix marks the field as optional (pointer type).
func ParseFields(raw string) ([]FieldDef, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	fields := make([]FieldDef, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid field definition %q — expected name:type", part)
		}

		rawName := strings.TrimSpace(kv[0])
		rawType := strings.TrimSpace(kv[1])

		optional := strings.HasSuffix(rawType, "?")
		if optional {
			rawType = rawType[:len(rawType)-1]
		}

		goType := rawType
		if optional {
			goType = "*" + rawType
		}

		dbType, ok := goTypeToDBType[rawType]
		if !ok {
			dbType = "TEXT" // safe fallback
		}

		dbTypeMySQL, ok := goTypeToDBTypeMySQL[rawType]
		if !ok {
			dbTypeMySQL = "TEXT" // safe fallback
		}

		snake := toSnakeCase(rawName)
		pascal := toPascalCase(rawName)

		gormTag := fmt.Sprintf("column:%s;type:%s", snake, strings.ToLower(dbType))
		if !optional {
			gormTag += ";not null"
		}

		validTag := ""
		if !optional {
			validTag = "required"
			if rawType == "string" {
				validTag += ",max=255"
			}
		}

		fields = append(fields, FieldDef{
			Name:        pascal,
			NameLC:      toCamelCase(rawName),
			NameSnake:   snake,
			GoType:      goType,
			DBType:      dbType,
			DBTypeMySQL: dbTypeMySQL,
			GormTag:     gormTag,
			JSONTag:     snake,
			ValidTag:    validTag,
			ColumnName:  snake,
			Required:    !optional,
		})
	}

	return fields, nil
}

// toPascalCase converts snake_case or camelCase to PascalCase.
func toPascalCase(s string) string {
	parts := splitWords(s)
	var b strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			b.WriteString(strings.ToUpper(p[:1]) + strings.ToLower(p[1:]))
		}
	}
	return b.String()
}

// toCamelCase converts to camelCase.
func toCamelCase(s string) string {
	p := toPascalCase(s)
	if len(p) == 0 {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// toSnakeCase converts PascalCase/camelCase to snake_case.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteRune('_')
			}
			b.WriteRune(r + 32) // toLower: safe only for A-Z
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func splitWords(s string) []string {
	var words []string
	var cur strings.Builder
	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		} else if r >= 'A' && r <= 'Z' && i > 0 && cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
			cur.WriteRune(r)
		} else {
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}
