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

// NexusUpdateKey returns the mod page of a "Nexus:1234" or "Nexus:1234@subkey" update key; the page is a positive number.
func NexusUpdateKey(key string) (int, bool) {
	v, ok := updateKey(key, "nexus")
	if !ok {
		return 0, false
	}
	n, ok := positive(v)
	return n, ok
}

// positive reads a number written as digits only, so "+7", "-5", "0" and "" are refused.
func positive(v string) (int, bool) {
	if v == "" || strings.Trim(v, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	return n, err == nil && n > 0
}

// CurseForgeUpdateKey returns the project id of a "CurseForge:<id>" update key; the id is a positive number, written
// without leading zeros however the key spells it.
func CurseForgeUpdateKey(key string) (string, bool) {
	v, ok := updateKey(key, "curseforge")
	if !ok {
		return "", false
	}
	n, ok := positive(v)
	return strconv.Itoa(n), ok
}

// GitHubUpdateKey returns the "owner/repo" of a "GitHub:owner/repo" update key. A repo that would move
// the path of a URL built from it ("..", "o/r?x", "o/r/s") is refused.
func GitHubUpdateKey(key string) (string, bool) {
	repo, ok := updateKey(key, "github")
	if !ok || !gh.ValidRepo(repo) {
		return "", false
	}
	return repo, true
}
