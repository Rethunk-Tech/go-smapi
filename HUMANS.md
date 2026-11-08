# HUMANS.md

Run and use `go-smapi`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-smapi
```

```go
package main

import (
 "fmt"
 "os"

 "github.com/Rethunk-Tech/go-smapi"
)

func main() {
 var mods []smapi.Mod
 for _, path := range os.Args[1:] {
  b, err := os.ReadFile(path)
  if err != nil {
   fmt.Fprintln(os.Stderr, err)
   continue
  }
  m, err := smapi.Parse(b)
  if err != nil {
   fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
   continue
  }
  mods = append(mods, smapi.Mod{
   UniqueID: m.UniqueID, Name: m.Name, Folder: path,
   Dependencies: m.Dependencies, ContentPackFor: m.ContentPackFor,
  })
 }
 for _, r := range smapi.Resolve(mods) {
  fmt.Println(r.Position, r.UniqueID, r.MissingRequired, r.Cycle)
 }
}
```

## Contracts

| Entry point | Contract |
| --- | --- |
| `Parse` | Reads Json.NET-lenient JSON; error when the text is not an object, has no `UniqueID`, or exceeds `MaxManifestSize` (`ErrTooLarge`). Dependencies without a `UniqueID` are skipped; `IsRequired` defaults to true. Never panics. |
| `Manifest.Version` | The string as written, or a legacy version object printed as `M.m.p[-build]`. Not validated. |
| `Dependency.SatisfiedBy` | False only when both versions parse and the installed one is older than `MinimumVersion`. |
| `Dependency.Check`, `Manifest.CheckSMAPI` | `Met`, `Unmet` or `Unknown`: `Unknown` when either version does not parse; an empty minimum (`MinimumVersion`, `MinimumAPIVersion`) is `Met`. SMAPI refuses a mod whose `MinimumApiVersion` is newer than the installed SMAPI. |
| `FoldID` | The comparison key for a `UniqueID`: trimmed, lower-cased. |
| `NexusUpdateKey`, `CurseForgeUpdateKey`, `GitHubUpdateKey` | Site name in any case, `@subkey` ignored; the GitHub key refuses anything but a plain `owner/repo`. |
| `RewriteDependencies` | Parses first (same errors), returns strict JSON, so comments are lost; other keys keep their casing and values. |
| `Resolve` | Pure and always terminates. Duplicate ids keep the first; a missing required dependency or loop never drops a mod, it flags the row. `Folder` accepts either separator. |
| `logline.Parser.Parse` | One SMAPI line. A non-header continues the previous entry; with none yet it is INFO. |
| `logline.LoadedSave` | The save folder from the last SMAPI `loaded save '…'` line; false when none. |
| `logline.ModsPath` | The `Mods go here:` path as SMAPI wrote it; false when the log does not say. |
| `logline.LogOwnedBy` | Whether the log's mods path is `modsDir`, expanding `~` with `home`. False when the log does not name a mods folder. |
| `logline.ParsePerformanceReport` | Pipe-delimited SMAPI performance tables into rows. |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
