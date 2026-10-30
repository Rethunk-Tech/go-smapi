package smapi

import (
	"strconv"
	"strings"

	gh "github.com/Rethunk-Tech/go-github"
)

// updateKey splits "Site:value@subkey" into the value when Site matches site (any case).
func updateKey(key, site string) (value string, ok bool) {
	s, rest, found := strings.Cut(key, ":")
	if !found || !strings.EqualFold(strings.TrimSpace(s), site) {
		return "", false
	}
	rest, _, _ = strings.Cut(rest, "@")
	return strings.TrimSpace(rest), true
}

// NexusUpdateKey returns the mod page of a "Nexus:1234" or "Nexus:1234@subkey" update key.
func NexusUpdateKey(key string) (int, bool) {
	v, ok := updateKey(key, "nexus")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	return n, err == nil
}

// CurseForgeUpdateKey returns the project id of a "CurseForge:<id>" update key; the id is a positive number.
func CurseForgeUpdateKey(key string) (string, bool) {
	v, ok := updateKey(key, "curseforge")
	if !ok {
		return "", false
	}
	n, err := strconv.Atoi(v)
	return v, err == nil && n > 0
}

// GitHubUpdateKey returns the "owner/repo" of a "GitHub:owner/repo" update key. A repo that would move
// the path of a URL built from it ("..", "o/r?x", "o/r/s") is refused.
func GitHubUpdateKey(key string) (string, bool) {
	s, rest, found := strings.Cut(key, ":")
	rest = strings.TrimSpace(rest)
	if !found || !strings.EqualFold(strings.TrimSpace(s), "github") || !gh.ValidRepo(rest) {
		return "", false
	}
	return rest, true
}
