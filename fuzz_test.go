package smapi

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	gh "github.com/Rethunk-Tech/go-github"
)

// FuzzParse feeds manifest.json files as mods ship them: BOMs, comments, trailing commas and odd key casing.
func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"\xef\xbb\xbf{\n  // comment\n  \"Name\": \"Mod\", \"Author\": \"A\", \"Version\": \"1.2.3\", \"UniqueID\": \"A.Mod\",\n  \"EntryDll\": \"Mod.dll\", \"UpdateKeys\": [\"Nexus:1234@sub\", \"GitHub:owner/repo\"],\n  \"Dependencies\": [{\"UniqueID\": \"B.Lib\", \"MinimumVersion\": \"2.0\", \"IsRequired\": \"false\"},],\n}",
		`{"uniqueid":"C.Pack","version":{"MajorVersion":1,"MinorVersion":0,"PatchVersion":0},"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"},"DeleteOldVersion":true}`,
		`{"UniqueID":"","UpdateKeys":"Nexus:1"}`,
		`{"UniqueID":"x","UpdateKeys":["GitHub:../..","GitHub:o/r?x","Nexus: 99999999999999999999"]}`,
		`/* unterminated`,
		`[]`,
		"{'UniqueID': 'A', Name: \"x\\\n\"}",
		"\xff\xfe{\x00\"\x00U\x00",
		"{“UniqueID”: “A”}",
		"\xef\xbb\xbf{\n  // the mod\n  \"Name\": \"Content Patcher\", /* inline */\n  \"Author\": \"Pathoschild\",\n  \"Version\": \"2.0.0\",\n  \"Description\": \"http://x // not a comment\",\n  \"UniqueID\": \"Pathoschild.ContentPatcher\",\n  \"Dependencies\": [ { \"UniqueID\": \"a\", }, ],\n}\n",
		`{"name":"N","author":"A","version":"1.2","uniqueid":"A.N","description":"d"}`,
		`{"UniqueID":"A.P","UpdateKeys":["Nexus:1"," ",7],"Dependencies":[{"UniqueID":"B.Req","MinimumVersion":"1.2-beta"},{"uniqueid":"B.Opt","isrequired":false},{"MinimumVersion":"1"},"junk"],"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"2.0"}}`,
		`{"Name":"Old","Author":"A","UniqueID":"A.Old","Version":{"MajorVersion":1,"MinorVersion":2,"PatchVersion":3,"Build":"beta"}}`,
		`{"UniqueID": "A", "Dependencies": [{"UniqueID": "Opt", "IsRequired": "false"}, {"UniqueID": "Req", "IsRequired": "True"}]}`,
		`{"UniqueID":"a.b","Version":1.2,"MinimumAPIVersion":4.1,"MinimumGameVersion":1.6}`,
		`{"Name":"x"}`,
		`not json`,
		`null`,
		`[1]`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := Parse(b)
		if len(b) > MaxManifestSize {
			if !errors.Is(err, ErrTooLarge) {
				t.Fatalf("oversize manifest: %v", err)
			}
			return
		}
		if err != nil {
			return
		}
		if m.UniqueID == "" {
			t.Fatal("accepted a manifest without a UniqueID")
		}
		for _, d := range m.Dependencies {
			if d.UniqueID == "" {
				t.Fatalf("dependency without a UniqueID: %+v", d)
			}
			_ = d.SatisfiedBy(m.Version)
		}
		for _, k := range m.UpdateKeys {
			_, _ = NexusUpdateKey(k)
			_, _ = CurseForgeUpdateKey(k)
			if repo, ok := GitHubUpdateKey(k); ok && !gh.ValidRepo(repo) {
				t.Fatalf("update key %q gives repo %q, which would move a URL's path", k, repo)
			}
		}
		// What a manager writes back must still be a manifest for the same mod.
		out, err := RewriteDependencies(b, []string{"Fuzz.Added"}, nil)
		if err != nil {
			t.Fatalf("rewrite of a parsed manifest: %v", err)
		}
		if again, err := Parse(out); err != nil || again.UniqueID != m.UniqueID {
			t.Fatalf("rewritten manifest: %+v, %v", again, err)
		}
		rows := Resolve([]Mod{{UniqueID: m.UniqueID, Name: m.Name, Dependencies: m.Dependencies, ContentPackFor: m.ContentPackFor}})
		if len(rows) != 1 {
			t.Fatalf("resolve gave %d rows for one mod", len(rows))
		}
	})
}

func FuzzRewriteDependencies(f *testing.F) {
	f.Add([]byte(`{"uniqueid":"A.Mod","Version":"1.0","Dependencies":[{"UniqueID":"Need.Req"},{"UniqueID":"Old.Opt","IsRequired":false}]}`), "new.opt", "OLD.OPT")
	f.Add([]byte(`{"UniqueID":"A","Dependencies":[{"UniqueID":"B","IsRequired":false}]}`), "", "b")
	f.Add([]byte(`{"Name":"no id"}`), "x", "")
	f.Add([]byte(`{"UniqueID":"Test.Mod","Dependencies":[{"UniqueID":"Opt","IsRequired":"false"},{"UniqueID":"Req","IsRequired":"True"}]}`), "", "")
	f.Add([]byte(`{"UniqueID":"a.b","Dependencies":[{"UniqueID":"B","IsRequired":false,"MinimumVersion":1.2}]}`), "b", "")
	f.Add([]byte(`{"UniqueID":"a.b","Dependencies":["junk",{"UniqueID":5},{"UniqueID":"x.y","MinimumVersion":"1.0"}]}`), "n.o", "")
	f.Fuzz(func(t *testing.T, raw []byte, want, drop string) {
		out, err := RewriteDependencies(raw, []string{want}, []string{drop})
		if err != nil {
			return
		}
		if !json.Valid(out) {
			t.Fatalf("rewrite is not JSON:\n%s", out)
		}
		orig, err := Parse(raw)
		if err != nil {
			t.Fatalf("rewrote what Parse refuses: %v", err)
		}
		m, err := Parse(out)
		if err != nil || m.UniqueID != orig.UniqueID {
			t.Fatalf("rewritten UniqueID = %q, %v; want %q", m.UniqueID, err, orig.UniqueID)
		}
		if id := FoldID(want); id != "" {
			found := false
			for _, d := range m.Dependencies {
				if FoldID(d.UniqueID) == id {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("want %q missing from %+v", want, m.Dependencies)
			}
		}
	})
}

func FuzzUpdateKeys(f *testing.F) {
	for _, s := range []string{
		"Nexus:1234@sub",
		"Nexus:abc",
		"GitHub:1",
		"CurseForge:42@x",
		"CurseForge:0",
		" github : owner/repo.name ",
		"GitHub:me/..",
		"GitHub:o/r?x",
		"GitHub:o/r#x",
		"GitHub:o/r/s",
		"Nexus:0",
		"Nexus:+7",
		"Nexus:-5",
		"Nexus:",
		"CurseForge:0123",
		"GitHub:owner/repo@YourMod",
		"GitHub:owner/foo/../bar@YourMod",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, key string) {
		n, nok := NexusUpdateKey(key)
		id, cok := CurseForgeUpdateKey(key)
		repo, gok := GitHubUpdateKey(key)
		matched := 0
		if nok {
			matched++
			if n <= 0 {
				t.Fatalf("Nexus accepted %d from %q", n, key)
			}
			v, ok := updateKey(key, "nexus")
			if !ok {
				t.Fatalf("Nexus matched a non-Nexus key %q", key)
			}
			parsed, err := strconv.Atoi(v)
			if err != nil || parsed != n {
				t.Fatalf("Nexus %q -> %d (value %q)", key, n, v)
			}
		} else if n != 0 {
			t.Fatalf("Nexus refused %q with n=%d", key, n)
		}
		if cok {
			matched++
			parsed, err := strconv.Atoi(id)
			if err != nil || parsed <= 0 || id != strconv.Itoa(parsed) {
				t.Fatalf("CurseForge %q -> %q", key, id)
			}
		}
		if gok {
			matched++
			if !gh.ValidRepo(repo) {
				t.Fatalf("GitHub %q -> %q, which would move a URL's path", key, repo)
			}
		}
		if matched > 1 {
			t.Fatalf("key %q matched more than one site", key)
		}
	})
}

func FuzzCheck(f *testing.F) {
	f.Add("1.0", "1.0.0")
	f.Add("2.0", "1.0")
	f.Add("", "1.0")
	f.Add("not-a-version", "1.0.0")
	f.Add("1.0.0", "not-a-version")
	f.Add("01.2.3", "1.2.3")
	f.Add("1.2-beta", "1.2.0")
	f.Add("4.1.0", "4.0.0")
	f.Fuzz(func(t *testing.T, minimum, installed string) {
		d := Dependency{UniqueID: "A", MinimumVersion: minimum}
		st := d.Check(installed)
		switch st {
		case Unknown, Met, Unmet:
		default:
			t.Fatalf("Check = %v", st)
		}
		if minimum == "" && st != Met {
			t.Fatalf("empty minimum: Check = %v", st)
		}
		if d.SatisfiedBy(installed) != (st != Unmet) {
			t.Fatalf("SatisfiedBy=%v Check=%v", d.SatisfiedBy(installed), st)
		}
		if (Manifest{MinimumAPIVersion: minimum}).CheckSMAPI(installed) != st {
			t.Fatal("CheckSMAPI disagrees with Check")
		}
	})
}
