package new

import "testing"

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "my-api", false},
		{"simple name", "api", false},
		{"with numbers", "api2", false},
		{"empty name", "", true},
		{"contains space", "my api", true},
		{"contains slash", "my/api", true},
		{"contains backslash", "my\\api", true},
		{"contains colon", "my:api", true},
		{"contains asterisk", "my*api", true},
		{"contains question mark", "my?api", true},
		{"contains quotes", "my\"api", true},
		{"contains angle bracket", "my<api", true},
		{"contains pipe", "my|api", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectName(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestResolveModulePath(t *testing.T) {
	tests := []struct {
		flagModule  string
		projectName string
		want        string
	}{
		{"github.com/user/custom", "api", "github.com/user/custom"},
		{"", "my-api", "github.com/username/my-api"},
		{"", "hello", "github.com/username/hello"},
	}
	for _, tt := range tests {
		got := resolveModulePath(tt.flagModule, tt.projectName)
		if got != tt.want {
			t.Errorf("resolveModulePath(%q, %q) = %q, want %q", tt.flagModule, tt.projectName, got, tt.want)
		}
	}
}

func TestTemplateDirForStack(t *testing.T) {
	tests := []struct {
		framework, database, want string
	}{
		{"gin", "postgres", "gin-postgres"},
		{"gin", "mysql", "gin-mysql"},
		{"fiber", "postgres", "fiber-postgres"},
		{"fiber", "mysql", "fiber-mysql"},
	}
	for _, tt := range tests {
		got := templateDirForStack(tt.framework, tt.database)
		if got != tt.want {
			t.Errorf("templateDirForStack(%q, %q) = %q, want %q", tt.framework, tt.database, got, tt.want)
		}
	}
}

func TestResolveOutputPath(t *testing.T) {
	tests := []struct {
		targetDir, tmplPath, templateDir, want string
	}{
		{"/tmp/myapp", "gin-postgres/cmd/api/main.go.tmpl", "gin-postgres", "/tmp/myapp/cmd/api/main.go"},
		{"/tmp/myapp", "gin-postgres/.gitignore.tmpl", "gin-postgres", "/tmp/myapp/.gitignore"},
		{"/tmp/myapp", "fiber-mysql/Dockerfile.tmpl", "fiber-mysql", "/tmp/myapp/Dockerfile"},
	}
	for _, tt := range tests {
		got := resolveOutputPath(tt.targetDir, tt.tmplPath, tt.templateDir)
		if got != tt.want {
			t.Errorf("resolveOutputPath(%q, %q, %q) = %q, want %q",
				tt.targetDir, tt.tmplPath, tt.templateDir, got, tt.want)
		}
	}
}
