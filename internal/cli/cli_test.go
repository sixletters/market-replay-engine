package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodStream = `{"type":"snapshot","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":100},"bids":[{"price":100,"quantity":5},{"price":99,"quantity":10}],"asks":[{"price":102,"quantity":4},{"price":103,"quantity":6}]}
{"type":"update","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":101},"changes":[{"side":0,"price":100,"quantityAfter":3}]}
{"type":"update","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":102},"changes":[{"side":1,"price":102,"quantityAfter":0},{"side":1,"price":101,"quantityAfter":8}]}
`

// writeStream writes a JSONL stream to a temp file and returns its path.
func writeStream(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}
	return path
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestReplayPrintsFinalBook(t *testing.T) {
	t.Parallel()

	code, out, errOut := run(t, "replay", "--file", writeStream(t, goodStream), "--depth", "2")
	if code != exitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	for _, want := range []string{
		"market: coinbase/BTC-USD",
		"final sequence: 102",
		"best bid: 100 x 3",
		"best ask: 101 x 8",
		"spread: 1",
		"mid: 100",
		"hash: ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\noutput:\n%s", want, out)
		}
	}
}

func TestReplayIsDeterministic(t *testing.T) {
	t.Parallel()

	path := writeStream(t, goodStream)
	code1, out1, _ := run(t, "replay", "--file", path)
	code2, out2, _ := run(t, "replay", "--file", path)
	if code1 != exitOK || code2 != exitOK {
		t.Fatalf("exits = %d, %d, want 0, 0", code1, code2)
	}
	if out1 != out2 {
		t.Fatalf("two replays produced different output:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", out1, out2)
	}
}

func TestReplayGapExitsViolation(t *testing.T) {
	t.Parallel()

	gapStream := `{"type":"snapshot","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":100},"bids":[{"price":100,"quantity":5}],"asks":[{"price":102,"quantity":4}]}
{"type":"update","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":102},"changes":[{"side":0,"price":100,"quantityAfter":3}]}
`
	code, _, errOut := run(t, "replay", "--file", writeStream(t, gapStream))
	if code != exitViolation {
		t.Fatalf("exit = %d, want %d", code, exitViolation)
	}
	if !strings.Contains(errOut, "sequence 102") {
		t.Errorf("stderr should name the offending sequence, got: %s", errOut)
	}
}

func TestReplayDuplicateDeltaIsSkipped(t *testing.T) {
	t.Parallel()

	dupStream := goodStream + `{"type":"update","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":102},"changes":[{"side":0,"price":100,"quantityAfter":9}]}
`
	code, out, errOut := run(t, "replay", "--file", writeStream(t, dupStream))
	if code != exitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "1 duplicate/stale skipped") {
		t.Errorf("output should report 1 skipped duplicate, got:\n%s", out)
	}
	if !strings.Contains(out, "best bid: 100 x 3") {
		t.Errorf("duplicate delta must not mutate the book, got:\n%s", out)
	}
}

func TestReplayRequiresSnapshotFirst(t *testing.T) {
	t.Parallel()

	deltaFirst := `{"type":"update","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":1},"changes":[]}
`
	code, _, errOut := run(t, "replay", "--file", writeStream(t, deltaFirst))
	if code != exitError {
		t.Fatalf("exit = %d, want %d", code, exitError)
	}
	if !strings.Contains(errOut, "must begin with a snapshot") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestValidateCleanStream(t *testing.T) {
	t.Parallel()

	code, out, errOut := run(t, "validate", "--file", writeStream(t, goodStream))
	if code != exitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "invariants held") {
		t.Errorf("output = %q", out)
	}
}

func TestValidateCrossedBookFails(t *testing.T) {
	t.Parallel()

	crossed := `{"type":"snapshot","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":100},"bids":[{"price":105,"quantity":5}],"asks":[{"price":102,"quantity":4}]}
`
	code, _, errOut := run(t, "validate", "--file", writeStream(t, crossed))
	if code != exitViolation {
		t.Fatalf("exit = %d, want %d", code, exitViolation)
	}
	if !strings.Contains(errOut, "crossed book") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestUnknownCommand(t *testing.T) {
	t.Parallel()

	code, _, errOut := run(t, "frobnicate")
	if code != exitError {
		t.Fatalf("exit = %d, want %d", code, exitError)
	}
	if !strings.Contains(errOut, `unknown command "frobnicate"`) {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestBenchRuns(t *testing.T) {
	t.Parallel()

	code, out, errOut := run(t, "bench", "--file", writeStream(t, goodStream), "--iters", "100")
	if code != exitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(out, "updates/sec") {
		t.Errorf("output = %q", out)
	}
}
