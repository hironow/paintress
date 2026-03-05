package platform

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/hironow/paintress/internal/domain"
)

//go:embed templates/expedition_*.md.tmpl
var expeditionFS embed.FS

var expeditionTemplates = template.Must(
	template.ParseFS(expeditionFS, "templates/expedition_*.md.tmpl"),
)

//go:embed templates/mission_*.md.tmpl
var missionFS embed.FS

var missionTemplates = template.Must(
	template.ParseFS(missionFS, "templates/mission_*.md.tmpl"),
)

// RenderExpeditionPrompt renders the expedition prompt template for the given language.
func RenderExpeditionPrompt(lang string, data domain.PromptData) string {
	tmplName := "expedition_en.md.tmpl"
	switch lang {
	case "ja":
		tmplName = "expedition_ja.md.tmpl"
	case "fr":
		tmplName = "expedition_fr.md.tmpl"
	}

	var buf strings.Builder
	if err := expeditionTemplates.ExecuteTemplate(&buf, tmplName, data); err != nil {
		panic(fmt.Sprintf("prompt template execution failed: %v", err))
	}
	return buf.String()
}

// MissionText returns the mission rules of engagement for the given language.
func MissionText(lang string) string {
	tmplName := "mission_en.md.tmpl"
	switch lang {
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
