package smapi

import (
	"fmt"
	"slices"
	"testing"
)

func needs(ids ...string) []Dependency {
	out := make([]Dependency, len(ids))
	for i, id := range ids {
		out[i] = Dependency{UniqueID: id, IsRequired: true}
	}
	return out
}

func optional(ids ...string) []Dependency {
	out := make([]Dependency, len(ids))
	for i, id := range ids {
		out[i] = Dependency{UniqueID: id}
	}
	return out
}

func ids(rows []Row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.UniqueID
	}
	return out
}

func TestDependencyLoadsBeforeDependent(t *testing.T) {
	got := ids(Resolve([]Mod{
		{UniqueID: "Z.Addon", Name: "Aaa Addon", Dependencies: needs("Z.Lib")},
		{UniqueID: "Z.Lib", Name: "Zzz Library"},
		{UniqueID: "A.Solo", Name: "Middle"},
	}))
	lib, addon := slices.Index(got, "Z.Lib"), slices.Index(got, "Z.Addon")
	if lib < 0 || addon < 0 || lib > addon {
		t.Fatalf("library should load before addon: %v", got)
	}
}

func TestContentPackLoadsAfterFramework(t *testing.T) {
	got := ids(Resolve([]Mod{
		{UniqueID: "Pack.A", Name: "Aaa Pack", ContentPackFor: "Pathoschild.ContentPatcher"},
		{UniqueID: "Pathoschild.ContentPatcher", Name: "Zzz Content Patcher"},
	}))
	if !slices.Equal(got, []string{"Pathoschild.ContentPatcher", "Pack.A"}) {
		t.Fatalf("framework then pack: %v", got)
	}
}

func TestOptionalDependencyLoadsBeforeDependent(t *testing.T) {
	got := ids(Resolve([]Mod{
		{UniqueID: "Z.Host", Name: "Aaa Host", Dependencies: optional("Z.Opt")},
		{UniqueID: "Z.Opt", Name: "Zzz Optional"},
	}))
	if !slices.Equal(got, []string{"Z.Opt", "Z.Host"}) {
		t.Fatalf("optional dep first: %v", got)
	}
}

func TestCycleDetection(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "A.One", Name: "One", Dependencies: needs("A.Two")},
		{UniqueID: "A.Two", Name: "Two", Dependencies: needs("A.One")},
		{UniqueID: "B.Free", Name: "Free"},
	})
	if len(rows) != 3 || rows[0].UniqueID != "B.Free" || rows[0].Cycle {
		t.Fatalf("free mod first, not a cycle: %+v", rows)
	}
	if !rows[1].Cycle || !rows[2].Cycle {
		t.Fatalf("cycle pair should be flagged: %+v", rows)
	}
}

func TestSelfDependencyEnds(t *testing.T) {
	rows := Resolve([]Mod{{UniqueID: "A.Self", Dependencies: needs("a.self")}})
	if len(rows) != 1 || !rows[0].Cycle {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestMissingRequiredDependency(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "A.Mod", Name: "Mod", Dependencies: append(needs("Missing.Lib"), optional("Also.Gone")...)},
	})
	if len(rows) != 1 || !slices.Equal(rows[0].MissingRequired, []string{"Missing.Lib"}) {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestAlphabeticalWhenIndependent(t *testing.T) {
	got := ids(Resolve([]Mod{
		{UniqueID: "B.Mod", Name: "Beta"},
		{UniqueID: "A.Mod", Name: "Alpha"},
	}))
	if !slices.Equal(got, []string{"A.Mod", "B.Mod"}) {
		t.Fatalf("by name: %v", got)
	}
}

// SMAPI loads a mod's dependency just before it, not where the dependency's own folder name would put it.
func TestADependencyLoadsJustBeforeItsMod(t *testing.T) {
	got := ids(Resolve([]Mod{
		{UniqueID: "B", Name: "B", Folder: "Mods/B"},
		{UniqueID: "Z", Name: "Z", Folder: "Mods/Z"},
		{UniqueID: "A", Name: "A", Folder: "Mods/A", Dependencies: needs("Z")},
	}))
	if !slices.Equal(got, []string{"Z", "A", "B"}) {
		t.Fatalf("order = %v, want Z, A, B", got)
	}
}

func TestIDsMatchWithoutRegardToCaseAndDuplicatesKeepTheFirst(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "A.Mod", Name: "First", Dependencies: needs("b.lib")},
		{UniqueID: "a.mod", Name: "Second"},
		{UniqueID: "B.Lib", Name: "Lib"},
	})
	if len(rows) != 2 || rows[1].Name != "First" || len(rows[1].MissingRequired) != 0 || !slices.Equal(rows[0].Dependents, []string{"A.Mod"}) {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestLongChainAndBigLoopTerminate(t *testing.T) {
	const n = 20000
	chain := make([]Mod, n)
	for i := range chain {
		chain[i] = Mod{UniqueID: fmt.Sprint("M", i), Dependencies: needs(fmt.Sprint("M", (i+1)%n))}
	}
	rows := Resolve(chain)
	if len(rows) != n || !rows[0].Cycle {
		t.Fatalf("rows = %d, first cycle = %v", len(rows), rows[0].Cycle)
	}
}

func TestResolveMarksDependentFailedWhenRequiredDepFailed(t *testing.T) {
	rows := Resolve([]Mod{
		{UniqueID: "A", Dependencies: []Dependency{{UniqueID: "B", IsRequired: true}}},
		{UniqueID: "B", Dependencies: []Dependency{{UniqueID: "Missing.Lib", IsRequired: true}}},
	})
	var a, b *Row
	for i := range rows {
		switch FoldID(rows[i].UniqueID) {
		case "a":
			a = &rows[i]
		case "b":
			b = &rows[i]
		}
	}
	if a == nil || b == nil {
		t.Fatalf("rows = %v", rows)
	}
	if a.Cycle {
		t.Error("A Cycle")
	}
	if len(a.MissingRequired) != 0 {
		t.Errorf("A MissingRequired = %v", a.MissingRequired)
	}
	if !a.Failed {
		t.Error("A not failed")
	}
	if len(b.MissingRequired) != 1 || b.MissingRequired[0] != "Missing.Lib" {
		t.Errorf("B MissingRequired = %v", b.MissingRequired)
	}
	if b.Cycle {
		t.Error("B Cycle")
	}
}
