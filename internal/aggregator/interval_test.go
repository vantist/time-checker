package aggregator_test

import (
	"testing"
	"time"

	"github.com/user/tt/internal/aggregator"
)

var idle15m = 15 * time.Minute

func TestUserIntervals(t *testing.T) {
	b := base()

	t.Run("normal adjacent turns produce interval", func(t *testing.T) {
		// turn1 response_at=T+60s, turn2 prompt_at=T+180s → interval [T+60, T+180]
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: ptr(b.Add(60 * time.Second))},
			{PromptAt: b.Add(180 * time.Second), ResponseAt: nil},
		}
		got := aggregator.UserIntervals(turns, time.Time{}, idle15m)
		if len(got) != 1 {
			t.Fatalf("want 1 interval, got %d", len(got))
		}
		want := 120 * time.Second
		if got[0].End.Sub(got[0].Start) != want {
			t.Errorf("interval length = %v, want %v", got[0].End.Sub(got[0].Start), want)
		}
	})

	t.Run("previous turn response_at nil skips interval", func(t *testing.T) {
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: nil},
			{PromptAt: b.Add(5 * time.Minute), ResponseAt: nil},
		}
		got := aggregator.UserIntervals(turns, time.Time{}, idle15m)
		if len(got) != 0 {
			t.Errorf("want 0 intervals, got %d", len(got))
		}
	})

	t.Run("non-zero session_start produces first interval", func(t *testing.T) {
		sessionStart := b.Add(-3 * time.Minute)
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: nil},
		}
		got := aggregator.UserIntervals(turns, sessionStart, idle15m)
		if len(got) != 1 {
			t.Fatalf("want 1 interval, got %d", len(got))
		}
		if got[0].Start != sessionStart || got[0].End != b {
			t.Errorf("first interval = [%v, %v], want [%v, %v]", got[0].Start, got[0].End, sessionStart, b)
		}
	})

	t.Run("zero session_start skips first interval", func(t *testing.T) {
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: nil},
		}
		got := aggregator.UserIntervals(turns, time.Time{}, idle15m)
		if len(got) != 0 {
			t.Errorf("want 0 intervals for zero sessionStart, got %d", len(got))
		}
	})

	t.Run("idle threshold discards long interval", func(t *testing.T) {
		// interval length = 20m >= 15m threshold → discard
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: ptr(b.Add(1 * time.Minute))},
			{PromptAt: b.Add(21 * time.Minute), ResponseAt: nil}, // interval = 20m
		}
		got := aggregator.UserIntervals(turns, time.Time{}, idle15m)
		if len(got) != 0 {
			t.Errorf("want 0 intervals (long interval discarded), got %d", len(got))
		}
	})

	t.Run("idle threshold keeps short interval", func(t *testing.T) {
		// interval length = 10m < 15m → keep
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: ptr(b.Add(1 * time.Minute))},
			{PromptAt: b.Add(11 * time.Minute), ResponseAt: nil}, // interval = 10m
		}
		got := aggregator.UserIntervals(turns, time.Time{}, idle15m)
		if len(got) != 1 {
			t.Errorf("want 1 interval (short interval kept), got %d", len(got))
		}
	})

	t.Run("session_start long interval discarded by threshold", func(t *testing.T) {
		// sessionStart 20m before first prompt → 20m >= 15m → discard
		sessionStart := b.Add(-20 * time.Minute)
		turns := []aggregator.Turn{
			{PromptAt: b, ResponseAt: nil},
		}
		got := aggregator.UserIntervals(turns, sessionStart, idle15m)
		if len(got) != 0 {
			t.Errorf("want 0 (sessionStart interval exceeds threshold), got %d", len(got))
		}
	})
}

func TestMergeAndSum(t *testing.T) {
	b := base()

	t.Run("empty slice returns 0", func(t *testing.T) {
		got := aggregator.MergeAndSum(nil)
		if got != 0 {
			t.Errorf("MergeAndSum(nil) = %v, want 0", got)
		}
	})

	t.Run("no overlap sums directly", func(t *testing.T) {
		// [10:00,10:05] + [10:10,10:15] = 10m
		intervals := []aggregator.Interval{
			{Start: b, End: b.Add(5 * time.Minute)},
			{Start: b.Add(10 * time.Minute), End: b.Add(15 * time.Minute)},
		}
		got := aggregator.MergeAndSum(intervals)
		want := 10 * time.Minute
		if got != want {
			t.Errorf("MergeAndSum no overlap = %v, want %v", got, want)
		}
	})

	t.Run("partial overlap merges", func(t *testing.T) {
		// [10:00,10:10] + [10:05,10:15] → [10:00,10:15] = 15m
		intervals := []aggregator.Interval{
			{Start: b, End: b.Add(10 * time.Minute)},
			{Start: b.Add(5 * time.Minute), End: b.Add(15 * time.Minute)},
		}
		got := aggregator.MergeAndSum(intervals)
		want := 15 * time.Minute
		if got != want {
			t.Errorf("MergeAndSum partial overlap = %v, want %v", got, want)
		}
	})

	t.Run("complete containment merges", func(t *testing.T) {
		// [10:00,10:20] contains [10:05,10:10] → [10:00,10:20] = 20m
		intervals := []aggregator.Interval{
			{Start: b, End: b.Add(20 * time.Minute)},
			{Start: b.Add(5 * time.Minute), End: b.Add(10 * time.Minute)},
		}
		got := aggregator.MergeAndSum(intervals)
		want := 20 * time.Minute
		if got != want {
			t.Errorf("MergeAndSum complete containment = %v, want %v", got, want)
		}
	})

	t.Run("three segments multiple overlap", func(t *testing.T) {
		// A=[10:00,10:08] B=[10:03,10:12] C=[10:10,10:20] → [10:00,10:20] = 20m
		intervals := []aggregator.Interval{
			{Start: b, End: b.Add(8 * time.Minute)},
			{Start: b.Add(3 * time.Minute), End: b.Add(12 * time.Minute)},
			{Start: b.Add(10 * time.Minute), End: b.Add(20 * time.Minute)},
		}
		got := aggregator.MergeAndSum(intervals)
		want := 20 * time.Minute
		if got != want {
			t.Errorf("MergeAndSum three overlapping = %v, want %v", got, want)
		}
	})
}
