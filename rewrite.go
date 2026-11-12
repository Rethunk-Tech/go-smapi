package smapi

import (
	"encoding/json"
	"strings"
)

// RewriteDependencies returns manifest.json text with each UniqueID in want added as an optional dependency
// (unless it is already listed) and each optional dependency in drop that is not in want removed. Other keys, in
// any casing, are kept; the output is strict JSON, so comments in the input are lost.
func RewriteDependencies(raw []byte, want, drop []string) ([]byte, error) {
	if _, err := Parse(raw); err != nil {
		return nil, err
	}
	doc, err := readObject(raw)
	if err != nil {
		return nil, err
	}
	var deps []dep
	// The key keeps the casing the manifest used, and a repeat in another casing goes, so one list is left.
	name, found := key(doc, "dependencies")
	if !found {
		name = "Dependencies"
	}
	deps = decodeDeps(doc[name])
	for k := range doc {
		if k != name && strings.EqualFold(k, name) {
			delete(doc, k)
		}
	}
	dropSet, wantSet := idSet(drop), idSet(want)
	kept := make([]dep, 0, len(deps)+len(want))
	seen := map[string]bool{}
	for _, d := range deps {
		low := FoldID(d.ID)
		required := d.IsRequired == nil || *d.IsRequired
		if !required && dropSet[low] && !wantSet[low] {
			continue
		}
		kept = append(kept, d)
		seen[low] = true
	}
	off := false
	for _, id := range want {
		if seen[FoldID(id)] {
			continue
		}
		kept = append(kept, dep{ID: id, IsRequired: &off})
		seen[FoldID(id)] = true
	}
	if len(kept) == 0 {
		delete(doc, name)
	} else {
		b, err := json.Marshal(kept)
		if err != nil {
			return nil, err
		}
		doc[name] = b
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

type dep struct {
	ID             string `json:"UniqueID"`
	IsRequired     *bool  `json:"IsRequired,omitempty"`
	MinimumVersion string `json:"MinimumVersion,omitempty"`
}

type rawDep struct {
	ID             string          `json:"UniqueID"`
	IsRequired     json.RawMessage `json:"IsRequired"`
	MinimumVersion string          `json:"MinimumVersion,omitempty"`
}

func decodeDeps(raw json.RawMessage) []dep {
	var in []rawDep
	if json.Unmarshal(raw, &in) != nil {
		return nil
	}
	out := make([]dep, 0, len(in))
	for _, d := range in {
		out = append(out, dep{
			ID:             d.ID,
			IsRequired:     requiredFlag(d.IsRequired),
			MinimumVersion: d.MinimumVersion,
		})
	}
	return out
}

func requiredFlag(raw json.RawMessage) *bool {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return &b
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		t := true
		return &t
	case "false":
		f := false
		return &f
	}
	return nil
}

func idSet(ids []string) map[string]bool {
	out := map[string]bool{}
	for _, id := range ids {
		out[FoldID(id)] = true
	}
	return out
}
