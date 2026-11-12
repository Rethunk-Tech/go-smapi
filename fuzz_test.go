package smapi

import (
	"errors"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"Name":"Mod","Author":"A","Version":"1.0.0","Description":"d","UniqueID":"A.Mod"}`))
	f.Add([]byte(`{"Name":"Mod","Author":"A","Version":"1.0.0","Description":"d","UniqueID":"A.Mod","Dependencies":[{"UniqueID":"B.Lib","MinimumVersion":"1.0.0"}]}`))
	f.Add([]byte(`{"Name":"Pack","Author":"A","Version":"1.0.0","Description":"d","UniqueID":"A.Pack","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`))
	f.Add([]byte(`{"Name":"Mod","Author":"A","Version":"1.0.0","Description":"d","UniqueID":"A.Mod","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"},"Dependencies":[{"UniqueID":"Pathoschild.ContentPatcher","IsRequired":false}]}`))
	f.Add([]byte("\ufeff{\"Name\":\"Mod\",\"Author\":\"A\",\"Version\":\"1.0.0\",\"Description\":\"d\",\"UniqueID\":\"A.Mod\"}"))
	f.Add([]byte(`{'Name':'Mod','Author':'A','Version':'1.0.0','Description':'d','UniqueID':'A.Mod',}`))
	f.Add([]byte(`{Name:"Mod",Author:"A",Version:"1.0.0",Description:"d",UniqueID:"A.Mod"}`))
	f.Add([]byte(`{"name":"Mod","author":"A","version":"1.0.0","description":"d","uniqueid":"A.Mod"}`))
	f.Add([]byte(`{"Name":"Mod","Author":"A","Version":"1.0.0","Description":"d","UniqueID":" A.Mod "}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{`))
	f.Add([]byte(``))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := Parse(data)
		if err != nil {
			if len(data) > MaxManifestSize && !errors.Is(err, ErrTooLarge) {
				t.Fatalf("oversize: %v", err)
			}
			return
		}
		if m.UniqueID == "" {
			t.Fatal("accepted empty UniqueID")
		}
		if m.ContentPackFor != "" {
			fold := FoldID(m.ContentPackFor)
			found := false
			for _, d := range m.Dependencies {
				if FoldID(d.UniqueID) == fold && d.IsRequired {
					found = true
					break
				}
			}
			if !found {
				t.Fatal("ContentPackFor not folded as required dependency")
			}
		}
	})
}
