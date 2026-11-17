package logline

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func FuzzParser(f *testing.F) {
	for _, s := range []string{
		"[12:00:00 INFO  SMAPI] hello",
		"[12:00:00 TRACE SMAPI] a\ncontinuation",
		"[12:00:00 DEBUG  Mod] x\r\ny",
		"no header",
		"[12:00:00 ALERT SMAPI] loaded save 'TheFarm'",
		"[12:00:00 WARN  SMAPI] Mods go here: ~/Mods",
		"",
		"[19:43:46 TRACE SMAPI] loading",
		"[19:43:46 INFO  Love of Cooking] Loaded 42 mods",
		"[19:43:50 ERROR SMAPI] Failed:\n   at Foo.Bar()",
		"[19:44:03 INFO ] no mod",
		"[19:44:05 LOUD SMAPI] not a level",
		"SMAPI 4.5.2 with Stardew Valley 1.6.15",
		"[19.43.46 INFO  SMAPI] Mods go here: ~/Mods",
		"[19:43:46 INFO  screen_1 SMAPI] Context: loaded save 'Farm_1'.",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, log string) {
		var p Parser
		for line := range strings.SplitSeq(log, "\n") {
			e := p.Parse(strings.TrimRight(line, "\r"))
			switch e.Level {
			case Trace, Debug, Info, Warn, Error, Alert:
			default:
				t.Fatalf("unknown level %q", e.Level)
			}
			if e.Cont && e.Time == "" {
				t.Fatal("continuation with no prior header")
			}
		}
	})
}

func FuzzLoadedSave(f *testing.F) {
	f.Add("[12:00:00 INFO  SMAPI] loaded save 'TheFarm'\n")
	f.Add("[12:00:00 INFO  SMAPI] Context: loaded save 'TheFarm'\n")
	f.Add("[12:00:00 INFO  Other] loaded save 'Nope'\n[12:00:00 INFO  SMAPI] loaded save 'Yes'\n")
	f.Add("loaded save 'Bare'\n")
	f.Add("")
	f.Add("[05:00:01 INFO  SMAPI] Context: loaded save 'O'Brien_123'.\n")
	f.Add("[05:00:01 INFO  SMAPI] Loaded Save 'Valley'\n")
	f.Add("[05:00:01 INFO  SMAPI] loaded save 'Keep'\n   at Foo()\n")
	f.Add("[05:00:01 INFO  SMAPI] reloaded save 'Nope'\n")
	f.Add("[05:00:01 INFO  SMAPI] Failed:\nloaded save 'Cont'\n")
	f.Fuzz(func(t *testing.T, log string) {
		folder, ok := LoadedSave(log)
		if ok != (folder != "") {
			t.Fatalf("ok=%v folder=%q", ok, folder)
		}
	})
}

func FuzzModsPath(f *testing.F) {
	f.Add("SMAPI 4.5.2 with Stardew Valley 1.6.15 on Unix\n[05:00:01 INFO  SMAPI] Mods go here: ~/p/mods\r\n")
	f.Add("[05:00:01 INFO  SMAPI] Mods go here: /srv/p/mods\n")
	f.Add("[05:00:01 INFO  SMAPI] Mods go here: ~\\AppData\\Roaming\\p\\mods\r\n")
	f.Add("[05:00:01 INFO  Other] Mods go here: /x\n")
	f.Add("[05:00:01 INFO  SMAPI] hello\n")
	f.Add("[19.43.46 INFO  SMAPI] Mods go here: ~/Mods\n")
	f.Add("")
	f.Fuzz(func(t *testing.T, log string) {
		path, ok := ModsPath(log)
		if ok != (path != "") {
			t.Fatalf("ok=%v path=%q", ok, path)
		}
	})
}

func FuzzLogOwnedBy(f *testing.F) {
	f.Add("[05:00:01 INFO  SMAPI] Mods go here: ~/p/a/mods/\n", "/home/u", "/home/u/p/a/mods")
	f.Add("[05:00:01 INFO  SMAPI] Mods go here: /srv/a\n", "/home/u", "/srv/a")
	f.Add("[05:00:01 INFO  SMAPI] hello\n", "/home/u", "/home/u/p/a/mods")
	f.Add("[05:00:01 INFO  SMAPI] Mods go here: ~/../etc\n", "/home/u", "/etc")
	f.Add("[05:00:01 INFO  SMAPI] Mods go here: ~/p/mods\n", "/home/u", "/home/u/p/mods/../secret")
	f.Fuzz(func(t *testing.T, log, home, modsDir string) {
		owned := LogOwnedBy(log, home, modsDir)
		path, ok := ModsPath(log)
		if owned && !ok {
			t.Fatal("owned a log that names no mods folder")
		}
		if !owned {
			return
		}
		expanded := path
		if rest, cut := strings.CutPrefix(path, "~"); cut && (rest == "" || rest[0] == '/' || rest[0] == '\\') {
			expanded = home + rest
		}
		if !safefs.SamePath(expanded, modsDir) {
			t.Fatalf("owned %q vs %q (log path %q)", expanded, modsDir, path)
		}
		if home != "" && filepath.IsAbs(home) && filepath.IsAbs(modsDir) {
			rel, err := filepath.Rel(filepath.Clean(home), filepath.Clean(modsDir))
			if err == nil && strings.HasPrefix(rel, "..") && !strings.Contains(path, "..") {
				if rest, cut := strings.CutPrefix(path, "~"); cut && (rest == "" || rest[0] == '/' || rest[0] == '\\') && !strings.Contains(rest, "..") {
					t.Fatalf("claimed modsDir %q escapes home %q from %q", modsDir, home, path)
				}
			}
		}
	})
}

func FuzzParsePerformanceReport(f *testing.F) {
	f.Add("| 1 | Mod | 10 | 1.00 | 0.50 | 2.00 | 5 |")
	f.Add("Mod | 10 | 1.00")
	f.Add("| Totals | 10 | 1.00 | 0.50 | 2.00 |")
	f.Add("")
	f.Add("|")
	f.Add("Mod | Avg Execution Time (last 60s) | Last Execution Time | Peak Execution Time\nDrachenkatze.AdvancedKeyBindings | 0.00 | 0.00 | 0.64")
	f.Add("Event | Average (ms) | Peak (ms) | Calls\nGameLoop.UpdateTicked | 0.44 | 2.10 | 98")
	f.Fuzz(func(t *testing.T, blob string) {
		lines := strings.Split(blob, "\n")
		rows := ParsePerformanceReport(lines)
		if len(rows) > len(lines) {
			t.Fatalf("%d rows from %d lines", len(rows), len(lines))
		}
		for _, r := range rows {
			if r.Name == "" {
				t.Fatal("empty performance row name")
			}
			if !finite(r.AverageMs) || !finite(r.PeakMs) || !finite(r.Calls) {
				t.Fatalf("non-finite numbers in %+v", r)
			}
		}
	})
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
