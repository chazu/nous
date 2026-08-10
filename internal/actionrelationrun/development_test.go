package actionrelationrun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chazu/nous/internal/actionrelationexp"
)

func TestPanelWriterAndAuditComparisonRequireExactFiles(t *testing.T) {
	primaryRoot, auditRoot := t.TempDir(), t.TempDir()
	logicalRoot := ".nous/actionrelations-v1-development-evidence"
	primary, audit := newPanelWriter(primaryRoot, logicalRoot, 1024), newPanelWriter(auditRoot, logicalRoot, 1024)
	files := []actionrelationexp.EvidenceFile{
		{Path: logicalRoot + "/packs/a.bin", Mode: "100644", Data: []byte("a")},
		{Path: logicalRoot + "/manifests/a.json", Mode: "100644", Data: []byte("manifest")},
	}
	if err := primary.writeAll(files); err != nil {
		t.Fatal(err)
	}
	if err := audit.writeAll(files); err != nil {
		t.Fatal(err)
	}
	if err := comparePanelFiles(primary, audit); err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(auditRoot, filepath.FromSlash(files[0].Path))
	if err := os.WriteFile(changed, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := comparePanelFiles(primary, audit); err == nil {
		t.Fatal("accepted changed audit bytes")
	}
	if err := primary.write(files[0]); err == nil {
		t.Fatal("accepted duplicate evidence path")
	}
}

func TestPanelWriterRejectsCapacityAndEscapingPath(t *testing.T) {
	root := ".nous/actionrelations-v1-development-evidence"
	writer := newPanelWriter(t.TempDir(), root, 1)
	if err := writer.write(actionrelationexp.EvidenceFile{Path: root + "/two", Mode: "100644", Data: []byte("xx")}); err == nil {
		t.Fatal("accepted evidence beyond capacity")
	}
	writer = newPanelWriter(t.TempDir(), root, 10)
	if err := writer.write(actionrelationexp.EvidenceFile{Path: root + "/../escape", Mode: "100644", Data: []byte("x")}); err == nil {
		t.Fatal("accepted escaping evidence path")
	}
}
