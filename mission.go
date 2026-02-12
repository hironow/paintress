package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/mission_*.md.tmpl
var missionFS embed.FS

var missionTemplates = template.Must(
	template.ParseFS(missionFS, "templates/mission_*.md.tmpl"),
)

// MissionPath returns the path to the mission file on the Continent.
func MissionPath(continent string) string {
	return filepath.Join(continent, ".expedition", "mission.md")
}

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

// WriteMission writes the mission file to the Continent in the active language.
func WriteMission(continent string) error {
	return os.WriteFile(MissionPath(continent), []byte(MissionText()), 0644)
}
