package smapi

import (
	"testing"
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
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := Parse(b)
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
			if repo, ok := GitHubUpdateKey(k); ok && !repoPattern.MatchString(repo) {
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
