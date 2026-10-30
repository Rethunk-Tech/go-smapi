package smapi

import (
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
