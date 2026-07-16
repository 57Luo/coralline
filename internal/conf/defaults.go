package conf

// defaults mirrors the built-in default values from statusline.sh's defaults
// block (before the user config is sourced).
var defaults = map[string]string{
	// Style / layout.
	"VL_STYLE":     "pill",
	"VL_LAYOUT":    "fixed",
	"VL_MAX_LINES": "3",
	"VL_SEGMENTS":  "dir git model ctx limit5h limit7d cost clock",
	"VL_SEGMENTS2": "",
	"VL_SEGMENTS3": "",

	// Bar / gauge.
	"VL_BAR_WIDTH": "5",
	"VL_BAR_FILL":  "▰", // ▰
	"VL_BAR_EMPTY": "▱", // ▱

	// Formatting / thresholds.
	"VL_PATH_DEPTH": "4",
	"VL_NAME_MAX":   "0",
	"VL_WARN_PCT":   "50",
	"VL_HOT_PCT":    "75",
	"VL_CTX_TOKENS": "1",
	"VL_ASCII":      "0",

	// Data-file feature flags.
	"VL_LIMIT_SYNC":  "0",
	"VL_USAGE_STATE": "0",

	// Powerline glyphs (Nerd Font).
	"VL_CAP_L": "", // U+E0B6 left rounded cap
	"VL_CAP_R": "", // U+E0B4 right rounded cap
	"VL_SEP":   "", // U+E0B0 segment separator

	// Burn segment.
	"CORALLINE_BURN_WINDOW": "600",
	"VL_BURN_GLYPH":         "↗", // ↗
	"VL_BG_BURN":            "",
	"BURN_TRIM":             "1500",

	// Default theme: claude-coral.
	"VL_BG_DIR":       "81,166,199",
	"VL_BG_PROJECT":   "",
	"VL_BG_GIT_OK":    "65",
	"VL_BG_STASH":     "",
	"VL_BG_GIT_DIRTY": "130",
	"VL_BG_MODEL":     "173",
	"VL_BG_CTX":       "238",
	"VL_BG_5H":        "237",
	"VL_BG_7D":        "236",
	"VL_BG_EFFORT":    "141",

	"VL_BG_COST":     "212,125,145",
	"VL_BG_CLOCK":    "70,80,110",
	"VL_BG_LINES":    "240",
	"VL_BG_STYLE":    "96",
	"VL_BG_DURATION": "60",
	"VL_BG_TOKENS":   "",
	"VL_BG_NODE":     "",
	"VL_BG_PYTHON":   "",

	"VL_COST_DECIMALS": "2",
	"VL_CLOCK":         "12h",
	"VL_CLOCK_SECONDS": "1",
	"VL_RUNTIME_PROBE": "0",
	"VL_NODE_GLYPH":    "",
	"VL_PY_GLYPH":      "",

	"VL_FG_TEXT": "231",
	"VL_FG_DIM":  "245",
	"VL_FG_OK":   "114",
	"VL_FG_WARN": "179",
	"VL_FG_HOT":  "167",
}
