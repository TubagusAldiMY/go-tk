package crud

import (
	"testing"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		check   func(t *testing.T, fields []FieldDef)
	}{
		{
			name:    "empty input",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "single string field",
			input:   "name:string",
			wantLen: 1,
			check: func(t *testing.T, fields []FieldDef) {
				f := fields[0]
				if f.Name != "Name" {
					t.Errorf("Name = %q, want %q", f.Name, "Name")
				}
				if f.GoType != "string" {
					t.Errorf("GoType = %q, want %q", f.GoType, "string")
				}
				if f.DBType != "VARCHAR(255)" {
					t.Errorf("DBType = %q, want %q", f.DBType, "VARCHAR(255)")
				}
				if !f.Required {
					t.Error("Required should be true for non-optional field")
				}
			},
		},
		{
			name:    "optional field with ?",
			input:   "description:string?",
			wantLen: 1,
			check: func(t *testing.T, fields []FieldDef) {
				f := fields[0]
				if f.GoType != "*string" {
					t.Errorf("GoType = %q, want %q", f.GoType, "*string")
				}
				if f.Required {
					t.Error("Required should be false for optional field")
				}
			},
		},
		{
			name:    "multiple fields",
			input:   "name:string,price:float64,stock:int,active:bool",
			wantLen: 4,
			check: func(t *testing.T, fields []FieldDef) {
				types := map[string]string{
					"Name":   "string",
					"Price":  "float64",
					"Stock":  "int",
					"Active": "bool",
				}
				for _, f := range fields {
					want, ok := types[f.Name]
					if !ok {
						t.Errorf("unexpected field: %s", f.Name)
						continue
					}
					if f.GoType != want {
						t.Errorf("field %s: GoType = %q, want %q", f.Name, f.GoType, want)
					}
				}
			},
		},
		{
			name:    "snake_case field name becomes PascalCase",
			input:   "product_name:string",
			wantLen: 1,
			check: func(t *testing.T, fields []FieldDef) {
				f := fields[0]
				if f.Name != "ProductName" {
					t.Errorf("Name = %q, want ProductName", f.Name)
				}
				if f.ColumnName != "product_name" {
					t.Errorf("ColumnName = %q, want product_name", f.ColumnName)
				}
				if f.JSONTag != "product_name" {
					t.Errorf("JSONTag = %q, want product_name", f.JSONTag)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := ParseFields(tt.input)
			if err != nil {
				t.Fatalf("ParseFields(%q) error: %v", tt.input, err)
			}
			if len(fields) != tt.wantLen {
				t.Fatalf("len(fields) = %d, want %d", len(fields), tt.wantLen)
			}
			if tt.check != nil {
				tt.check(t, fields)
			}
		})
	}
}

func TestParseFieldsInvalidFormat(t *testing.T) {
	_, err := ParseFields("nameonly")
	if err == nil {
		t.Error("expected error for field without type")
	}
}

func TestParseFieldsDBTypeMySQL(t *testing.T) {
	fields, err := ParseFields("name:string,count:int,price:float64,active:bool,data:[]byte")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"name":   "VARCHAR(255)",
		"count":  "INT",
		"price":  "DECIMAL(10,4)",
		"active": "TINYINT(1)",
		"data":   "BLOB",
	}
	for _, f := range fields {
		if exp, ok := want[f.NameSnake]; ok {
			if f.DBTypeMySQL != exp {
				t.Errorf("DBTypeMySQL for %q = %q, want %q", f.NameSnake, f.DBTypeMySQL, exp)
			}
		}
	}
}

func TestHandlerTemplate(t *testing.T) {
	if got := handlerTemplate("fiber"); got != "handler_fiber.go.tmpl" {
		t.Errorf("handlerTemplate(fiber) = %q", got)
	}
	if got := handlerTemplate("gin"); got != "handler_gin.go.tmpl" {
		t.Errorf("handlerTemplate(gin) = %q", got)
	}
	if got := handlerTemplate(""); got != "handler_gin.go.tmpl" {
		t.Errorf("handlerTemplate('') = %q, want gin default", got)
	}
}

func TestMigrationUpTemplate(t *testing.T) {
	if got := migrationUpTemplate("mysql"); got != "migration.up.mysql.sql.tmpl" {
		t.Errorf("migrationUpTemplate(mysql) = %q", got)
	}
	if got := migrationUpTemplate("postgres"); got != "migration.up.postgres.sql.tmpl" {
		t.Errorf("migrationUpTemplate(postgres) = %q", got)
	}
	if got := migrationUpTemplate(""); got != "migration.up.postgres.sql.tmpl" {
		t.Errorf("migrationUpTemplate('') = %q, want postgres default", got)
	}
}

func TestFieldCaseHelpers(t *testing.T) {
	cases := []struct {
		input  string
		pascal string
		camel  string
		snake  string
	}{
		{"product", "Product", "product", "product"},
		{"product_name", "ProductName", "productName", "product_name"},
		{"ProductName", "ProductName", "productName", "product_name"},
	}
	for _, c := range cases {
		if got := toPascalCase(c.input); got != c.pascal {
			t.Errorf("toPascalCase(%q) = %q, want %q", c.input, got, c.pascal)
		}
		if got := toCamelCase(c.input); got != c.camel {
			t.Errorf("toCamelCase(%q) = %q, want %q", c.input, got, c.camel)
		}
		if got := toSnakeCase(c.input); got != c.snake {
			t.Errorf("toSnakeCase(%q) = %q, want %q", c.input, got, c.snake)
		}
	}
}
