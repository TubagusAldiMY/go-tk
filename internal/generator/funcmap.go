package generator

import (
	"strings"
	"text/template"
	"unicode"
)

// defaultFuncMap returns template helper functions available in all templates.
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// String transformations
		"toLower":    strings.ToLower,
		"toUpper":    strings.ToUpper,
		"title":      toPascalCase,
		"toCamel":    toCamelCase,
		"toPascal":   toPascalCase,
		"toSnake":    toSnakeCase,
		"toPlural":   toPlural,
		"trimSpace":  strings.TrimSpace,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"trimSuffix": strings.TrimSuffix,
		"trimPrefix": strings.TrimPrefix,
		"replace":    strings.ReplaceAll,
		"contains":   strings.Contains,
		"join":       strings.Join,
		// Conditionals
		"ternary": ternary,
	}
}

// toCamelCase converts PascalCase or snake_case to camelCase.
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
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

// toSnakeCase converts PascalCase or camelCase to snake_case.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteRune('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// toPlural appends 's' or 'es' based on simple English rules.
func toPlural(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"),
		strings.HasSuffix(s, "z"), strings.HasSuffix(s, "ch"),
		strings.HasSuffix(s, "sh"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

func isVowel(r rune) bool {
	return strings.ContainsRune("aeiou", r)
}

// splitWords splits a string on underscores, hyphens, and capital letter boundaries.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else if unicode.IsUpper(r) && i > 0 && current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
			current.WriteRune(r)
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func ternary(cond bool, a, b any) any {
	if cond {
		return a
	}
	return b
}
