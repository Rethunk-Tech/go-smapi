package smapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-semverish"
)

// FileName is the manifest file SMAPI looks for.
const FileName = "manifest.json"

// MaxManifestSize is the largest manifest [Parse] reads. Real manifests are a few kilobytes; the bound
// keeps a crafted file from costing memory in proportion to its size.
const MaxManifestSize = 4 << 20

// ErrTooLarge is returned by [Parse] for input over [MaxManifestSize].
var ErrTooLarge = errors.New("manifest is too large")

// Dependency is one mod another mod needs.
type Dependency struct {
	UniqueID string
	// MinimumVersion is the oldest version of the dependency that works; empty for any.
	MinimumVersion string
	// IsRequired is true when the manifest omits IsRequired, as in SMAPI.
	IsRequired bool
}

// Status is the outcome of comparing a version requirement with an installed version.
type Status int

const (
	// Unknown means a version could not be ordered, so the requirement is neither met nor unmet.
	Unknown Status = iota
	Met
	Unmet
)

// checkMinimum orders installed against minimum; an empty minimum is always Met.
func checkMinimum(minimum, installed string) Status {
	if minimum == "" {
		return Met
	}
	c, ok := semverish.Compare(installed, minimum)
	switch {
	case !ok:
		return Unknown
	case c < 0:
		return Unmet
	}
	return Met
}

// Check compares an installed version of the dependency with MinimumVersion. It is Unknown when either
// version does not parse; an empty MinimumVersion is Met whatever is installed.
func (d Dependency) Check(installed string) Status { return checkMinimum(d.MinimumVersion, installed) }

// SatisfiedBy reports whether an installed version of the dependency meets MinimumVersion. It is false
// only when both versions parse and the installed one is older, so an unreadable version never blocks a mod.
func (d Dependency) SatisfiedBy(installed string) bool { return d.Check(installed) != Unmet }

// Manifest holds the manifest.json fields a mod manager reads.
type Manifest struct {
	Name        string
	Author      string
	Version     string
	UniqueID    string
	Description string
	UpdateKeys  []string
	// EntryDll is the C# mod's assembly file, relative to the mod folder; empty for content packs.
	EntryDll string
	// ContentPackFor is the framework's UniqueID when the manifest declares a ContentPackFor.
	ContentPackFor string
	// Dependencies lists Dependencies[] and, as a required entry, the ContentPackFor framework.
	Dependencies []Dependency
	// UpdateCautionMessage is the author's warning to show before updating to this version.
	UpdateCautionMessage string
	// MinimumGameVersion is the oldest game version the mod runs on, when it declares one.
	MinimumGameVersion string
	// MinimumAPIVersion is the oldest SMAPI the mod runs on, as written; empty when it declares none.
	MinimumAPIVersion string
	// DeleteOldVersion asks an updater to remove the previous version's files rather than overlay them.
	DeleteOldVersion bool
}

// CheckSMAPI compares the installed SMAPI version with MinimumAPIVersion, which SMAPI refuses to load a
// mod below. It is Unknown when either version does not parse; no MinimumAPIVersion is Met.
func (m Manifest) CheckSMAPI(installed string) Status {
	return checkMinimum(m.MinimumAPIVersion, installed)
}

// FoldID is the key a UniqueID is compared by: SMAPI ids are case-insensitive and may carry stray spaces.
func FoldID(uniqueID string) string { return strings.ToLower(strings.TrimSpace(uniqueID)) }

// Parse reads a manifest tolerating a BOM or UTF-16/32 text, comments, trailing commas, single quotes,
// unquoted names, curly quotes and any key casing. A manifest without a UniqueID is an error, since
// nothing can refer to that mod. Input over [MaxManifestSize] is [ErrTooLarge].
func Parse(b []byte) (Manifest, error) {
	if len(b) > MaxManifestSize {
		return Manifest{}, ErrTooLarge
	}
	raw, err := readObject(b)
	if err != nil {
		return Manifest{}, err
	}
	m := Manifest{
		Name:                 text(raw, "name"),
		Author:               text(raw, "author"),
		Version:              version(field(raw, "version")),
		UniqueID:             text(raw, "uniqueid"),
		Description:          text(raw, "description"),
		EntryDll:             text(raw, "entrydll"),
		UpdateKeys:           texts(field(raw, "updatekeys")),
		UpdateCautionMessage: text(raw, "updatecautionmessage"),
		DeleteOldVersion:     boolean(field(raw, "deleteoldversion")),
		MinimumGameVersion:   text(raw, "minimumgameversion"),
		MinimumAPIVersion:    text(raw, "minimumapiversion"),
	}
	if d, ok := dependency(field(raw, "contentpackfor")); ok {
		m.ContentPackFor = d.UniqueID
	}
	m.Dependencies = dependencies(raw)
	if m.UniqueID == "" {
		return Manifest{}, errors.New("manifest has no UniqueID")
	}
	return m, nil
}

// curlyQuotes turns typographic quotes into straight ones, the second try SMAPI makes on a manifest that fails to parse.
var curlyQuotes = strings.NewReplacer("“", `"`, "”", `"`)

// readObject reads b as the JSON object Newtonsoft would, or says where it stops being one.
func readObject(b []byte) (map[string]json.RawMessage, error) {
	clean := lenientjson.Clean(b)
	var raw map[string]json.RawMessage
	err := json.Unmarshal(clean, &raw)
	if err != nil && bytes.ContainsAny(b, "“”") {
		retry := lenientjson.Clean([]byte(curlyQuotes.Replace(string(b))))
		if json.Unmarshal(retry, &raw) == nil {
			return raw, nil
		}
	}
	if err == nil {
		if raw == nil {
			return nil, errors.New("manifest is not a JSON object")
		}
		return raw, nil
	}
	if syntax, ok := errors.AsType[*json.SyntaxError](err); ok {
		return nil, fmt.Errorf("invalid JSON at line %d: %s", 1+bytes.Count(clean[:min(int(syntax.Offset), len(clean))], []byte("\n")), syntax.Error())
	}
	if _, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		return nil, errors.New("manifest is not a JSON object")
	}
	return nil, err
}

func texts(v json.RawMessage) []string {
	var items []json.RawMessage
	if json.Unmarshal(v, &items) != nil {
		return nil
	}
	var out []string
	for _, it := range items {
		var s string
		if json.Unmarshal(it, &s) == nil && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func dependency(v json.RawMessage) (Dependency, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(v, &obj) != nil {
		return Dependency{}, false
	}
	d := Dependency{UniqueID: text(obj, "uniqueid"), MinimumVersion: text(obj, "minimumversion"), IsRequired: true}
	if b := field(obj, "isrequired"); b != nil {
		if required, ok := flexBool(b); ok {
			d.IsRequired = required
		}
	}
	return d, d.UniqueID != ""
}

// dependencies reads Dependencies[] and ContentPackFor, skipping entries without a UniqueID.
func dependencies(raw map[string]json.RawMessage) []Dependency {
	var out []Dependency
	var items []json.RawMessage
	if json.Unmarshal(field(raw, "dependencies"), &items) == nil {
		for _, it := range items {
			if d, ok := dependency(it); ok {
				out = append(out, d)
			}
		}
	}
	if d, ok := dependency(field(raw, "contentpackfor")); ok {
		d.IsRequired = true
		// Packs often list their framework in Dependencies too; it is one dependency, so it is checked and shown once.
		i := slices.IndexFunc(out, func(o Dependency) bool { return FoldID(o.UniqueID) == FoldID(d.UniqueID) })
		switch {
		case i < 0:
			out = append(out, d)
		case out[i].MinimumVersion == "":
			out[i].IsRequired, out[i].MinimumVersion = true, d.MinimumVersion
		default:
			// SMAPI checks the Dependencies entry and the ContentPackFor entry separately, so both minimums
			// must hold; the higher one implies the other.
			out[i].IsRequired = true
			switch c, ok := semverish.Compare(d.MinimumVersion, out[i].MinimumVersion); {
			case !ok && !readable(d.MinimumVersion):
				// An unreadable minimum stays, so Check reports Unknown instead of Met.
				out[i].MinimumVersion = d.MinimumVersion
			case ok && c > 0:
				out[i].MinimumVersion = d.MinimumVersion
			}
		}
	}
	return out
}

// key is the spelling raw holds name under, in any casing. When a manifest repeats a name in several casings, the
// all-lower-case spelling wins, then the alphabetically first, so one manifest always reads the same.
func key(raw map[string]json.RawMessage, name string) (string, bool) {
	best, found := "", false
	for k := range raw {
		if !strings.EqualFold(k, name) {
			continue
		}
		if !found || k == name || best != name && k < best {
			best, found = k, true
		}
	}
	return best, found
}

func readable(v string) bool {
	_, ok := semverish.Compare(v, v)
	return ok
}

func field(raw map[string]json.RawMessage, name string) json.RawMessage {
	k, _ := key(raw, name)
	return raw[k]
}

func text(raw map[string]json.RawMessage, name string) string {
	var s string
	if json.Unmarshal(field(raw, name), &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func boolean(v json.RawMessage) bool {
	b, _ := flexBool(v)
	return b
}

// flexBool reads a JSON boolean or the strings "true"/"false" in any case, as SMAPI's Json.NET reader does.
func flexBool(v json.RawMessage) (value, ok bool) {
	if json.Unmarshal(v, &value) == nil {
		return value, true
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

// version accepts a string or SMAPI's legacy {MajorVersion, MinorVersion, PatchVersion, Build} object.
func version(v json.RawMessage) string {
	var s string
	if json.Unmarshal(v, &s) == nil {
		return strings.TrimSpace(s)
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(v, &obj) != nil {
		return ""
	}
	num := func(name string) int {
		var n int
		_ = json.Unmarshal(field(obj, name), &n)
		return n
	}
	// SMAPI prints the legacy object's Build as a prerelease tag.
	return semverish.Version{
		Major: num("majorversion"), Minor: num("minorversion"), Patch: num("patchversion"), Pre: text(obj, "build"),
	}.String()
}
