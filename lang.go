package main

import "fmt"

// Lang is the active language.
var Lang = "en"

// Messages keyed by ID, with en/ja/fr variants.
var messages = map[string]map[string]string{
	// === Paintress ===
	"continent":         {"en": "Continent: %s", "ja": "Continent: %s", "fr": "Continent : %s"},
	"monolith_reads":    {"en": "Monolith: %s issues remaining", "ja": "Monolith: 残り %s issue", "fr": "Monolithe : %s issues restantes"},
	"max_expeditions":   {"en": "Max expeditions: %d", "ja": "最大遠征回数: %d", "fr": "Expéditions max : %d"},
	"timeout_info":      {"en": "Timeout: %ds", "ja": "タイムアウト: %d秒", "fr": "Délai : %ds"},
	"claude_cmd_info":   {"en": "Claude command: %s", "ja": "Claude コマンド: %s", "fr": "Commande Claude : %s"},
	"dry_run":           {"en": "DRY RUN", "ja": "DRY RUN", "fr": "SIMULATION"},
	"dry_run_prompt":    {"en": "DRY RUN — prompt: %s", "ja": "DRY RUN — prompt: %s", "fr": "SIMULATION — prompt : %s"},
	"interrupted":       {"en": "Interrupted", "ja": "中断", "fr": "Interrompu"},
	"all_complete":      {"en": "*** Monolith reads 0 — all issues complete!", "ja": "*** Monolith が 0 に — 全 issue 完了！", "fr": "*** Le Monolithe affiche 0 — toutes les issues sont terminées !"},
	"report_parse_fail": {"en": "Failed to parse Expedition Report", "ja": "Expedition Report のパース失敗", "fr": "Échec de l'analyse du rapport d'Expédition"},
	"output_check":      {"en": "Check output: %s/expedition-%03d-output.txt", "ja": "出力確認: %s/expedition-%03d-output.txt", "fr": "Vérifier la sortie : %s/expedition-%03d-output.txt"},
	"issue_skipped":     {"en": "%s skipped: %s", "ja": "%s スキップ: %s", "fr": "%s ignorée : %s"},
	"issue_failed":      {"en": "%s failed: %s", "ja": "%s 失敗: %s", "fr": "%s échouée : %s"},
	"gommage":           {"en": "Gommage — %d consecutive failures, halting", "ja": "Gommage — 連続 %d 回失敗、停止", "fr": "Gommage — %d échecs consécutifs, arrêt"},
	"cooldown":          {"en": "Cooldown 10s until next Expedition...", "ja": "次の Expedition まで 10秒...", "fr": "Repos 10s avant la prochaine Expédition..."},
	"expeditions_sent":  {"en": "Expeditions sent: %d", "ja": "送り出した Expedition: %d", "fr": "Expéditions envoyées : %d"},
	"success_count":     {"en": "Success: %d", "ja": "成功: %d", "fr": "Succès : %d"},
	"skipped_count":     {"en": "Skipped: %d", "ja": "スキップ: %d", "fr": "Ignorées : %d"},
	"failed_count":      {"en": "Failed: %d", "ja": "失敗: %d", "fr": "Échouées : %d"},
	"bugs_count":        {"en": "Bugs found: %d", "ja": "検出した不具合: %d", "fr": "Bugs trouvés : %d"},

	// === Expedition ===
	"departing":  {"en": "--- Expedition #%d departing ---", "ja": "--- Expedition #%d 出発 ---", "fr": "--- Expedition #%d en route ---"},
	"exp_failed": {"en": "Expedition #%d failed: %v", "ja": "Expedition #%d 失敗: %v", "fr": "Expédition #%d échouée : %v"},
	"sending":    {"en": "Sending Expeditioner (model: %s)...", "ja": "Expeditioner 送出中 (model: %s)...", "fr": "Envoi de l'Expéditionnaire (modèle : %s)..."},

	// === Flag ===
	"resting_at_flag":  {"en": "Resting at Flag — scanning Lumina...", "ja": "Flag で休息中 — Lumina スキャン...", "fr": "Repos au Drapeau d'Expédition — scan des Lumina..."},
	"lumina_extracted": {"en": "Extracted %d Lumina(s)", "ja": "Lumina %d 件を抽出", "fr": "%d Lumina extraite(s)"},

	// === Gradient ===
	"gradient_info": {"en": "Gradient: %s", "ja": "Gradient: %s", "fr": "Gradient : %s"},
	"party_info":    {"en": "Party: %s", "ja": "Party: %s", "fr": "Party : %s"},
	"grad_empty":    {"en": "[DEFEND] Gauge empty: start with a small, safe issue. Build momentum with one success first.", "ja": "[DEFEND] ゲージ空: 小さく確実な issue から始めよ。まず1つ成功させて勢いを取り戻す。", "fr": "[DEFEND] Gradient vide : commencez par une issue simple et sûre. Un premier succès pour prendre de l'élan."},
	"grad_normal":   {"en": "Normal: pick a standard-priority issue.", "ja": "通常: 標準的な優先度の issue を選ぶ。", "fr": "Normal : choisissez une issue de priorité standard."},
	"grad_high":     {"en": "[CHARGE] Gauge high: high-priority issues are fair game. You have momentum.", "ja": "[CHARGE] ゲージ高: 優先度の高い issue を選んでよい。勢いに乗っている。", "fr": "[CHARGE] Gradient haut : les issues prioritaires sont accessibles. Vous avez de l'élan."},
	"grad_attack":   {"en": "[GRADIENT ATTACK] Take on the most complex, highest-priority issue. The gauge is full — time for the big challenge.", "ja": "[GRADIENT ATTACK] 最も複雑で優先度の高い issue に挑戦せよ。ゲージ満タン。", "fr": "[GRADIENT ATTACK] Affrontez l'issue la plus complexe et prioritaire. Le Gradient est plein — c'est l'heure du grand défi."},

	// === Reserve ===
	"reserve_activated": {"en": "[RESERVE] %s -> %s (rate limit detected)", "ja": "[RESERVE] %s -> %s (rate limit 検知)", "fr": "[RESERVE] %s -> %s (limite de débit détectée)"},
	"reserve_forced":    {"en": "[RESERVE] Forced: %s -> %s", "ja": "[RESERVE] 強制発動: %s -> %s", "fr": "[RESERVE] Forcé : %s -> %s"},
	"primary_recovered": {"en": "[RESERVE] Primary recovered: %s -> %s (cooldown expired)", "ja": "[RESERVE] Primary 復帰: %s -> %s (cooldown 終了)", "fr": "[RESERVE] Primaire rétabli : %s -> %s (cooldown écoulé)"},

	// === Dev Server ===
	"devserver_already": {"en": "Dev server already running: %s", "ja": "Dev server 起動済み: %s", "fr": "Serveur dev deja en cours : %s"},
	"devserver_start":   {"en": "Starting dev server: %s (dir: %s)", "ja": "Dev server 起動: %s (dir: %s)", "fr": "Démarrage du serveur dev : %s (dir : %s)"},
	"devserver_ready":   {"en": "Dev server ready: %s", "ja": "Dev server ready: %s", "fr": "Serveur dev prêt : %s"},
	"devserver_stop":    {"en": "Stopping dev server...", "ja": "Dev server 停止中...", "fr": "Arrêt du serveur dev..."},
	"devserver_timeout": {"en": "Dev server startup timeout (60s)", "ja": "Dev server 起動タイムアウト (60秒)", "fr": "Délai de démarrage du serveur dev (60s)"},
	"devserver_warn":    {"en": "Dev server: %v (may affect verify missions)", "ja": "Dev server: %v (verify mission に影響の可能性)", "fr": "Serveur dev : %v (peut affecter les missions de vérification)"},

	// === QA ===
	"qa_all_pass": {"en": "All checks passed", "ja": "全項目 Pass", "fr": "Tous les contrôles réussis"},
	"qa_bugs":     {"en": "%d bug(s) found -> %s", "ja": "不具合 %d 件 -> %s", "fr": "%d bug(s) trouvé(s) -> %s"},

	// === Review ===
	"review_running":    {"en": "Running code review (%d/%d)...", "ja": "コードレビュー実行中 (%d/%d)...", "fr": "Revue de code en cours (%d/%d)..."},
	"review_passed":     {"en": "Code review passed", "ja": "コードレビュー通過", "fr": "Revue de code validée"},
	"review_comments":   {"en": "Review cycle %d: comments found, fixing...", "ja": "レビュー %d 回目: 指摘あり、修正中...", "fr": "Cycle de revue %d : commentaires trouvés, correction..."},
	"review_error":      {"en": "Review skipped (command error: %v)", "ja": "レビュースキップ（コマンドエラー: %v）", "fr": "Revue ignorée (erreur de commande : %v)"},
	"review_limit":      {"en": "Review cycle limit reached — remaining insights recorded in journal", "ja": "レビュー回数上限 — 残りの指摘を journal に記録", "fr": "Limite de cycles de revue atteinte — remarques restantes consignées dans le journal"},
	"reviewfix_running": {"en": "Running reviewfix (model: %s)...", "ja": "レビュー修正実行中 (model: %s)...", "fr": "Correction de revue en cours (modèle : %s)..."},
	"reviewfix_error":   {"en": "Reviewfix failed: %v", "ja": "レビュー修正失敗: %v", "fr": "Échec de la correction de revue : %v"},

	// === Signal ===
	"signal_received": {"en": "Signal received: %v — cleaning up...", "ja": "シグナル受信: %v — クリーンアップ中...", "fr": "Signal reçu : %v — nettoyage en cours..."},

	// === Lumina file header ===
	"lumina_header":    {"en": "# Lumina — Learned Passive Skills\n\nAutomatically extracted from past Expedition Journals.\nUpdated each time an Expeditioner rests at a Flag.\n", "ja": "# Lumina — 学習済みパッシブスキル\n\n過去の Expedition Journal から自動抽出された知恵。\nExpedition Flag で休息するたびに更新される。\n", "fr": "# Lumina — Compétences passives acquises\n\nExtraites automatiquement des Journaux d'Expédition passés.\nMises à jour chaque fois qu'un Expéditionnaire se repose à un Drapeau d'Expédition.\n"},
	"lumina_defensive": {"en": "## Defensive (lessons from failures)", "ja": "## 防御系（失敗から学んだ教訓）", "fr": "## Défensif (leçons tirées des échecs)"},
	"lumina_offensive": {"en": "## Offensive (proven patterns)", "ja": "## 攻撃系（成功パターン）", "fr": "## Offensif (stratégies éprouvées)"},
	"lumina_none":      {"en": "(No Lumina learned yet)", "ja": "（まだ学習済みの Lumina はありません）", "fr": "(Aucune Lumina apprise pour le moment)"},
}

// Msg returns a localized message by key.
// Falls back to English if the key or language is missing.
func Msg(key string) string {
	if m, ok := messages[key]; ok {
		if s, ok := m[Lang]; ok {
			return s
		}
		if s, ok := m["en"]; ok {
			return s
		}
	}
	return fmt.Sprintf("[missing: %s]", key)
}
