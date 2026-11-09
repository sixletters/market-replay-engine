// Package cli implements the market-replay command-line interface.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"sixletters/back-to-the-past/internal/orderbook"
	"sixletters/back-to-the-past/internal/replayio"
)

const (
	exitOK        = 0
	exitError     = 1
	exitViolation = 2 // gap or invariant violation
)

// maxDepth requests the full book from View (the implementation clamps).
const maxDepth = 1 << 30

const usage = `market-replay — drive the L2 order book over a JSONL event stream

usage:
  market-replay replay   --file events.jsonl [--depth 10]
  market-replay validate --file events.jsonl
  market-replay bench    --file events.jsonl [--iters 10000]
`

// Run dispatches a subcommand and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitError
	}
	switch args[0] {
	case "replay":
		return runReplay(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "bench":
		return runBench(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return exitOK
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usage)
		return exitError
	}
}

// result summarizes one applied event stream.
type result struct {
	book    orderbook.OrderBook
	applied int
	skipped int // duplicate/stale deltas
}

// runStream builds a book from an event stream. The stream must begin with a
// snapshot; when check is true, book invariants are verified after every event.
func runStream(r io.Reader, check bool) (*result, error) {
	dec := replayio.NewDecoder(r)
	res := &result{}

	for dec.Next() {
		ev := dec.Event()
		switch ev.Kind {
		case replayio.KindSnapshot:
			if res.book != nil {
				return nil, fmt.Errorf("line %d: unexpected snapshot mid-stream", ev.Line)
			}
			res.book = orderbook.NewSimpleOrderBook(ev.Snapshot.Market)
			if err := res.book.ApplySnapshot(*ev.Snapshot); err != nil {
				return nil, fmt.Errorf("line %d: %w", ev.Line, err)
			}
			res.applied++
		case replayio.KindUpdate:
			if res.book == nil {
				return nil, fmt.Errorf("line %d: stream must begin with a snapshot", ev.Line)
			}
			u := ev.Update
			if u.Cursor.Sequence <= res.book.Cursor().Sequence {
				res.skipped++
				continue
			}
			if err := res.book.ApplyUpdate(*u); err != nil {
				return nil, fmt.Errorf("line %d (sequence %d): %w", ev.Line, u.Cursor.Sequence, err)
			}
			res.applied++
		}

		if check {
			if err := checkInvariants(res.book.View(maxDepth)); err != nil {
				return nil, fmt.Errorf("line %d (sequence %d): %w", ev.Line, res.book.Cursor().Sequence, err)
			}
		}
	}
	if err := dec.Err(); err != nil {
		return nil, err
	}
	if res.book == nil {
		return nil, errors.New("empty stream: no snapshot")
	}
	return res, nil
}

// applyAll replays a decoded stream from scratch; used by bench.
func applyAll(events []replayio.Event) error {
	res := &result{}
	for _, ev := range events {
		switch ev.Kind {
		case replayio.KindSnapshot:
			res.book = orderbook.NewSimpleOrderBook(ev.Snapshot.Market)
			if err := res.book.ApplySnapshot(*ev.Snapshot); err != nil {
				return err
			}
		case replayio.KindUpdate:
			if res.book == nil {
				return errors.New("stream must begin with a snapshot")
			}
			if err := res.book.ApplyUpdate(*ev.Update); err != nil {
				return err
			}
		}
	}
	if res.book == nil {
		return errors.New("empty stream")
	}
	return nil
}

func openFile(path string, stderr io.Writer) (*os.File, int) {
	if path == "" {
		fmt.Fprintln(stderr, "--file is required")
		return nil, exitError
	}
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return nil, exitError
	}
	return f, exitOK
}

func reportErr(err error, stderr io.Writer) int {
	fmt.Fprintln(stderr, err)
	if errors.Is(err, orderbook.ErrGap) || errors.Is(err, errInvariant) {
		return exitViolation
	}
	return exitError
}

func runReplay(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "path to JSONL event file")
	depth := fs.Int("depth", 10, "book depth to print per side")
	if err := fs.Parse(args); err != nil {
		return exitError
	}
	f, code := openFile(*file, stderr)
	if f == nil {
		return code
	}
	defer f.Close()

	res, err := runStream(f, false)
	if err != nil {
		return reportErr(err, stderr)
	}
	printResult(stdout, res, *depth)
	return exitOK
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "path to JSONL event file")
	if err := fs.Parse(args); err != nil {
		return exitError
	}
	f, code := openFile(*file, stderr)
	if f == nil {
		return code
	}
	defer f.Close()

	res, err := runStream(f, true)
	if err != nil {
		return reportErr(err, stderr)
	}
	fmt.Fprintf(stdout, "ok: %d events applied (%d duplicate/stale skipped), invariants held\n", res.applied, res.skipped)
	return exitOK
}

func runBench(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "", "path to JSONL event file")
	iters := fs.Int("iters", 10000, "number of full-stream replay iterations")
	if err := fs.Parse(args); err != nil {
		return exitError
	}
	f, code := openFile(*file, stderr)
	if f == nil {
		return code
	}
	defer f.Close()

	events, err := replayio.ReadAll(f)
	if err != nil {
		return reportErr(err, stderr)
	}
	updates := 0
	for _, ev := range events {
		if ev.Kind == replayio.KindUpdate {
			updates++
		}
	}

	start := time.Now()
	for i := 0; i < *iters; i++ {
		if err := applyAll(events); err != nil {
			return reportErr(err, stderr)
		}
	}
	elapsed := time.Since(start)

	total := float64(updates * *iters)
	fmt.Fprintf(stdout, "iterations: %d, updates/iteration: %d\n", *iters, updates)
	fmt.Fprintf(stdout, "elapsed: %s, updates/sec: %.0f, avg update: %.0f ns\n",
		elapsed.Round(time.Microsecond), total/elapsed.Seconds(), float64(elapsed.Nanoseconds())/total)
	return exitOK
}

func printResult(w io.Writer, res *result, depth int) {
	book := res.book
	view := book.View(maxDepth)

	fmt.Fprintf(w, "market: %s/%s\n", view.Market.Venue, view.Market.Instrument)
	fmt.Fprintf(w, "events applied: %d (%d duplicate/stale skipped)\n", res.applied, res.skipped)
	fmt.Fprintf(w, "final sequence: %d\n", book.Cursor().Sequence)

	if bid, ok := book.BestBid(); ok {
		fmt.Fprintf(w, "best bid: %d x %d\n", bid.Price, bid.Quantity)
	} else {
		fmt.Fprintln(w, "best bid: —")
	}
	if ask, ok := book.BestAsk(); ok {
		fmt.Fprintf(w, "best ask: %d x %d\n", ask.Price, ask.Quantity)
	} else {
		fmt.Fprintln(w, "best ask: —")
	}
	if spread, ok := book.Spread(); ok {
		fmt.Fprintf(w, "spread: %d\n", spread)
	}
	if mid, ok := book.MidPrice(); ok {
		fmt.Fprintf(w, "mid: %d\n", mid)
	}

	fmt.Fprintf(w, "bids (depth %d):\n", depth)
	for i, lvl := range view.Bids {
		if i >= depth {
			break
		}
		fmt.Fprintf(w, "  %d  x %d\n", lvl.Price, lvl.Quantity)
	}
	fmt.Fprintf(w, "asks (depth %d):\n", depth)
	for i, lvl := range view.Asks {
		if i >= depth {
			break
		}
		fmt.Fprintf(w, "  %d  x %d\n", lvl.Price, lvl.Quantity)
	}

	fmt.Fprintf(w, "hash: %016x\n", hashView(view))
}
