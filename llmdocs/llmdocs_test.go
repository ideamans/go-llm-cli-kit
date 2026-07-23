package llmdocs

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"00-guide.md":    {Data: []byte("# Guide\n\nHow to drive the CLI.\n")},
		"90-commands.md": {Data: []byte("# Command catalog\n\n## `demo ping`\n")},
		"10-auth.md":     {Data: []byte("# Authentication\n\nSet DEMO_TOKEN.\n")},
		"notes.txt":      {Data: []byte("ignored")},
	}
}

func TestSectionsOrderedByFilename(t *testing.T) {
	sections, err := New(testFS(), ".").Sections()
	if err != nil {
		t.Fatalf("Sections: %v", err)
	}
	want := []string{"00-guide.md", "10-auth.md", "90-commands.md"}
	if len(sections) != len(want) {
		t.Fatalf("got %d sections, want %d", len(sections), len(want))
	}
	for i, name := range want {
		if sections[i].File != name {
			t.Errorf("section %d = %q, want %q", i, sections[i].File, name)
		}
	}
	if sections[1].Title != "Authentication" {
		t.Errorf("title = %q, want %q", sections[1].Title, "Authentication")
	}
}

func TestMarkdownConcatenatesInOrder(t *testing.T) {
	out, err := New(testFS(), ".").Markdown()
	if err != nil {
		t.Fatalf("Markdown: %v", err)
	}
	guide := strings.Index(out, "# Guide")
	auth := strings.Index(out, "# Authentication")
	catalog := strings.Index(out, "# Command catalog")
	if guide < 0 || auth < 0 || catalog < 0 {
		t.Fatalf("missing chapters in output:\n%s", out)
	}
	if !(guide < auth && auth < catalog) {
		t.Errorf("chapters out of order: guide=%d auth=%d catalog=%d", guide, auth, catalog)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error("output should end with a newline")
	}
	if strings.Contains(out, "ignored") {
		t.Error("non-Markdown files must be skipped")
	}
}

func TestMarkdownSeparatesChaptersWithBlankLine(t *testing.T) {
	out, err := New(fstest.MapFS{
		"00-a.md": {Data: []byte("# A\n\nbody\n")},
		"01-b.md": {Data: []byte("# B\n\nbody\n")},
	}, ".").Markdown()
	if err != nil {
		t.Fatalf("Markdown: %v", err)
	}
	if !strings.Contains(out, "body\n\n# B") {
		t.Errorf("chapters not separated by a blank line:\n%q", out)
	}
}

func TestJSONRoundTrips(t *testing.T) {
	data, err := New(testFS(), ".").JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var sections []Section
	if err := json.Unmarshal(data, &sections); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(sections))
	}
	if sections[0].Title != "Guide" || sections[0].File != "00-guide.md" {
		t.Errorf("unexpected first section: %+v", sections[0])
	}
}

func TestSubdirectoryRoot(t *testing.T) {
	fsys := fstest.MapFS{"docs/00-guide.md": {Data: []byte("# Guide\n")}}
	out, err := New(fsys, "docs").Markdown()
	if err != nil {
		t.Fatalf("Markdown: %v", err)
	}
	if !strings.Contains(out, "# Guide") {
		t.Errorf("got %q", out)
	}
}

func TestEmptyDirectory(t *testing.T) {
	fsys := fstest.MapFS{"readme.txt": {Data: []byte("x")}}
	d := New(fsys, ".")
	out, err := d.Markdown()
	if err != nil {
		t.Fatalf("Markdown: %v", err)
	}
	if out != "" {
		t.Errorf("got %q, want empty", out)
	}
	data, err := d.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("got %s, want []", data)
	}
}

func TestMissingDirectoryIsAnError(t *testing.T) {
	if _, err := New(fstest.MapFS{}, "nope").Sections(); err == nil {
		t.Error("expected an error for a missing directory")
	}
}
