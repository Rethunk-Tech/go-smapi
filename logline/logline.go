package logline

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// Level is a SMAPI console log level.
type Level string

// SMAPI's console log levels.
const (
	Trace Level = "TRACE"
	Debug Level = "DEBUG"
	Info  Level = "INFO"
	Warn  Level = "WARN"
	Error Level = "ERROR"
	Alert Level = "ALERT"
)

// Entry is one line of SMAPI's log. A line that continues an earlier entry's multi-line
// message repeats that entry's time, level and mod and has Cont set.
type Entry struct {
	Time    string
	Level   Level
	Mod     string
	Message string
	Cont    bool
}

// SMAPI writes `[HH:MM:SS LEVEL  Mod] message`, padding the level to five characters.
var header = regexp.MustCompile(`^\[(\d\d:\d\d:\d\d) (TRACE|DEBUG|INFO|WARN|ERROR|ALERT) *([^\]]*)\] ?(.*)$`)

var loadedSaveRe = regexp.MustCompile(`(?i)(?:Context:\s*)?loaded save '([^']+)'`)

// LoadedSave returns the save folder SMAPI loaded, from the last matching SMAPI line in log.
func LoadedSave(log string) (string, bool) {
	folder := ""
	var p Parser
	for l := range strings.SplitSeq(log, "\n") {
		if l = strings.TrimRight(l, "\r"); l == "" {
			continue
		}
		e := p.Parse(l)
		if e.Mod != "SMAPI" {
			continue
		}
		if m := loadedSaveRe.FindStringSubmatch(e.Message); m != nil {
			folder = m[1]
		}
	}
	return folder, folder != ""
}

// ModsPath returns the mods folder SMAPI's log says it loaded, as SMAPI wrote it, and false
// when the log does not say. SMAPI writes the home folder as ~ (`PathUtilities.AnonymizePathForDisplay`).
func ModsPath(log string) (string, bool) {
	for l := range strings.Lines(log) {
		m := header.FindStringSubmatch(strings.TrimRight(l, "\r\n"))
		if m == nil || strings.TrimSpace(m[3]) != "SMAPI" {
			continue
		}
		if path, ok := strings.CutPrefix(m[4], "Mods go here: "); ok {
			return path, true
		}
	}
	return "", false
}

// LogOwnedBy reports whether log was written by SMAPI loading modsDir. home is the folder
// SMAPI shortened to ~. A log that does not name its mods folder belongs to no one.
func LogOwnedBy(log, home, modsDir string) bool {
	path, ok := ModsPath(log)
	if !ok {
		return false
	}
	if rest, ok := strings.CutPrefix(path, "~"); ok && (rest == "" || rest[0] == '/' || rest[0] == '\\') {
		path = home + rest
	}
	return safefs.SamePath(path, modsDir)
}

// Parser turns log lines into entries, remembering the last header so continuation lines
// can inherit it.
type Parser struct {
	last Entry
	seen bool
}

// Parse returns the entry for one SMAPI line. A line that is not a header continues the
// previous entry; with none yet it stands alone as an INFO line.
func (p *Parser) Parse(line string) Entry {
	if m := header.FindStringSubmatch(line); m != nil {
		p.last = Entry{Time: m[1], Level: Level(m[2]), Mod: strings.TrimSpace(m[3]), Message: m[4]}
		p.seen = true
		return p.last
	}
	if !p.seen {
		return Entry{Level: Info, Message: line}
	}
	e := p.last
	e.Message, e.Cont = line, true
	return e
}
