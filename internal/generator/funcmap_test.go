package generator

import (
	"testing"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ProductName", "productName"},
		{"product_name", "productName"},
		{"product", "product"},
		{"ID", "iD"},
		{"UserID", "userID"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"product_name", "ProductName"},
		{"productName", "ProductName"},
		{"product", "Product"},
		{"user_id", "UserId"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ProductName", "product_name"},
		{"product", "product"},
		{"UserID", "user_i_d"},
		{"HTTPStatus", "h_t_t_p_status"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPlural(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"product", "products"},
		{"category", "categories"},
		{"box", "boxes"},
		{"class", "classes"},
		{"user", "users"},
		{"company", "companies"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPlural(tt.input)
			if got != tt.want {
				t.Errorf("toPlural(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTernary(t *testing.T) {
	if ternary(true, "yes", "no") != "yes" {
		t.Error("ternary(true) should return first arg")
	}
	if ternary(false, "yes", "no") != "no" {
		t.Error("ternary(false) should return second arg")
	}
}

func TestDefaultFuncMap(t *testing.T) {
	fm := defaultFuncMap()
	requiredKeys := []string{
		"toLower", "toUpper", "title", "toCamel", "toPascal",
		"toSnake", "toPlural", "trimSpace", "ternary",
	}
	for _, k := range requiredKeys {
		if _, ok := fm[k]; !ok {
			t.Errorf("funcmap missing key: %s", k)
		}
	}
}
