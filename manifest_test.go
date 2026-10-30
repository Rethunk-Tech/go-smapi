package smapi

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseLenient(t *testing.T) {
	cases := map[string]struct {
		in   string
		want Manifest
	}{
		"bom comments trailing commas": {
			"\xef\xbb\xbf{\n  // the mod\n  \"Name\": \"Content Patcher\", /* inline */\n  \"Author\": \"Pathoschild\",\n  \"Version\": \"2.0.0\",\n  \"Description\": \"http://x // not a comment\",\n  \"UniqueID\": \"Pathoschild.ContentPatcher\",\n  \"Dependencies\": [ { \"UniqueID\": \"a\", }, ],\n}\n",
			Manifest{Name: "Content Patcher", Author: "Pathoschild", Version: "2.0.0", UniqueID: "Pathoschild.ContentPatcher", Description: "http://x // not a comment", Dependencies: []Dependency{{UniqueID: "a", IsRequired: true}}},
		},
		"lowercase keys": {
			`{"name":"N","author":"A","version":"1.2","uniqueid":"A.N","description":"d"}`,
			Manifest{Name: "N", Author: "A", Version: "1.2", UniqueID: "A.N", Description: "d"},
		},
		"dependencies and content pack": {
			`{"UniqueID":"A.P","UpdateKeys":["Nexus:1"," ",7],"Dependencies":[{"UniqueID":"B.Req","MinimumVersion":"1.2-beta"},{"uniqueid":"B.Opt","isrequired":false},{"MinimumVersion":"1"},"junk"],"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"2.0"}}`,
			Manifest{
				UniqueID: "A.P", UpdateKeys: []string{"Nexus:1"}, ContentPackFor: "Pathoschild.ContentPatcher",
				Dependencies: []Dependency{
					{UniqueID: "B.Req", MinimumVersion: "1.2-beta", IsRequired: true}, {UniqueID: "B.Opt"}, {UniqueID: "Pathoschild.ContentPatcher", MinimumVersion: "2.0", IsRequired: true},
				},
			},
		},
		"framework listed twice": {
			`{"UniqueID":"A.P","Dependencies":[{"UniqueID":"pathoschild.contentpatcher","IsRequired":false}],"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"2.0"}}`,
			Manifest{
				UniqueID: "A.P", ContentPackFor: "Pathoschild.ContentPatcher",
				Dependencies: []Dependency{{UniqueID: "pathoschild.contentpatcher", MinimumVersion: "2.0", IsRequired: true}},
			},
		},
		"legacy version object": {
			`{"Name":"Old","Author":"A","UniqueID":"A.Old","Version":{"MajorVersion":1,"MinorVersion":2,"PatchVersion":3,"Build":"beta"}}`,
			Manifest{Name: "Old", Author: "A", Version: "1.2.3-beta", UniqueID: "A.Old"},
		},
		"update hints case insensitive": {
			`{"UniqueID":"A.B","UpdateCautionMessage":"Read the release notes","DELETEOLDVERSION":true,"MinimumGameVersion":"1.6.0"}`,
			Manifest{UniqueID: "A.B", UpdateCautionMessage: "Read the release notes", DeleteOldVersion: true, MinimumGameVersion: "1.6.0"},
		},
		"escaped quote in string": {
			`{"Name":"Say \"hi\" // ok","UniqueID":"A.B","Version":"1.0"}`,
			Manifest{Name: `Say "hi" // ok`, Version: "1.0", UniqueID: "A.B"},
		},
	}
	for name, c := range cases {
		got, err := Parse([]byte(c.in))
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v, %v; want %+v", name, got, err, c.want)
		}
	}
	for _, bad := range []string{`{"Name":"x"}`, `not json`, `{"UniqueID":"a"`} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func TestParseReadsStringIsRequired(t *testing.T) {
	m, err := Parse([]byte(`{"UniqueID": "A", "Dependencies": [
		{"UniqueID": "Opt", "IsRequired": "false"},
		{"UniqueID": "Req", "IsRequired": "True"},
		{"UniqueID": "Plain"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Opt": false, "Req": true, "Plain": true}
	for _, d := range m.Dependencies {
		if w, ok := want[d.UniqueID]; ok && d.IsRequired != w {
			t.Errorf("%s IsRequired = %v, want %v", d.UniqueID, d.IsRequired, w)
		}
	}
}

func TestParseNewtonsoftForms(t *testing.T) {
	utf16le := []byte{0xff, 0xfe}
	for _, c := range []byte(`{'UniqueID': 'A.U16'}`) {
		utf16le = append(utf16le, c, 0)
	}
	for name, tc := range map[string]struct {
		in   []byte
		want string
	}{
		"single quotes":         {[]byte(`{'Name':'N','UniqueID':'A.Single'}`), "A.Single"},
		"unquoted names":        {[]byte(`{Name:"N", UniqueID: "A.Bare",}`), "A.Bare"},
		"utf16":                 {utf16le, "A.U16"},
		"raw newline in string": {[]byte("{\"Description\":\"two\nlines\",\"UniqueID\":\"A.NL\"}"), "A.NL"},
		"curly quotes":          {[]byte("{“UniqueID”: “A.Curly”}"), "A.Curly"},
	} {
		m, err := Parse(tc.in)
		if err != nil || m.UniqueID != tc.want {
			t.Errorf("%s: %+v, %v", name, m, err)
		}
	}
}

func TestParseErrorsSayWhy(t *testing.T) {
	for in, want := range map[string]string{
		"{\n\"UniqueID\": \"A\",\n\"Name\" \"x\"\n}": "invalid JSON at line 3",
		`{"Name":"x"}`: "manifest has no UniqueID",
		`[1]`:          "not a JSON object",
		`null`:         "not a JSON object",
	} {
		if _, err := Parse([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", in, err, want)
		}
	}
}

func TestParseRefusesAnOversizedManifest(t *testing.T) {
	big := `{"UniqueID":"A","Description":"` + strings.Repeat("x", MaxManifestSize) + `"}`
	if _, err := Parse([]byte(big)); err != ErrTooLarge {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestParseSurvivesDeepNesting(t *testing.T) {
	deep := `{"UniqueID":"A","X":` + strings.Repeat("[", 100000) + strings.Repeat("]", 100000) + `}`
	if _, err := Parse([]byte(deep)); err == nil {
		t.Fatal("a manifest nested 100000 deep was accepted")
	}
}

func TestSatisfiedByComparesSMAPIVersions(t *testing.T) {
	for _, c := range []struct {
		min, have string
		want      bool
	}{
		{"", "1.0", true},
		{"1.2", "1.2.0", true},
		{"1.2", "1.1.9", false},
		{"1.0-beta", "1.0", true},
		{"1.0", "1.0-beta", false},
		{"1.0", "not a version", true},
	} {
		if got := (Dependency{MinimumVersion: c.min}).SatisfiedBy(c.have); got != c.want {
			t.Errorf("min %q have %q = %v, want %v", c.min, c.have, got, c.want)
		}
	}
}

func TestFoldIDIgnoresCaseAndSpaces(t *testing.T) {
	if FoldID(" Pathoschild.ContentPatcher ") != "pathoschild.contentpatcher" {
		t.Fatal(FoldID(" Pathoschild.ContentPatcher "))
	}
}

func TestGitHubUpdateKeyRefusesWhatMovesTheURL(t *testing.T) {
	for _, repo := range []string{"me/..", "./mod", "../..", "me/.", "o/r?x", "o/r#x", "o/r/s", "o /r"} {
		if _, ok := GitHubUpdateKey("GitHub:" + repo); ok {
			t.Errorf("GitHubUpdateKey accepted %q", repo)
		}
	}
	if r, ok := GitHubUpdateKey(" github : owner/repo.name "); !ok || r != "owner/repo.name" {
		t.Errorf("valid key = %q, %v", r, ok)
	}
}

func TestUpdateKeys(t *testing.T) {
	if n, ok := NexusUpdateKey("Nexus:1234@sub"); !ok || n != 1234 {
		t.Errorf("nexus = %d, %v", n, ok)
	}
	if _, ok := NexusUpdateKey("Nexus:abc"); ok {
		t.Error("nexus accepted a non-number")
	}
	if _, ok := NexusUpdateKey("GitHub:1"); ok {
		t.Error("nexus accepted another site")
	}
	if id, ok := CurseForgeUpdateKey("CurseForge:42@x"); !ok || id != "42" {
		t.Errorf("curseforge = %q, %v", id, ok)
	}
	if _, ok := CurseForgeUpdateKey("CurseForge:0"); ok {
		t.Error("curseforge accepted 0")
	}
}
