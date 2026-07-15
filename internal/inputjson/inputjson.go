// Package inputjson reads and parses the Claude Code session JSON from stdin.
//
// It extracts exactly the fields the go-renderer-core segments consume. Any
// missing field becomes an empty string / zero value; malformed JSON degrades
// to an all-empty Input. Parsing never panics and never writes to stdout.
package inputjson

import (
	"encoding/json"
	"io"
	"strconv"
)

// MaxInputBytes is the hard ceiling on stdin consumption (4 MiB). Input beyond
// this is ignored; parsing proceeds on the truncated data.
const MaxInputBytes = 4 << 20

// Input holds the extracted session fields. Percentage and reset fields are kept
// as their raw string representation (as the bash `jq ... | tostring` path does)
// so byte-for-byte data-file compatibility is preserved downstream.
type Input struct {
	Cwd    string // workspace.current_dir, fallback cwd
	Model  string // model.display_name
	CtxPct string // context_window.used_percentage (raw)
	TokIn  int64  // context_window.total_input_tokens
	TokOut int64  // context_window.total_output_tokens
	TokCR  int64  // context_window.current_usage.cache_read_input_tokens
	TokCW  int64  // context_window.current_usage.cache_creation_input_tokens
	FhPct  string // rate_limits.five_hour.used_percentage (raw)
	FhRst  string // rate_limits.five_hour.resets_at
	WdPct  string // rate_limits.seven_day.used_percentage (raw)
	WdRst  string // rate_limits.seven_day.resets_at
	Effort string // effort.level
}

// flexStr captures a scalar as its raw text whether the producer sent a JSON
// string or a JSON number (live Claude Code sends resets_at as a numeric
// epoch; sample payloads use ISO strings). This mirrors jq's `tostring`: the
// bash implementation is type-agnostic here, so the port must be too. Its
// UnmarshalJSON never returns an error — a type variation in one field must
// not degrade any other field (spec: stdin session JSON requirement).
type flexStr string

func (f *flexStr) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if json.Unmarshal(b, &s) == nil {
			*f = flexStr(s)
		}
		return nil
	}
	// Objects/arrays have no scalar meaning for these fields: leave empty.
	if b[0] == '{' || b[0] == '[' {
		return nil
	}
	*f = flexStr(b) // number/bool token, raw text
	return nil
}

// raw mirrors the input document's shape for the consumed fields. flexStr
// preserves the original scalar text (e.g. "74.5", "1784028600") whether it
// arrived as a JSON number or string, matching jq tostring.
type raw struct {
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Cwd   string `json:"cwd"`
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage    flexStr `json:"used_percentage"`
		TotalInputTokens  flexStr `json:"total_input_tokens"`
		TotalOutputTokens flexStr `json:"total_output_tokens"`
		CurrentUsage      struct {
			CacheRead     flexStr `json:"cache_read_input_tokens"`
			CacheCreation flexStr `json:"cache_creation_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	RateLimits struct {
		FiveHour struct {
			UsedPercentage flexStr `json:"used_percentage"`
			ResetsAt       flexStr `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage flexStr `json:"used_percentage"`
			ResetsAt       flexStr `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
	Effort struct {
		Level string `json:"level"`
	} `json:"effort"`
}

// Parse reads at most MaxInputBytes from r and returns the extracted Input.
func Parse(r io.Reader) Input {
	data := readBounded(r)
	var rw raw
	if err := json.Unmarshal(data, &rw); err != nil {
		// Malformed / truncated JSON: degrade to all-empty.
		return Input{}
	}
	cwd := rw.Workspace.CurrentDir
	if cwd == "" {
		cwd = rw.Cwd
	}
	return Input{
		Cwd:    cwd,
		Model:  rw.Model.DisplayName,
		CtxPct: string(rw.ContextWindow.UsedPercentage),
		TokIn:  numToInt(rw.ContextWindow.TotalInputTokens),
		TokOut: numToInt(rw.ContextWindow.TotalOutputTokens),
		TokCR:  numToInt(rw.ContextWindow.CurrentUsage.CacheRead),
		TokCW:  numToInt(rw.ContextWindow.CurrentUsage.CacheCreation),
		FhPct:  string(rw.RateLimits.FiveHour.UsedPercentage),
		FhRst:  string(rw.RateLimits.FiveHour.ResetsAt),
		WdPct:  string(rw.RateLimits.SevenDay.UsedPercentage),
		WdRst:  string(rw.RateLimits.SevenDay.ResetsAt),
		Effort: rw.Effort.Level,
	}
}

// readBounded reads at most MaxInputBytes from r, discarding the rest.
func readBounded(r io.Reader) []byte {
	data, _ := io.ReadAll(io.LimitReader(r, MaxInputBytes))
	return data
}

// numToInt converts a scalar token to int64, treating empty/invalid as 0.
// A float-valued token (e.g. "1234.0") is truncated toward zero, matching the
// bash `// 0` integer path.
func numToInt(n flexStr) int64 {
	if n == "" {
		return 0
	}
	if i, err := strconv.ParseInt(string(n), 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(string(n), 64); err == nil {
		return int64(f)
	}
	return 0
}
