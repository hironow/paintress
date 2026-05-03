package filter

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// MissionText returns the mission rules of engagement for the given language and mode.
// It delegates to the PromptRegistry, selecting the appropriate pre-rendered variant.
func MissionText(reg *PromptRegistry, lang string, isWaveMode bool) string {
	name := missionPromptName(lang, isWaveMode)
	entry, err := reg.Get(name)
	if err != nil {
		panic(fmt.Sprintf("mission prompt %q not found: %v", name, err))
	}
	return entry.Template
}

// RenderExpeditionPrompt renders the expedition prompt template for the given language.
// Conditional sections (scope, environment, context, inbox) are pre-rendered by this
// function and injected as simple string variables into the YAML template.
func RenderExpeditionPrompt(reg *PromptRegistry, lang string, data domain.PromptData) string {
	name := expeditionPromptName(lang)
	numberStr := fmt.Sprintf("%d", data.Number)

	vars := map[string]string{
		"number":                     numberStr,
		"timestamp":                  data.Timestamp,
		"lumina_section":             data.LuminaSection,
		"gradient_section":           data.GradientSection,
		"scope_section":              renderScopeSection(lang, data),
		"environment_section":        renderEnvironmentSection(lang, data),
		"context_section":            renderContextSection(lang, data),
		"inbox_section":              renderInboxSection(lang, data),
		"mission_section":            data.MissionSection,
		"is_wave_mode":               boolFlag(data.WaveTarget != nil),
		"has_event_sourced_contract": boolFlag(data.HasEventSourcedContract),
	}

	result, err := reg.Expand(name, vars)
	if err != nil {
		panic(fmt.Sprintf("expedition prompt expansion failed: %v", err))
	}
	return result
}

// missionPromptName returns the registry key for the mission prompt.
func missionPromptName(lang string, isWaveMode bool) string {
	l := lang
	switch l {
	case "ja", "fr":
		// keep
	default:
		l = "en"
	}
	mode := "linear"
	if isWaveMode {
		mode = "wave"
	}
	return "mission_" + l + "_" + mode
}

// expeditionPromptName returns the registry key for the expedition prompt.
func expeditionPromptName(lang string) string {
	switch lang {
	case "ja":
		return "expedition_ja"
	case "fr":
		return "expedition_fr"
	default:
		return "expedition_en"
	}
}

func boolFlag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// renderScopeSection pre-renders the Wave Target or Linear Scope block.
func renderScopeSection(lang string, data domain.PromptData) string {
	if data.WaveTarget != nil {
		return renderWaveTargetSection(lang, data)
	}
	if data.LinearTeam != "" {
		return renderLinearScopeSection(lang, data)
	}
	return ""
}

func renderWaveTargetSection(lang string, data domain.PromptData) string {
	var b strings.Builder
	switch lang {
	case "ja":
		b.WriteString("\n## Wave ターゲット\n\n")
		b.WriteString(fmt.Sprintf("- **ステップ:** `%s` — %s\n", data.WaveTarget.ID, data.WaveTarget.Title))
		if data.WaveTarget.Description != "" {
			b.WriteString(fmt.Sprintf("- **ウェーブ:** %s\n", data.WaveTarget.Description))
		}
		if data.WaveTarget.Acceptance != "" {
			b.WriteString(fmt.Sprintf("- **完了条件:** %s\n", data.WaveTarget.Acceptance))
		}
		b.WriteString("- このステップのみを実装すること。他のステップや無関係な issue には触れないこと。")
	case "fr":
		b.WriteString("\n## Cible Wave\n\n")
		b.WriteString(fmt.Sprintf("- **Étape :** `%s` — %s\n", data.WaveTarget.ID, data.WaveTarget.Title))
		if data.WaveTarget.Description != "" {
			b.WriteString(fmt.Sprintf("- **Wave :** %s\n", data.WaveTarget.Description))
		}
		if data.WaveTarget.Acceptance != "" {
			b.WriteString(fmt.Sprintf("- **Critères d'acceptation :** %s\n", data.WaveTarget.Acceptance))
		}
		b.WriteString("- Implémenter UNIQUEMENT cette étape. Ne pas toucher aux autres étapes.")
	default:
		b.WriteString("\n## Wave Target\n\n")
		b.WriteString(fmt.Sprintf("- **Step:** `%s` — %s\n", data.WaveTarget.ID, data.WaveTarget.Title))
		if data.WaveTarget.Description != "" {
			b.WriteString(fmt.Sprintf("- **Wave:** %s\n", data.WaveTarget.Description))
		}
		if data.WaveTarget.Acceptance != "" {
			b.WriteString(fmt.Sprintf("- **Acceptance Criteria:** %s\n", data.WaveTarget.Acceptance))
		}
		b.WriteString("- Implement ONLY this step. Do not work on other steps or unrelated issues.")
	}
	return b.String()
}

func renderLinearScopeSection(lang string, data domain.PromptData) string {
	var b strings.Builder
	switch lang {
	case "ja":
		b.WriteString("\n## Linear スコープ\n\n")
		b.WriteString(fmt.Sprintf("- チーム: `%s`\n", data.LinearTeam))
		if data.LinearProject != "" {
			b.WriteString(fmt.Sprintf("- プロジェクト: `%s`\n", data.LinearProject))
		}
		if data.LinearProject != "" {
			b.WriteString("- このチームとプロジェクトの issue のみを対象にすること。")
		} else {
			b.WriteString("- このチームの issue のみを対象にすること。")
		}
	case "fr":
		b.WriteString("\n## Scope Linear\n\n")
		b.WriteString(fmt.Sprintf("- Équipe : `%s`\n", data.LinearTeam))
		if data.LinearProject != "" {
			b.WriteString(fmt.Sprintf("- Projet : `%s`\n", data.LinearProject))
		}
		if data.LinearProject != "" {
			b.WriteString("- Ne traiter que les issues de cette équipe et de ce projet.")
		} else {
			b.WriteString("- Ne traiter que les issues de cette équipe.")
		}
	default:
		b.WriteString("\n## Linear Scope\n\n")
		b.WriteString(fmt.Sprintf("- Team: `%s`\n", data.LinearTeam))
		if data.LinearProject != "" {
			b.WriteString(fmt.Sprintf("- Project: `%s`\n", data.LinearProject))
		}
		if data.LinearProject != "" {
			b.WriteString("- Only pick issues from this team and project.")
		} else {
			b.WriteString("- Only pick issues from this team.")
		}
	}
	return b.String()
}

// renderEnvironmentSection pre-renders the environment lines block.
func renderEnvironmentSection(lang string, data domain.PromptData) string {
	var b strings.Builder
	b.WriteString("- " + data.ReserveSection + "\n")
	switch lang {
	case "ja":
		b.WriteString(fmt.Sprintf("- Base branch: `%s`\n", data.BaseBranch))
		if data.DevURL != "" {
			b.WriteString(fmt.Sprintf("- Dev server: `%s`（verify mission で使用、既に起動済み）\n", data.DevURL))
		}
		if data.WaveTarget != nil {
			b.WriteString("- GitHub: Pull Request と Issues の両方を使用する\n")
		} else {
			b.WriteString("- GitHub: Pull Request 専用で使用する（GitHub Issues は使わないこと）\n")
			b.WriteString("- Linear: Issue 専用で使用する（取得・ステータス更新・コメント）\n")
		}
		b.WriteString("- 注意: MCP ツールは遅延ロード — 使用前に `ToolSearch` で利用可能なツールを検索すること")
	case "fr":
		b.WriteString(fmt.Sprintf("- Branche de base : `%s`\n", data.BaseBranch))
		if data.DevURL != "" {
			b.WriteString(fmt.Sprintf("- Serveur dev : `%s` (utilisé pour les missions verify, déjà lancé)\n", data.DevURL))
		}
		if data.WaveTarget != nil {
			b.WriteString("- GitHub : utiliser pour les Pull Requests et les Issues\n")
		} else {
			b.WriteString("- GitHub : utiliser uniquement pour les Pull Requests (ne PAS utiliser les GitHub Issues)\n")
			b.WriteString("- Linear : utiliser uniquement pour les Issues (récupérer, mettre à jour le statut, commenter)\n")
		}
		b.WriteString("- Note : les outils MCP sont en chargement différé — utiliser `ToolSearch` pour découvrir les outils disponibles avant utilisation")
	default:
		b.WriteString(fmt.Sprintf("- Base branch: `%s`\n", data.BaseBranch))
		if data.DevURL != "" {
			b.WriteString(fmt.Sprintf("- Dev server: `%s` (used for verify missions, already running)\n", data.DevURL))
		}
		if data.WaveTarget != nil {
			b.WriteString("- GitHub: use for both Pull Requests and Issues\n")
		} else {
			b.WriteString("- GitHub: use for Pull Requests only (do NOT use GitHub Issues)\n")
			b.WriteString("- Linear: use for Issues only (retrieve, update status, comment)\n")
		}
		b.WriteString("- Note: MCP tools are lazy-loaded — use `ToolSearch` to discover available tools before use")
	}
	return b.String()
}

// renderContextSection pre-renders the injected context block with header.
func renderContextSection(lang string, data domain.PromptData) string {
	if data.ContextSection == "" {
		return ""
	}
	var header string
	switch lang {
	case "ja":
		header = "\n## 注入コンテキスト\n\n以下はこの Expedition 向けに提供されたコンテキスト。権威ある指針として扱うこと：\n\n"
	case "fr":
		header = "\n## Contexte injecté\n\nLe contexte suivant a été fourni pour cette expédition. Traitez-le comme une directive faisant autorité :\n\n"
	default:
		header = "\n## Injected Context\n\nThe following context has been provided for this expedition. Treat it as authoritative guidance:\n\n"
	}
	return header + data.ContextSection
}

// renderInboxSection pre-renders the D-Mail inbox block with header.
func renderInboxSection(lang string, data domain.PromptData) string {
	if data.InboxSection == "" {
		return ""
	}
	var header string
	switch lang {
	case "ja":
		header = "\n## D-Mail 受信箱\n\n以下のD-Mailが受信箱に届いている。種別に応じて行動すること（specification = 仕様に従い実装、implementation-feedback = 教訓を取り入れる）：\n\n"
	case "fr":
		header = "\n## Boîte de réception D-Mail\n\nLes d-mails suivants ont été livrés dans votre boîte de réception. Lisez-les et agissez selon leur type (specification = implémentez-le, implementation-feedback = intégrez la leçon) :\n\n"
	default:
		header = "\n## D-Mail Inbox\n\nThe following d-mails have been delivered to your inbox. Read and act on them according to their kind (specification = implement it, implementation-feedback = incorporate the lesson):\n\n"
	}
	return header + data.InboxSection
}
