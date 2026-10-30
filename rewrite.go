package smapi

import "encoding/json"

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
	type dep struct {
		ID             string `json:"UniqueID"`
		IsRequired     *bool  `json:"IsRequired,omitempty"`
		MinimumVersion string `json:"MinimumVersion,omitempty"`
	}
	var deps []dep
	if v, ok := doc["Dependencies"]; ok {
		_ = json.Unmarshal(v, &deps)
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
		delete(doc, "Dependencies")
	} else {
		b, err := json.Marshal(kept)
		if err != nil {
			return nil, err
		}
		doc["Dependencies"] = b
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func idSet(ids []string) map[string]bool {
	out := map[string]bool{}
	for _, id := range ids {
		out[FoldID(id)] = true
	}
	return out
}
