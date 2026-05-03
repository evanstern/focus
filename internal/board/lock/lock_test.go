package lock

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	l, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("Release: %v", err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("double Release should be no-op: %v", err)
	}
}

func TestWithSerializesOperations(t *testing.T) {
	// Two goroutines bump a counter inside With(); without
	// serialization the increments could overlap. The With contract is
	// "block until lock is held", so the test asserts that the inner
	// fn is never running on both goroutines simultaneously.
	dir := t.TempDir()
	var inFlight, maxInFlight atomic.Int32

	work := func() error {
		n := inFlight.Add(1)
		for {
			cur := maxInFlight.Load()
			if n <= cur || maxInFlight.CompareAndSwap(cur, n) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		inFlight.Add(-1)
		return nil
	}

	done := make(chan error, 4)
	for i := 0; i < 4; i++ {
		go func() {
			done <- With(dir, work)
		}()
	}
	for i := 0; i < 4; i++ {
		if err := <-done; err != nil {
			t.Errorf("With err: %v", err)
		}
	}
	if got := maxInFlight.Load(); got != 1 {
		t.Errorf("maxInFlight = %d, want 1 (lock should serialize)", got)
	}
}

func TestWithReleasesOnError(t *testing.T) {
	dir := t.TempDir()
	sentinel := errors.New("inner err")
	if err := With(dir, func() error { return sentinel }); !errors.Is(err, sentinel) {
		t.Errorf("With didn't return inner err: %v", err)
	}
	// Second With must succeed — if the lock weren't released after
	// the first call's error, this would deadlock.
	done := make(chan error, 1)
	go func() {
		done <- With(dir, func() error { return nil })
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("second With: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second With deadlocked: lock not released after error")
	}
}

func TestWithReleasesOnPanic(t *testing.T) {
	dir := t.TempDir()
	defer func() {
		_ = recover()
		// After panic recovery, lock should still be released so a
		// subsequent acquire doesn't hang.
		done := make(chan error, 1)
		go func() {
			done <- With(dir, func() error { return nil })
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("post-panic With: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("post-panic With deadlocked")
		}
	}()
	_ = With(dir, func() error {
		panic("boom")
	})
}
