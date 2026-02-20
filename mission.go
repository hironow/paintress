package paintress

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
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
