package logline

import (
	"strings"
	"testing"
)

func FuzzParser(f *testing.F) {
	f.Add("[12:00:00 INFO  SMAPI] hello")
	f.Add("[12:00:00 TRACE SMAPI] a\ncontinuation")
	f.Add("[12:00:00 DEBUG  Mod] x\r\ny")
	f.Add("no header")
	f.Add("[12:00:00 ALERT SMAPI] loaded save 'TheFarm'")
	f.Add("[12:00:00 WARN  SMAPI] Mods go here: ~/Mods")
	f.Add("")
	f.Fuzz(func(t *testing.T, log string) {
		var p Parser
		for line := range strings.SplitSeq(log, "\n") {
			e := p.Parse(strings.TrimRight(line, "\r"))
			if e.Level == "" {
				t.Fatal("empty level")
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
	f.Fuzz(func(t *testing.T, log string) {
		folder, ok := LoadedSave(log)
		if ok != (folder != "") {
			t.Fatalf("ok=%v folder=%q", ok, folder)
		}
	})
}

func FuzzParsePerformanceReport(f *testing.F) {
	f.Add("| 1 | Mod | 10 | 1.00 | 0.50 | 2.00 | 5 |")
	f.Add("Mod | 10 | 1.00")
	f.Add("| Totals | 10 | 1.00 | 0.50 | 2.00 |")
	f.Add("")
	f.Add("|")
	f.Fuzz(func(t *testing.T, blob string) {
		_ = ParsePerformanceReport(strings.Split(blob, "\n"))
	})
}
