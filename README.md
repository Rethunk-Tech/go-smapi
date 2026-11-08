<h1 align="center">go-smapi</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

Reads SMAPI mod manifests and orders mods the way SMAPI's mod resolver does. Package `smapi`, module `github.com/Rethunk-Tech/go-smapi`.

A manifest reader that accepts what SMAPI accepts (comments, trailing commas, BOMs, single quotes, any key casing) and a resolver that returns mods in SMAPI's load order. Neither touches the filesystem, so both are safe to point at downloaded mods.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-smapi
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- `Parse` reads a `manifest.json` the way SMAPI's Json.NET reader does, with a size bound and no panics on hostile input.
- Content packs list their framework as one required dependency, however the manifest declares it.
- `Dependency.Check` and `Manifest.CheckSMAPI` answer met, unmet or unknown for a dependency or `MinimumAPIVersion`, so an unreadable version is never shown as met.
- `Resolve` orders mods as SMAPI's mod resolver does and flags missing requirements and dependency loops.
- `NexusUpdateKey`, `CurseForgeUpdateKey` and `GitHubUpdateKey` read update keys; the GitHub one refuses a repo that would move a URL's path.
- `RewriteDependencies` adds or drops optional dependencies and keeps every other key.
- `logline.Parser` reads `[HH:MM:SS LEVEL Mod]` headers and continuation lines; `LoadedSave`, `ModsPath` and `LogOwnedBy` read the loaded save, the `Mods go here:` path, and whether a log belongs to a mods folder; `ParsePerformanceReport` reads performance-report rows.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
