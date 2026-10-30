// Package smapi reads SMAPI mod manifests and orders mods the way SMAPI's mod resolver does.
//
// [Parse] reads a manifest.json as leniently as SMAPI's Json.NET reader. [Resolve] returns mods in
// SMAPI's load order. Mod ids (UniqueID) are case-insensitive and compared with [FoldID].
package smapi
