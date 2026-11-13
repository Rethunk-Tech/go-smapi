package smapi

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestRewriteDependenciesAddsAndDropsOptionals(t *testing.T) {
	in := `{
		// kept as data, not as a comment
		"uniqueid": "A.Mod", "Version": "1.0",
		"Dependencies": [
			{"UniqueID": "Need.Req"},
			{"UniqueID": "Old.Opt", "IsRequired": false},
			{"UniqueID": "Keep.Opt", "IsRequired": false, "MinimumVersion": "2"},
		],
	}`
	out, err := RewriteDependencies([]byte(in), []string{"new.opt", "need.req", "keep.opt"}, []string{"OLD.OPT", "keep.opt"})
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(out)
	if err != nil {
		t.Fatalf("output unreadable: %v\n%s", err, out)
	}
	got := map[string]Dependency{}
	for _, d := range m.Dependencies {
		got[d.UniqueID] = d
	}
	if len(got) != 3 || !got["Need.Req"].IsRequired || got["new.opt"].IsRequired || got["Keep.Opt"].MinimumVersion != "2" || m.Version != "1.0" {
		t.Fatalf("rewritten = %+v\n%s", m, out)
	}
}

func TestRewriteDependenciesDropsTheKeyWhenEmpty(t *testing.T) {
	out, err := RewriteDependencies([]byte(`{"UniqueID":"A","Dependencies":[{"UniqueID":"B","IsRequired":false}]}`), nil, []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := Parse(out)
	if m.UniqueID != "A" || len(m.Dependencies) != 0 || slices.Contains([]byte(string(out)), '[') {
		t.Fatalf("out = %s", out)
	}
}

func TestRewriteDependenciesRefusesAnUnreadableManifest(t *testing.T) {
	if _, err := RewriteDependencies([]byte(`{"Name":"no id"}`), []string{"x"}, nil); err == nil {
		t.Fatal("rewrote a manifest without a UniqueID")
	}
}

func TestRewriteDependenciesFollowsTheKeysCasing(t *testing.T) {
	in := `{"uniqueid":"a.b","name":"x","dependencies":[{"UniqueID":"Opt.Mod","IsRequired":false},{"UniqueID":"Need.Req"}]}`
	out, err := RewriteDependencies([]byte(in), []string{"Want.Mod"}, []string{"Opt.Mod"})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(out, &doc); err != nil || len(doc) != 3 {
		t.Fatalf("keys %v, %v\n%s", doc, err, out)
	}
	m, err := Parse(out)
	got := map[string]bool{}
	for _, d := range m.Dependencies {
		got[d.UniqueID] = true
	}
	if err != nil || len(got) != 2 || !got["Need.Req"] || !got["Want.Mod"] {
		t.Fatalf("dependencies %v, %v\n%s", got, err, out)
	}
}

func TestRewriteDependenciesKeepsStringIsRequired(t *testing.T) {
	raw := []byte(`{
		"Name": "Test",
		"Author": "A",
		"Version": "1.0.0",
		"UniqueID": "Test.Mod",
		"Dependencies": [
			{"UniqueID": "Opt", "IsRequired": "false"},
			{"UniqueID": "Req", "IsRequired": "True"}
		]
	}`)
	out, err := RewriteDependencies(raw, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	var opt, req *Dependency
	for i := range m.Dependencies {
		d := &m.Dependencies[i]
		switch FoldID(d.UniqueID) {
		case "opt":
			opt = d
		case "req":
			req = d
		}
	}
	if opt == nil || req == nil {
		t.Fatalf("deps = %v", m.Dependencies)
	}
	if opt.IsRequired {
		t.Error("Opt required")
	}
	if !req.IsRequired {
		t.Error("Req optional")
	}
}

func TestRewriteDependenciesKeepsEntriesItCannotRead(t *testing.T) {
	in := `{"UniqueID":"a.b","Dependencies":["junk",{"UniqueID":5},{"UniqueID":"x.y","MinimumVersion":"1.0"}]}`
	out, err := RewriteDependencies([]byte(in), []string{"n.o"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct{ Dependencies []json.RawMessage }
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Dependencies) != 4 || string(doc.Dependencies[0]) != `"junk"` {
		t.Errorf("dependencies = %s", doc.Dependencies)
	}
}
