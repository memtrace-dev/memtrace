package util

import "testing"

func TestStripPrivate_NoTags(t *testing.T) {
	s := "normal content with no private tags"
	if got := StripPrivate(s); got != s {
		t.Errorf("want unchanged, got %q", got)
	}
}

func TestStripPrivate_SingleBlock(t *testing.T) {
	s := "public part <private>secret key=abc123</private> more public"
	got := StripPrivate(s)
	if got != "public part  more public" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestStripPrivate_BlockAtStart(t *testing.T) {
	s := "<private>secret</private> public content"
	got := StripPrivate(s)
	if got != "public content" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestStripPrivate_BlockAtEnd(t *testing.T) {
	s := "public content <private>secret</private>"
	got := StripPrivate(s)
	if got != "public content" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestStripPrivate_MultipleBlocks(t *testing.T) {
	s := "start <private>a</private> middle <private>b</private> end"
	got := StripPrivate(s)
	if got != "start  middle  end" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestStripPrivate_MultilineBlock(t *testing.T) {
	s := "before\n<private>\nline1\nline2\n</private>\nafter"
	got := StripPrivate(s)
	// The newline before <private> and after </private> both survive stripping,
	// producing one blank line between the surrounding paragraphs.
	if got != "before\n\nafter" {
		t.Errorf("unexpected result: %q", got)
	}
}

func TestStripPrivate_CaseInsensitive(t *testing.T) {
	cases := []string{
		"text <PRIVATE>secret</PRIVATE> text",
		"text <Private>secret</Private> text",
		"text <PRIVATE>secret</private> text",
	}
	for _, s := range cases {
		got := StripPrivate(s)
		if got != "text  text" {
			t.Errorf("case-insensitive strip failed for %q: got %q", s, got)
		}
	}
}

func TestStripPrivate_OnlyPrivateContent(t *testing.T) {
	s := "<private>everything is secret</private>"
	got := StripPrivate(s)
	if got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

func TestStripPrivate_NormalisesExtraBlankLines(t *testing.T) {
	s := "line1\n\n\n\n<private>gone</private>\n\n\n\nline2"
	got := StripPrivate(s)
	if got != "line1\n\nline2" {
		t.Errorf("unexpected whitespace: %q", got)
	}
}

func TestStripPrivate_EmptyString(t *testing.T) {
	if got := StripPrivate(""); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}
