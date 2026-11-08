# AGENTS.md

Package `smapi` reads SMAPI `manifest.json` files and computes the load order SMAPI's mod resolver would. Package `logline` parses SMAPI log text. It imports `go-lenientjson` for Json.NET-style leniency, `go-semverish` for SMAPI version order, `go-github` for repo-name validation and `go-safefs` for log path comparison, and does no file or network I/O. Mod ids are plain `UniqueID` strings; a caller with its own id type maps at the boundary.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH.

## Shape

- `Parse` is the single path from bytes to a `Manifest`; `RewriteDependencies` goes through it first, so only a manifest `Parse` accepts is ever rewritten. Both read the object through `readObject`, which carries the curly-quote retry SMAPI makes.
- Ids compare through `FoldID` (trim, lower-case) everywhere, because SMAPI matches `UniqueID` that way. Never compare ids with `==`.
- A ContentPackFor framework is folded into `Dependencies` as required (merged with a matching entry), as SMAPI's resolver does, and `Resolve` repeats the merge for callers that build `Mod` by hand.
- `MaxManifestSize` bounds `Parse` because the lenient pass copies its input; the bound is far above any real manifest and is policy, so a caller may read less.
- `Resolve` marks each mod `checking` before descending, so a loop ends at the mod that meets itself rather than recursing; recursion depth is at most the number of mods.
- `Dependency.SatisfiedBy` is false only when both versions parse, so a version SMAPI would reject never hides a mod the caller can still run.
- Versions: `go-semverish` is SMAPI's non-standard-tolerant comparison (`allowNonStandard: true`, which SMAPI uses for dependency minimums). It also accepts leading zeros, which a strict SMAPI manifest `Version` does not; `Version` is kept as written and never rejected.
- `logline` is SMAPI's log format only: `Parser` remembers the last header so a continuation inherits time, level and mod; a first line with no header is INFO. `LoadedSave` reads only SMAPI-mod lines. `LogOwnedBy` expands `~` with the caller's home and compares through go-safefs `SamePath`. `ParsePerformanceReport` reads pipe-delimited performance tables.
- Out of scope, by design: scanning a folder tree and loader-managed mod lists are caller policy, not SMAPI rules.

## Files

| File | Holds |
| --- | --- |
| `doc.go` | Package doc |
| `manifest.go` | `Manifest`, `Dependency`, `Parse`, `FoldID`, lenient field readers |
| `updatekey.go` | Nexus, CurseForge and GitHub update key readers |
| `rewrite.go` | `RewriteDependencies` |
| `loadorder.go` | `Mod`, `Row`, `Resolve` |
| `logline/logline.go` | `Entry`, `Level`, `Parser`, `LoadedSave`, `ModsPath`, `LogOwnedBy` |
| `logline/perf.go` | `PerformanceRow`, `ParsePerformanceReport` |
