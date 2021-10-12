package acceptance

import (
	"os"
	"path/filepath"
	"testing"
)

func Cleanup(t *testing.T) {
	files, err := filepath.Glob("./terraform.tfstate.*")
	if err != nil {
		t.Fatalf("File search failed due to: %v", err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			t.Fatalf("Removal failed due to: %v", err)
		}
	}

	t.Log("Cleanup succeeded")
}
