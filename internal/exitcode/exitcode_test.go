package exitcode

import "testing"

func TestCodeAndFormat(t *testing.T) {
	err := WithHint(New(Usage, "missing required argument: state"), "run `mcp2cli tools weather get-alerts`")
	if got := Code(err); got != 2 {
		t.Fatalf("Code(err) = %d, want 2", got)
	}
	formatted := Format(err)
	if formatted != "error: missing required argument: state\nhint: run `mcp2cli tools weather get-alerts`" {
		t.Fatalf("Format(err) = %q", formatted)
	}
}
