package marketdata

import (
	"time"

	"sixletters/back-to-the-past/internal/replayio"
)

// FromReplayIO converts a replayio event into a normalized market event.
func FromReplayIO(ev replayio.Event) Event {
	out := Event{ReceivedAt: time.Time{}}
	switch ev.Kind {
	case replayio.KindSnapshot:
		out.Kind = KindSnapshot
		out.Snapshot = ev.Snapshot
	case replayio.KindUpdate:
		out.Kind = KindUpdate
		out.Update = ev.Update
	}
	return out
}

// ToReplayIO converts a normalized market event into the replayio wire format.
func ToReplayIO(ev Event) replayio.Event {
	out := replayio.Event{Line: 0}
	switch ev.Kind {
	case KindSnapshot:
		out.Kind = replayio.KindSnapshot
		out.Snapshot = ev.Snapshot
	case KindUpdate:
		out.Kind = replayio.KindUpdate
		out.Update = ev.Update
	}
	return out
}
