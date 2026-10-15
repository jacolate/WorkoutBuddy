package media

import (
	"strings"
	"testing"
)

func TestPathRejectsTraversal(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "../x.mp4", "a/b.mp4", ".upload-1", ".."} {
		if _, err := s.Path(name); err == nil {
			t.Errorf("%q accepted", name)
		}
	}
	name, n, err := s.Save(strings.NewReader("abc"), "video/mp4")
	if err != nil || n != 3 || !strings.HasSuffix(name, ".mp4") {
		t.Fatalf("save: %q %d %v", name, n, err)
	}
	if err := s.Delete(name); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(name); err != nil {
		t.Fatalf("second delete should be a no-op: %v", err)
	}
}

func TestExtForMime(t *testing.T) {
	cases := map[string]string{
		"video/webm;codecs=vp9": ".webm",
		"video/mp4":             ".mp4",
		"video/quicktime":       ".mov",
		"image/png":             ".bin",
	}
	for mime, want := range cases {
		if got := ExtForMime(mime); got != want {
			t.Errorf("%s: got %s want %s", mime, got, want)
		}
	}
}
