package smapi

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

// Mod is the manifest fields the resolver needs.
type Mod struct {
	UniqueID string
	Name     string
	// Folder is the mod's folder; its name orders the walk, as SMAPI's scanner meets folders. Either OS separator is read.
	Folder string
	// Dependencies are in the order the manifest declares them. MinimumVersion is not read.
	Dependencies   []Dependency
	ContentPackFor string
}

// Row is one mod in load order.
type Row struct {
	// Position counts from 1.
	Position int
	UniqueID string
	Name     string
	// Required and Optional are the UniqueIDs the mod depends on, as declared.
	Required []string
	Optional []string
	// Dependents are the loaded mods that depend on this one, in folder order.
	Dependents []string
	// MissingRequired are required dependencies no listed mod provides.
	MissingRequired []string
	// Cycle is set on a mod that a dependency loop stopped, and on those depending on it.
	Cycle bool
}

type node struct {
	mod      Mod
	deps     []Dependency
	required []string
	optional []string
}

// byFolder is the order SMAPI's scanner meets mod folders on Windows: folder by folder, names without regard to case.
// Mods without a folder go by name, then id.
func byFolder(a, b node) int {
	sa := strings.Split(strings.ToLower(filepath.ToSlash(a.mod.Folder)), "/")
	sb := strings.Split(strings.ToLower(filepath.ToSlash(b.mod.Folder)), "/")
	if n := slices.Compare(sa, sb); n != 0 {
		return n
	}
	if n := cmp.Compare(strings.ToLower(a.mod.Name), strings.ToLower(b.mod.Name)); n != 0 {
		return n
	}
	return cmp.Compare(FoldID(a.mod.UniqueID), FoldID(b.mod.UniqueID))
}

type state int

const (
	queued state = iota
	checking
	sorted
	failed
)

// Resolve returns mods in SMAPI's load order, which is ModResolver.ProcessDependencies
// (Pathoschild/SMAPI develop e5b8bf3, src/SMAPI/Framework/ModLoading/ModResolver.cs:221-391): it walks the mods
// in folder order and loads each mod's dependencies (required, optional, and the mod a content pack is for)
// just before it, depth first. Duplicate UniqueIDs keep the first. A mod that misses a required dependency, or
// that a dependency loop or a failed required dependency stops, is placed where SMAPI gives up on it.
// The walk visits each mod once, so a loop ends at the first mod that meets itself.
func Resolve(mods []Mod) []Row {
	nodes := make([]node, 0, len(mods))
	index := map[string]int{}
	for _, m := range mods {
		key := FoldID(m.UniqueID)
		if m.UniqueID == "" {
			continue
		}
		if _, ok := index[key]; ok {
			continue
		}
		n := node{mod: m}
		seen := map[string]bool{}
		for _, d := range m.Dependencies {
			if d.UniqueID != "" && !seen[FoldID(d.UniqueID)] {
				seen[FoldID(d.UniqueID)] = true
				n.deps = append(n.deps, d)
			}
		}
		// SMAPI lists the content pack's parent after the manifest's dependencies, always required.
		if cp := m.ContentPackFor; cp != "" {
			if i := slices.IndexFunc(n.deps, func(d Dependency) bool { return FoldID(d.UniqueID) == FoldID(cp) }); i >= 0 {
				n.deps[i].IsRequired = true
			} else {
				n.deps = append(n.deps, Dependency{UniqueID: cp, IsRequired: true})
			}
		}
		for _, d := range n.deps {
			if d.IsRequired {
				n.required = append(n.required, d.UniqueID)
			} else {
				n.optional = append(n.optional, d.UniqueID)
			}
		}
		index[key] = len(nodes)
		nodes = append(nodes, n)
	}
	find := func(id string) (int, bool) {
		i, ok := index[FoldID(id)]
		return i, ok
	}
	walk := make([]int, len(nodes))
	for i := range walk {
		walk[i] = i
	}
	slices.SortStableFunc(walk, func(a, b int) int { return byFolder(nodes[a], nodes[b]) })

	states := make([]state, len(nodes))
	cycle := make([]bool, len(nodes))
	ordered := make([]int, 0, len(nodes))
	var visit func(i int) state
	visit = func(i int) state {
		if states[i] != queued {
			return states[i]
		}
		n := nodes[i]
		for _, d := range n.deps {
			if _, ok := find(d.UniqueID); !ok && d.IsRequired {
				ordered = append(ordered, i)
				states[i] = failed
				return failed
			}
		}
		states[i] = checking
		for _, d := range n.deps {
			j, ok := find(d.UniqueID)
			if !ok {
				continue
			}
			if states[j] == checking {
				ordered = append(ordered, i)
				cycle[i] = true
				states[i] = failed
				return failed
			}
			if visit(j) == failed && d.IsRequired {
				ordered = append(ordered, i)
				cycle[i] = cycle[j]
				states[i] = failed
				return failed
			}
		}
		ordered = append(ordered, i)
		states[i] = sorted
		return sorted
	}
	for _, i := range walk {
		visit(i)
	}

	dependents := make([][]string, len(nodes))
	for _, n := range nodes {
		for _, d := range n.deps {
			if p, ok := find(d.UniqueID); ok {
				dependents[p] = append(dependents[p], n.mod.UniqueID)
			}
		}
	}
	for i := range dependents {
		slices.SortFunc(dependents[i], func(a, b string) int {
			return byFolder(nodes[index[FoldID(a)]], nodes[index[FoldID(b)]])
		})
	}

	out := make([]Row, 0, len(ordered))
	for pos, i := range ordered {
		n := nodes[i]
		missing := make([]string, 0)
		for _, id := range n.required {
			if _, ok := find(id); !ok {
				missing = append(missing, id)
			}
		}
		out = append(out, Row{
			Position:        pos + 1,
			UniqueID:        n.mod.UniqueID,
			Name:            n.mod.Name,
			Required:        n.required,
			Optional:        n.optional,
			Dependents:      dependents[i],
			MissingRequired: missing,
			Cycle:           cycle[i],
		})
	}
	return out
}
