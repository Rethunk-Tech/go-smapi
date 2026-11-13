package logline

import (
	"reflect"
	"testing"
)

func TestParserLevelsModsAndContinuations(t *testing.T) {
	var p Parser
	cases := []struct {
		line string
		want Entry
	}{
		{"[19:43:46 TRACE SMAPI] loading", Entry{Time: "19:43:46", Level: Trace, Mod: "SMAPI", Message: "loading"}},
		{"[19:43:46 DEBUG SMAPI] d", Entry{Time: "19:43:46", Level: Debug, Mod: "SMAPI", Message: "d"}},
		{"[19:43:46 INFO  Love of Cooking] Loaded 42 mods", Entry{Time: "19:43:46", Level: Info, Mod: "Love of Cooking", Message: "Loaded 42 mods"}},
		{"[19:43:49 WARN  SMAPI] obsolete", Entry{Time: "19:43:49", Level: Warn, Mod: "SMAPI", Message: "obsolete"}},
		{"[19:43:50 ERROR SMAPI] Failed:", Entry{Time: "19:43:50", Level: Error, Mod: "SMAPI", Message: "Failed:"}},
		{"   at Foo.Bar()", Entry{Time: "19:43:50", Level: Error, Mod: "SMAPI", Message: "   at Foo.Bar()", Cont: true}},
		{"", Entry{Time: "19:43:50", Level: Error, Mod: "SMAPI", Message: "", Cont: true}},
		{"[19:44:02 ALERT SMAPI] alert!", Entry{Time: "19:44:02", Level: Alert, Mod: "SMAPI", Message: "alert!"}},
		{"[19:44:03 INFO ] no mod", Entry{Time: "19:44:03", Level: Info, Message: "no mod"}},
		{"[19:44:04 INFO SMAPI]", Entry{Time: "19:44:04", Level: Info, Mod: "SMAPI"}},
		{"[19:44:05 LOUD SMAPI] not a level", Entry{Time: "19:44:04", Level: Info, Mod: "SMAPI", Message: "[19:44:05 LOUD SMAPI] not a level", Cont: true}},
	}
	for _, c := range cases {
		if got := p.Parse(c.line); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Parse(%q) = %+v, want %+v", c.line, got, c.want)
		}
	}
}

func TestParserHeaderlessFirstLine(t *testing.T) {
	var p Parser
	got := p.Parse("SMAPI 4.5.2 with Stardew Valley 1.6.15")
	if got.Level != Info || got.Cont || got.Mod != "" || got.Message != "SMAPI 4.5.2 with Stardew Valley 1.6.15" {
		t.Fatalf("got %+v", got)
	}
}

func TestModsPathReadsSMAPIsIntroLine(t *testing.T) {
	cases := map[string]struct {
		log  string
		want string
		ok   bool
	}{
		"home":     {"SMAPI 4.5.2 with Stardew Valley 1.6.15 on Unix\n[05:00:01 INFO  SMAPI] Mods go here: ~/p/mods\r\n", "~/p/mods", true},
		"absolute": {"[05:00:01 INFO  SMAPI] Mods go here: /srv/p/mods\n", "/srv/p/mods", true},
		"windows":  {"[05:00:01 INFO  SMAPI] Mods go here: ~\\AppData\\Roaming\\p\\mods\r\n", "~\\AppData\\Roaming\\p\\mods", true},
		"a mod":    {"[05:00:01 INFO  Other] Mods go here: /x\n", "", false},
		"missing":  {"[05:00:01 INFO  SMAPI] hello\n", "", false},
	}
	for name, c := range cases {
		if got, ok := ModsPath(c.log); got != c.want || ok != c.ok {
			t.Errorf("%s: got %q, %v", name, got, ok)
		}
	}
}

func TestLogOwnedByExpandsHome(t *testing.T) {
	log := "[05:00:01 INFO  SMAPI] Mods go here: ~/p/a/mods/\n"
	if !LogOwnedBy(log, "/home/u", "/home/u/p/a/mods") {
		t.Error("profile a wrote the log")
	}
	if LogOwnedBy(log, "/home/u", "/home/u/p/b/mods") || LogOwnedBy(log, "/home/v", "/home/u/p/a/mods") {
		t.Error("only profile a under /home/u wrote the log")
	}
	if !LogOwnedBy("[05:00:01 INFO  SMAPI] Mods go here: /srv/a\n", "/home/u", "/srv/a") {
		t.Error("an absolute path compares as written")
	}
	if LogOwnedBy("[05:00:01 INFO  SMAPI] hello\n", "/home/u", "/home/u/p/a/mods") {
		t.Error("a log without the line belongs to no profile")
	}
}

func TestLoadedSaveLastSMAPIMatch(t *testing.T) {
	cases := map[string]struct {
		log  string
		want string
		ok   bool
	}{
		"plain":    {"[05:00:01 INFO  SMAPI] loaded save 'Spring'\n", "Spring", true},
		"context":  {"[05:00:01 INFO  SMAPI] Context: loaded save 'Farm_123'\n", "Farm_123", true},
		"case":     {"[05:00:01 INFO  SMAPI] Loaded Save 'Valley'\n", "Valley", true},
		"last":     {"[05:00:01 INFO  SMAPI] loaded save 'One'\n[05:00:02 INFO  SMAPI] loaded save 'Two'\n", "Two", true},
		"other":    {"[05:00:01 INFO  Other] loaded save 'Nope'\n", "", false},
		"missing":  {"[05:00:01 INFO  SMAPI] hello\n", "", false},
		"continue": {"[05:00:01 INFO  SMAPI] loaded save 'Keep'\n   at Foo()\n", "Keep", true},
	}
	for name, c := range cases {
		if got, ok := LoadedSave(c.log); got != c.want || ok != c.ok {
			t.Errorf("%s: got %q, %v", name, got, ok)
		}
	}
}

// SMAPI formats the time with HH:mm:ss, where ':' is the culture's time separator (a period in fi-FI, for one).
func TestParserReadsACulturesTimeSeparator(t *testing.T) {
	var p Parser
	want := Entry{Time: "19.43.46", Level: Info, Mod: "SMAPI", Message: "Mods go here: ~/Mods"}
	if got := p.Parse("[19.43.46 INFO  SMAPI] Mods go here: ~/Mods"); got != want {
		t.Fatalf("got %+v", got)
	}
	if path, ok := ModsPath("[19.43.46 INFO  SMAPI] Mods go here: ~/Mods\n"); !ok || path != "~/Mods" {
		t.Fatalf("ModsPath = %q, %v", path, ok)
	}
}

// In split-screen SMAPI writes the screen between the level and the mod.
func TestParserSplitScreenHeaderNamesTheMod(t *testing.T) {
	var p Parser
	want := Entry{Time: "19:43:46", Level: Info, Mod: "SMAPI", Message: "Context: loaded save 'Farm_1'."}
	if got := p.Parse("[19:43:46 INFO  screen_1 SMAPI] Context: loaded save 'Farm_1'."); got != want {
		t.Fatalf("got %+v", got)
	}
}
