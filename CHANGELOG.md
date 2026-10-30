# Changelog

Notable changes to `go-smapi`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-10-30

### Added

- `Parse` reads a SMAPI `manifest.json` with Json.NET's leniency (comments, trailing commas, BOMs and UTF-16/32, single quotes, unquoted names, curly quotes, any key casing, legacy version objects), bounded by `MaxManifestSize`.
- `Dependency.SatisfiedBy` and `FoldID` compare minimum versions and mod ids as SMAPI does.
- `NexusUpdateKey`, `CurseForgeUpdateKey` and `GitHubUpdateKey` read update keys.
- `RewriteDependencies` adds and drops optional dependencies in a manifest.
- `Resolve` returns mods in SMAPI's load order, flagging missing required dependencies and dependency loops.

[Unreleased]: https://github.com/Rethunk-Tech/go-smapi/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Rethunk-Tech/go-smapi/releases/tag/v0.1.0
