package domain

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/expedition_*.md.tmpl
var expeditionFS embed.FS

// ExpeditionTemplates is the parsed expedition prompt templates.
var ExpeditionTemplates = template.Must(
	template.ParseFS(expeditionFS, "templates/expedition_*.md.tmpl"),
)

//go:embed templates/mission_*.md.tmpl
var missionFS embed.FS

var missionTemplates = template.Must(
	template.ParseFS(missionFS, "templates/mission_*.md.tmpl"),
)

// MissionText returns the mission rules of engagement in the active language.
func MissionText() string {
	tmplName := "mission_en.md.tmpl"
	switch Lang {
	case "ja":
		tmplName = "mission_ja.md.tmpl"
	case "fr":
		tmplName = "mission_fr.md.tmpl"
	}

	var buf strings.Builder
	if err := missionTemplates.ExecuteTemplate(&buf, tmplName, nil); err != nil {
		panic(fmt.Sprintf("mission template execution failed: %v", err))
	}
	return buf.String()
}

// PromptData holds all dynamic values injected into the expedition prompt template.
type PromptData struct {
	Number          int
	Timestamp       string
	Bt              string // "`"
	Cb              string // "```"
	LuminaSection   string
	GradientSection string
	ReserveSection  string
	BaseBranch      string
	DevURL          string
	ContextSection  string
	InboxSection    string
	LinearTeam      string
	LinearProject   string
	MissionSection  string
}

// ContextDir returns the path to the context injection directory.
func ContextDir(continent string) string {
	return filepath.Join(continent, ".expedition", "context")
}
