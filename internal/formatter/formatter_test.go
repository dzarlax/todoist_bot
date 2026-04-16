package formatter

import (
	"reflect"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestParseTaskMeta(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		wantPrio    int
		wantLabels  []string
		wantCleaned string
	}{
		{"empty", "", 1, nil, ""},
		{"no meta", "buy milk", 1, nil, "buy milk"},
		{"priority only", "buy milk !p2", 2, nil, "buy milk"},
		{"priority p4", "!p4 urgent call", 4, nil, "urgent call"},
		{"single label", "read book #home", 1, []string{"home"}, "read book"},
		{"multi labels", "#home clean up #chores now", 1, []string{"home", "chores"}, "clean up now"},
		{"priority + labels", "!p3 review PR #work #urgent", 3, []string{"work", "urgent"}, "review PR"},
		{"priority boundary not matched inside word", "hello!p2world", 1, nil, "hello!p2world"},
		{"hash inside URL is not a label", "see https://example.com/path#section", 1, nil, "see https://example.com/path#section"},
		{"case insensitive priority marker", "buy milk !P1", 1, nil, "buy milk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, l, c := ParseTaskMeta(tt.in)
			if p != tt.wantPrio {
				t.Errorf("priority: got %d, want %d", p, tt.wantPrio)
			}
			if !reflect.DeepEqual(l, tt.wantLabels) {
				t.Errorf("labels: got %v, want %v", l, tt.wantLabels)
			}
			if c != tt.wantCleaned {
				t.Errorf("clean: got %q, want %q", c, tt.wantCleaned)
			}
		})
	}
}

func TestFindProjectNameForUser(t *testing.T) {
	mapping := map[string][]string{
		"Family": {"@masik904"},
		"Work":   {"@alice", "Bob Jones"},
	}
	cases := []struct {
		user string
		want string
	}{
		{"@masik904", "Family"},
		{"Bob Jones", "Work"},
		{"@alice", "Work"},
		{"@unknown", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := FindProjectNameForUser(c.user, mapping); got != c.want {
			t.Errorf("%q → %q, want %q", c.user, got, c.want)
		}
	}
}

func TestFormatTextWithLinks_BareURL(t *testing.T) {
	in := "see https://example.com for info"
	want := "see [https://example.com](https://example.com) for info"
	if got := FormatTextWithLinks(in, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTextWithLinks_WWWPrefix(t *testing.T) {
	in := "visit www.example.com/path today"
	want := "visit [www.example.com/path](https://www.example.com/path) today"
	if got := FormatTextWithLinks(in, nil); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTextWithLinks_PreservesExistingMarkdown(t *testing.T) {
	in := "click [here](https://a.b/c) please"
	if got := FormatTextWithLinks(in, nil); got != in {
		t.Errorf("got %q, want %q (unchanged)", got, in)
	}
}

func TestFormatTextWithLinks_TextLinkEntity(t *testing.T) {
	text := "hello world"
	entities := []tgbotapi.MessageEntity{{
		Type:   "text_link",
		Offset: 6,
		Length: 5,
		URL:    "https://world.example/",
	}}
	want := "hello [world](https://world.example/)"
	if got := FormatTextWithLinks(text, entities); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
