package inputjson

import (
	"strings"
	"testing"
)

const fullJSON = `{
  "workspace": {"current_dir": "/home/u/proj"},
  "cwd": "/fallback/dir",
  "model": {"display_name": "Claude Fable 5"},
  "context_window": {
    "used_percentage": 74.5,
    "total_input_tokens": 1234,
    "total_output_tokens": 567,
    "current_usage": {
      "cache_read_input_tokens": 1234567,
      "cache_creation_input_tokens": 89
    }
  },
  "rate_limits": {
    "five_hour": {"used_percentage": 41, "resets_at": "2026-07-14T10:00:00Z"},
    "seven_day": {"used_percentage": 79, "resets_at": "2026-07-18T00:00:00Z"}
  },
  "effort": {"level": "high"}
}`

func TestParseFull(t *testing.T) {
	in := Parse(strings.NewReader(fullJSON))
	// workspace.current_dir wins over cwd.
	if in.Cwd != "/home/u/proj" {
		t.Errorf("Cwd = %q, want /home/u/proj", in.Cwd)
	}
	if in.Model != "Claude Fable 5" {
		t.Errorf("Model = %q", in.Model)
	}
	if in.CtxPct != "74.5" {
		t.Errorf("CtxPct = %q, want 74.5", in.CtxPct)
	}
	if in.TokIn != 1234 || in.TokOut != 567 || in.TokCR != 1234567 || in.TokCW != 89 {
		t.Errorf("tokens = %d/%d/%d/%d", in.TokIn, in.TokOut, in.TokCR, in.TokCW)
	}
	if in.FhPct != "41" || in.FhRst != "2026-07-14T10:00:00Z" {
		t.Errorf("five_hour = %q/%q", in.FhPct, in.FhRst)
	}
	if in.WdPct != "79" || in.WdRst != "2026-07-18T00:00:00Z" {
		t.Errorf("seven_day = %q/%q", in.WdPct, in.WdRst)
	}
	if in.Effort != "high" {
		t.Errorf("Effort = %q", in.Effort)
	}
}

func TestCwdFallback(t *testing.T) {
	in := Parse(strings.NewReader(`{"cwd":"/only/cwd"}`))
	if in.Cwd != "/only/cwd" {
		t.Errorf("Cwd = %q, want /only/cwd", in.Cwd)
	}
}

func TestMissingFields(t *testing.T) {
	in := Parse(strings.NewReader(`{}`))
	if in.Cwd != "" || in.Model != "" || in.CtxPct != "" || in.FhPct != "" ||
		in.FhRst != "" || in.WdPct != "" || in.WdRst != "" || in.Effort != "" {
		t.Errorf("missing fields should be empty strings, got %+v", in)
	}
	if in.TokIn != 0 || in.TokOut != 0 || in.TokCR != 0 || in.TokCW != 0 {
		t.Errorf("missing token fields should be zero, got %+v", in)
	}
}

// Live Claude Code payloads send resets_at as a numeric epoch (and may send
// percentages as strings). Spec: a type variation in one field must not
// degrade any other field — values are captured as raw text like jq tostring.
func TestNumericResetsAt(t *testing.T) {
	in := Parse(strings.NewReader(`{"model":{"display_name":"X"},` +
		`"context_window":{"used_percentage":19},` +
		`"rate_limits":{"five_hour":{"used_percentage":92,"resets_at":1784028600}}}`))
	if in.Model != "X" {
		t.Errorf("Model = %q, want X (numeric resets_at must not degrade other fields)", in.Model)
	}
	if in.CtxPct != "19" {
		t.Errorf("CtxPct = %q, want 19", in.CtxPct)
	}
	if in.FhPct != "92" || in.FhRst != "1784028600" {
		t.Errorf("five_hour = %q/%q, want 92/1784028600", in.FhPct, in.FhRst)
	}
}

func TestStringTypedNumbersAccepted(t *testing.T) {
	in := Parse(strings.NewReader(`{"context_window":{"used_percentage":"74.5","total_input_tokens":"1234"},` +
		`"rate_limits":{"seven_day":{"used_percentage":"79","resets_at":"2026-07-18T00:00:00Z"}}}`))
	if in.CtxPct != "74.5" || in.TokIn != 1234 {
		t.Errorf("ctx = %q tokIn = %d, want 74.5/1234", in.CtxPct, in.TokIn)
	}
	if in.WdPct != "79" || in.WdRst != "2026-07-18T00:00:00Z" {
		t.Errorf("seven_day = %q/%q", in.WdPct, in.WdRst)
	}
}

func TestMalformedJSONDegrades(t *testing.T) {
	in := Parse(strings.NewReader(`this is not json {{{`))
	// All fields empty/zero, no panic.
	if in.Cwd != "" || in.Model != "" || in.CtxPct != "" || in.TokIn != 0 {
		t.Errorf("malformed JSON should degrade to empty, got %+v", in)
	}
}

func TestOversizedInputTruncated(t *testing.T) {
	// Build > 4 MiB of input: a valid-looking prefix followed by megabytes of
	// padding inside a huge string value. Reading stops at 4 MiB, which breaks
	// the JSON, so it degrades to empty — but must not error or hang.
	var b strings.Builder
	b.WriteString(`{"model":{"display_name":"`)
	b.WriteString(strings.Repeat("x", 5*1024*1024))
	b.WriteString(`"}}`)
	in := Parse(strings.NewReader(b.String()))
	// Truncation breaks the JSON → degrades to empty; the point is no panic/hang
	// and bounded memory.
	if in.Model != "" {
		t.Errorf("expected truncated/degraded parse, got Model = %q", in.Model)
	}
}

func TestReadCapAtLimit(t *testing.T) {
	// Directly assert the read cap: feeding more than MaxInputBytes returns
	// exactly MaxInputBytes.
	huge := strings.Repeat("a", MaxInputBytes+1000)
	got := readBounded(strings.NewReader(huge))
	if len(got) != MaxInputBytes {
		t.Errorf("readBounded length = %d, want %d", len(got), MaxInputBytes)
	}
}
