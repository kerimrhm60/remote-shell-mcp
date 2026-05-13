package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitKVOp(t *testing.T) {
	// The four operators in order of length so the parser can't get confused
	// by a `=` inside a `:=` or `@=`.
	cases := []struct {
		in        string
		key, op   string
		val       string
		ok        bool
	}{
		{"command=ls", "command", "=", "ls", true},
		{"timeout_ms:=30000", "timeout_ms", ":=", "30000", true},
		{"command@=/tmp/script.sh", "command", "@=", "/tmp/script.sh", true},
		{"spec:@=/tmp/spec.json", "spec", ":@=", "/tmp/spec.json", true},
		// Underscores and trailing digits in the key.
		{"session_id_2=abc", "session_id_2", "=", "abc", true},
		// Empty value is legal — empty string arg.
		{"x=", "x", "=", "", true},
		// Operator with no value still parses (the JSON path will error
		// downstream, which is the right place for that diagnostic).
		{"x:=", "x", ":=", "", true},
		// No operator at all → not a kv-pair.
		{"justaword", "", "", "", false},
		// Starts with operator (no key) → not a kv-pair.
		{"=value", "", "", "", false},
		// Key can't start with a digit (matches Go identifier rules).
		{"1session=abc", "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			k, op, v, ok := splitKVOp(c.in)
			if ok != c.ok || k != c.key || op != c.op || v != c.val {
				t.Errorf("splitKVOp(%q) = (%q, %q, %q, %v); want (%q, %q, %q, %v)",
					c.in, k, op, v, ok, c.key, c.op, c.val, c.ok)
			}
		})
	}
}

func TestParseKVArgsBasic(t *testing.T) {
	got, err := parseKVArgs([]string{"command=ls", "session_id=foo"})
	if err != nil {
		t.Fatalf("parseKVArgs: %v", err)
	}
	want := map[string]any{"command": "ls", "session_id": "foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseKVArgsRawJSON(t *testing.T) {
	got, err := parseKVArgs([]string{
		`timeout_ms:=30000`,
		`forward_agent:=true`,
		`agent:=null`,
		`tags:=["a","b"]`,
		`auth:={"agent":true,"user":"root"}`,
	})
	if err != nil {
		t.Fatalf("parseKVArgs: %v", err)
	}
	// `:=` parses via json.Unmarshal, which decodes all JSON numbers as
	// float64. That's fine — the wire format preserves the value, and any
	// downstream schema validator treats 30000 and 30000.0 as the same
	// JSON number. coerceLooseJSON is a fallback for inputs json.Unmarshal
	// rejects (e.g. unquoted booleans), not the primary path for digits.
	if got["timeout_ms"] != float64(30000) {
		t.Fatalf("timeout_ms = %T(%v), want float64(30000)", got["timeout_ms"], got["timeout_ms"])
	}
	if got["forward_agent"] != true {
		t.Fatalf("forward_agent = %v", got["forward_agent"])
	}
	if got["agent"] != nil {
		t.Fatalf("agent = %v, want nil", got["agent"])
	}
	tags, ok := got["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("tags = %#v", got["tags"])
	}
	auth, ok := got["auth"].(map[string]any)
	if !ok || auth["agent"] != true || auth["user"] != "root" {
		t.Fatalf("auth = %#v", got["auth"])
	}
}

func TestParseKVArgsFromFile(t *testing.T) {
	dir := t.TempDir()
	strPath := filepath.Join(dir, "cmd.txt")
	jsonPath := filepath.Join(dir, "spec.json")
	if err := os.WriteFile(strPath, []byte("uname -a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte(`{"addresses":["host:22"],"auth":{"agent":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := parseKVArgs([]string{
		"command@=" + strPath,
		"spec:@=" + jsonPath,
	})
	if err != nil {
		t.Fatalf("parseKVArgs: %v", err)
	}
	if got["command"] != "uname -a" {
		t.Fatalf("command = %q, want %q", got["command"], "uname -a")
	}
	spec, ok := got["spec"].(map[string]any)
	if !ok {
		t.Fatalf("spec not an object: %#v", got["spec"])
	}
	addrs, _ := spec["addresses"].([]any)
	if len(addrs) != 1 || addrs[0] != "host:22" {
		t.Fatalf("spec.addresses = %#v", addrs)
	}
}

func TestParseKVArgsErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string // substring of expected error message
	}{
		{[]string{"justaword"}, "expected key=value"},
		{[]string{"x:=not[valid]json"}, "not valid JSON"},
		{[]string{"x@=/this/path/does/not/exist/we/promise"}, "no such file"},
		{[]string{"x:@=/this/path/does/not/exist/we/promise"}, "no such file"},
	}
	for _, c := range cases {
		t.Run(c.args[0], func(t *testing.T) {
			_, err := parseKVArgs(c.args)
			if err == nil {
				t.Fatalf("expected error for %q", c.args[0])
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %q, want substring %q", err.Error(), c.want)
			}
		})
	}
}

func TestCoerceLooseJSON(t *testing.T) {
	cases := []struct {
		in   string
		want any
		ok   bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"null", nil, true},
		{"42", int64(42), true},
		{"-7", int64(-7), true},
		{"3.14", float64(3.14), true},
		{"hello", nil, false},
		{"", nil, false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := coerceLooseJSON(c.in)
			if ok != c.ok {
				t.Fatalf("ok=%v, want %v", ok, c.ok)
			}
			if ok && !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %T(%v), want %T(%v)", got, got, c.want, c.want)
			}
		})
	}
}

func TestTranscodeOutputPassThrough(t *testing.T) {
	// outputAsIs should never touch the text.
	in := "anything goes here\nincluding bytes that aren't TOON\xff"
	if got := transcodeOutput(in, outputAsIs); got != in {
		t.Fatalf("outputAsIs mutated text: %q", got)
	}
}

func TestTranscodeJSONFromTOON(t *testing.T) {
	// A simple TOON document the daemon would emit: top-level scalar fields.
	toonIn := "uptime: 5s\ngoroutines: 6\n"
	got := transcodeOutput(toonIn, outputJSON)
	if !strings.Contains(got, `"uptime"`) || !strings.Contains(got, `"goroutines"`) {
		t.Fatalf("expected JSON keys, got:\n%s", got)
	}
}

func TestTranscodeTOONFromJSON(t *testing.T) {
	jsonIn := `{"uptime":"5s","goroutines":6}`
	got := transcodeOutput(jsonIn, outputTOON)
	if !strings.Contains(got, "uptime") || !strings.Contains(got, "goroutines") {
		t.Fatalf("expected TOON keys, got:\n%s", got)
	}
}

// (TOON's decoder is permissive enough to accept arbitrary text as a single
// scalar, so the "neither valid JSON nor valid TOON" pass-through path
// isn't really reachable in practice. The behavior we DO guarantee is that
// the function never panics on weird input — covered implicitly by the
// other transcode tests.)
