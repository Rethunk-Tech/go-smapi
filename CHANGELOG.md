# Changelog

Notable changes to `go-smapi`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-11-07

### Fixed

- `Parse` reads a manifest that repeats a key in different casings the same way every time (the all-lower-case spelling, else the alphabetically first), instead of by map order.
- `RewriteDependencies` finds a `dependencies` key in any casing, and leaves one list.
- A `ContentPackFor` minimum version that cannot be read stays, so `Check` is `Unknown` instead of `Met`.
- `NexusUpdateKey` and `CurseForgeUpdateKey` refuse `+7`, `-5` and `0`, and `CurseForgeUpdateKey` returns the id without leading zeros.

## [0.2.0] - 2026-10-31

### Added

- `Manifest.MinimumAPIVersion` reads the manifest's `MinimumApiVersion` as written.
- `Dependency.Check` and `Manifest.CheckSMAPI` compare an installed version with a minimum and answer `Met`, `Unmet` or `Unknown` (a version that does not parse), so a caller can show "unknown" rather than "met"; `SatisfiedBy` is unchanged.

### Fixed

- `Parse` keeps the higher minimum when a manifest names its `ContentPackFor` framework in `Dependencies` too; it kept the `Dependencies` one, so a pack could pass `SatisfiedBy` below its framework minimum.

## [0.1.0] - 2026-10-30

### Added

- `Parse` reads a SMAPI `manifest.json` with Json.NET's leniency (comments, trailing commas, BOMs and UTF-16/32, single quotes, unquoted names, curly quotes, any key casing, legacy version objects), bounded by `MaxManifestSize`.
- `Dependency.SatisfiedBy` and `FoldID` compare minimum versions and mod ids as SMAPI does.
- `NexusUpdateKey`, `CurseForgeUpdateKey` and `GitHubUpdateKey` read update keys.
- `RewriteDependencies` adds and drops optional dependencies in a manifest.
- `Resolve` returns mods in SMAPI's load order, flagging missing required dependencies and dependency loops.

[Unreleased]: https://github.com/Rethunk-Tech/go-smapi/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/Rethunk-Tech/go-smapi/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Rethunk-Tech/go-smapi/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Rethunk-Tech/go-smapi/releases/tag/v0.1.0
