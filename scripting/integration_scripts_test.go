package scripting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// TestIntegrationScriptsCompile reads every .ss file from the
// fixtures/integration_tests directory and verifies that each one
// compiles without errors through the full SentanylScript pipeline.
func TestIntegrationScriptsCompile(t *testing.T) {
	dir := filepath.Join("fixtures", "integration_tests")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read integration_tests dir: %v", err)
	}

	var ssFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ss") {
			ssFiles = append(ssFiles, e.Name())
		}
	}

	if len(ssFiles) == 0 {
		t.Fatal("no .ss files found in fixtures/integration_tests")
	}

	t.Logf("found %d integration test scripts", len(ssFiles))

	for _, name := range ssFiles {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name)
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", name, err)
			}

			ResetIDCounter()
			result := CompileScript(string(src), "sub_integ", bson.NewObjectId())

			// Check for compilation errors
			for _, d := range result.Diagnostics {
				if d.Level == DiagError {
					t.Errorf("compile error at %s: %s", d.Pos, d.Message)
				}
			}

			// Every script must produce at least 1 story
			if len(result.Stories) == 0 {
				t.Errorf("expected at least 1 story from %s", name)
			}

			// Log structure for debugging
			for _, story := range result.Stories {
				t.Logf("  story %q: %d storylines", story.Name, len(story.Storylines))
				for _, sl := range story.Storylines {
					t.Logf("    storyline %q: %d enactments", sl.Name, len(sl.Acts))
				}
			}
		})
	}
}
