# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- `Parse` and `RewriteDependencies` take the bytes of a `manifest.json` from any downloaded mod. A crafted file must not panic, nest deeply enough to exhaust the stack, or cost memory beyond a small multiple of its size; input over `MaxManifestSize` is refused.
- `GitHubUpdateKey` takes update keys from those manifests. A crafted key must not yield a repo (`..`, `o/r?x`, `o/r/s`) that moves the path of a URL built from it; `NexusUpdateKey` and `CurseForgeUpdateKey` return numbers only.
- `Resolve` takes dependency lists from those manifests. A dependency loop, a self-dependency or a dependency on a missing mod must not loop forever or drop a mod.

## What is out of scope

- Reading the file: the caller opens `manifest.json` and bounds how much it reads from disk or an archive.
- Treating a `UniqueID`, `EntryDll` or update key as a path or URL: the caller validates before use; this package only guarantees the properties above.
- Deciding which mods to trust or load: `Resolve` orders, it does not vet.
