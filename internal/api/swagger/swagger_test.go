package swagger

import (
	"testing"
)

func TestSwaggerFiles(t *testing.T) {
	files, err := swaggerFiles.ReadDir("docs")
	if err != nil {
		t.Fatalf("Failed to read docs directory: %v", err)
	}

	found := make(map[string]bool)
	for _, file := range files {
		found[file.Name()] = true
	}

	requiredFiles := []string{"swagger.yaml", "swagger-ui.html"}
	for _, required := range requiredFiles {
		if !found[required] {
			t.Errorf("Required file %s not found in embedded files", required)
		}

		data, err := swaggerFiles.ReadFile("docs/" + required)
		if err != nil {
			t.Errorf("Failed to read %s: %v", required, err)
		}
		if len(data) == 0 {
			t.Errorf("File %s is empty", required)
		}
	}
}
