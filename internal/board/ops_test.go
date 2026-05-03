package board

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
)

func setupBoard(t *testing.T) *Board {
	t.Helper()
	root := t.TempDir()
	b, err := Init(root)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return b
}

func TestNewCardCreatesEverything(t *testing.T) {
	b := setupBoard(t)
	c, dir, err := b.NewCard("Ship the feature", NewCardOpts{})
	if err != nil {
		t.Fatalf("NewCard: %v", err)
	}
	if c.ID != 1 {
		t.Errorf("first card id = %d, want 1", c.ID)
	}
	if c.UUID == "" {
		t.Error("UUID empty")
	}
	if c.Status != card.StatusBacklog {
		t.Errorf("Status = %s, want backlog", c.Status)
	}
	if c.Priority != card.PriorityP2 {
		t.Errorf("default Priority = %s, want p2", c.Priority)
	}
	if dir != "0001-ship-the-feature" {
		t.Errorf("dir = %q", dir)
	}

	wantPath := filepath.Join(b.CardsDir(), "0001-ship-the-feature", "INDEX.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Errorf("card file missing: %v", err)
	}

	idxPath := filepath.Join(b.Dir, "index.json")
	if _, err := os.Stat(idxPath); err != nil {
		t.Errorf("index.json missing after first new: %v", err)
	}
}

func TestNewCardAllocatesMonotonicIDs(t *testing.T) {
	b := setupBoard(t)
	for i := 1; i <= 5; i++ {
		c, _, err := b.NewCard("card", NewCardOpts{})
		if err != nil {
			t.Fatalf("NewCard %d: %v", i, err)
		}
		if c.ID != i {
			t.Errorf("card[%d].ID = %d, want %d", i, c.ID, i)
		}
	}
}

func TestNewCardConcurrentNoDoubleAllocation(t *testing.T) {
	b := setupBoard(t)
	const n = 8
	var wg sync.WaitGroup
	ids := make(chan int, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, _, err := b.NewCard("card", NewCardOpts{})
			if err != nil {
				t.Errorf("NewCard: %v", err)
				return
			}
			ids <- c.ID
		}()
	}
	wg.Wait()
	close(ids)

	seen := map[int]bool{}
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate id %d allocated under concurrent NewCard", id)
		}
		seen[id] = true
	}
	if len(seen) != n {
		t.Errorf("got %d unique ids, want %d", len(seen), n)
	}
}

func TestActivatePark(t *testing.T) {
	b := setupBoard(t)
	c, _, _ := b.NewCard("c", NewCardOpts{})
	if _, err := b.Activate(c.ID, false); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	got, _, _ := b.LoadCard(c.ID)
	if got.Status != card.StatusActive {
		t.Errorf("status after Activate = %s", got.Status)
	}

	if _, err := b.Activate(c.ID, false); err == nil {
		t.Error("re-Activate should error")
	}

	if _, err := b.Park(c.ID, false); err != nil {
		t.Errorf("Park: %v", err)
	}
	got, _, _ = b.LoadCard(c.ID)
	if got.Status != card.StatusBacklog {
		t.Errorf("status after Park = %s", got.Status)
	}
}

func TestDoneRequiresActive(t *testing.T) {
	b := setupBoard(t)
	c, _, _ := b.NewCard("c", NewCardOpts{})
	if _, err := b.Done(c.ID, false); err == nil {
		t.Error("Done on backlog should error")
	}
	_, _ = b.Activate(c.ID, false)
	if _, err := b.Done(c.ID, false); err != nil {
		t.Errorf("Done on active: %v", err)
	}
}

func TestKillUnrestricted(t *testing.T) {
	b := setupBoard(t)
	c, _, _ := b.NewCard("c", NewCardOpts{})
	if _, err := b.Kill(c.ID, false); err != nil {
		t.Errorf("Kill from backlog: %v", err)
	}
	got, _, _ := b.LoadCard(c.ID)
	if got.Status != card.StatusArchived {
		t.Errorf("status after Kill = %s", got.Status)
	}
}

func TestRevive(t *testing.T) {
	b := setupBoard(t)
	c, _, _ := b.NewCard("c", NewCardOpts{})
	_, _ = b.Kill(c.ID, false)
	if _, err := b.Revive(c.ID, false); err != nil {
		t.Errorf("Revive: %v", err)
	}
	got, _, _ := b.LoadCard(c.ID)
	if got.Status != card.StatusBacklog {
		t.Errorf("status after Revive = %s", got.Status)
	}

	// Reviving a backlog card should error.
	if _, err := b.Revive(c.ID, false); err == nil {
		t.Error("Revive on backlog should error")
	}
}

func TestActivateEnforcesWIPLimit(t *testing.T) {
	b := setupBoard(t)
	cfgPath := filepath.Join(b.Dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("wip_limit: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var ids []int
	for i := 0; i < 3; i++ {
		c, _, _ := b.NewCard("c", NewCardOpts{})
		ids = append(ids, c.ID)
	}
	if _, err := b.Activate(ids[0], false); err != nil {
		t.Fatalf("first Activate: %v", err)
	}
	if _, err := b.Activate(ids[1], false); err != nil {
		t.Fatalf("second Activate: %v", err)
	}
	_, err := b.Activate(ids[2], false)
	if !errors.Is(err, ErrWIPLimit) {
		t.Errorf("third Activate err = %v, want ErrWIPLimit", err)
	}
	if _, err := b.Activate(ids[2], true); err != nil {
		t.Errorf("Activate force: %v", err)
	}
}

func TestEpicAdd(t *testing.T) {
	b := setupBoard(t)
	epic, _, err := b.NewCard("Launch v2", NewCardOpts{Type: card.TypeEpic})
	if err != nil {
		t.Fatalf("NewCard epic: %v", err)
	}
	c, _, _ := b.NewCard("Ship feature", NewCardOpts{})
	if err := b.EpicAdd(epic.ID, c.ID, false); err != nil {
		t.Fatalf("EpicAdd: %v", err)
	}
	got, _, _ := b.LoadCard(c.ID)
	if got.Epic == nil || *got.Epic != epic.ID {
		t.Errorf("Epic = %v, want %d", got.Epic, epic.ID)
	}

	// Adding to a non-epic card should error without --force.
	if err := b.EpicAdd(c.ID, c.ID, false); err == nil {
		t.Error("EpicAdd to non-epic should error")
	}
}

func TestList(t *testing.T) {
	b := setupBoard(t)
	c1, _, _ := b.NewCard("a", NewCardOpts{Project: "alpha"})
	c2, _, _ := b.NewCard("b", NewCardOpts{Project: "beta"})
	_, _, _ = b.NewCard("c", NewCardOpts{Project: "alpha"})

	all, err := b.List(ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("List() len = %d, want 3", len(all))
	}

	alpha, _ := b.List(ListOpts{Project: "alpha"})
	if len(alpha) != 2 {
		t.Errorf("project=alpha len = %d, want 2", len(alpha))
	}

	_, _ = b.Activate(c1.ID, false)
	active, _ := b.List(ListOpts{Status: card.StatusActive})
	if len(active) != 1 || active[0].ID != c1.ID {
		t.Errorf("active list = %v", active)
	}
	_ = c2
}

func TestBoardView(t *testing.T) {
	b := setupBoard(t)
	c1, _, _ := b.NewCard("a", NewCardOpts{Priority: card.PriorityP1})
	_, _, _ = b.NewCard("b", NewCardOpts{Priority: card.PriorityP3})
	c3, _, _ := b.NewCard("c", NewCardOpts{Priority: card.PriorityP0})
	_, _, _ = b.NewCard("epic", NewCardOpts{Type: card.TypeEpic})
	_, _ = b.Activate(c3.ID, false)

	v, err := b.Board()
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Active) != 1 || v.Active[0].ID != c3.ID {
		t.Errorf("Active = %v", v.Active)
	}
	if len(v.Backlog) != 2 {
		t.Errorf("Backlog len = %d, want 2", len(v.Backlog))
	}
	// Backlog should be sorted by priority then id.
	if v.Backlog[0].ID != c1.ID {
		t.Errorf("Backlog[0].ID = %d, want %d (p1 should come before p3)", v.Backlog[0].ID, c1.ID)
	}
	if len(v.Epics) != 1 {
		t.Errorf("Epics len = %d, want 1", len(v.Epics))
	}
}

func TestEpicShow(t *testing.T) {
	b := setupBoard(t)
	epic, _, _ := b.NewCard("epic", NewCardOpts{Type: card.TypeEpic})
	c1, _, _ := b.NewCard("c1", NewCardOpts{})
	c2, _, _ := b.NewCard("c2", NewCardOpts{})
	_ = b.EpicAdd(epic.ID, c1.ID, false)
	_ = b.EpicAdd(epic.ID, c2.ID, false)
	_, _ = b.Activate(c1.ID, false)
	_, _ = b.Done(c1.ID, false)

	p, err := b.EpicShow(epic.ID)
	if err != nil {
		t.Fatalf("EpicShow: %v", err)
	}
	if p.Done != 1 {
		t.Errorf("Done = %d, want 1", p.Done)
	}
	if p.Backlog != 1 {
		t.Errorf("Backlog = %d, want 1", p.Backlog)
	}
	if p.Total() != 2 {
		t.Errorf("Total = %d, want 2", p.Total())
	}
}

func TestReindexRebuildsFromDisk(t *testing.T) {
	b := setupBoard(t)
	c1, _, _ := b.NewCard("a", NewCardOpts{})
	_, _, _ = b.NewCard("b", NewCardOpts{})

	// Wipe the index — simulating a hand-deletion or merge artifact.
	if err := os.Remove(filepath.Join(b.Dir, "index.json")); err != nil {
		t.Fatal(err)
	}
	idx, err := b.Reindex()
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if len(idx.Cards) != 2 {
		t.Errorf("reindexed len = %d, want 2", len(idx.Cards))
	}
	if idx.NextID != 3 {
		t.Errorf("NextID = %d, want 3", idx.NextID)
	}
	_ = c1
}

func TestReindexPreservesHighWaterMark(t *testing.T) {
	b := setupBoard(t)
	c1, _, _ := b.NewCard("a", NewCardOpts{})
	_, _, _ = b.NewCard("b", NewCardOpts{})

	// Hand-delete card 2's directory but leave the index alone.
	dir2, _ := b.FindCardDir(2)
	if err := os.RemoveAll(filepath.Join(b.CardsDir(), dir2)); err != nil {
		t.Fatal(err)
	}

	idx, err := b.Reindex()
	if err != nil {
		t.Fatal(err)
	}
	// max(id) is now 1; but next_id from the prior index was 3 and
	// must be preserved per the high-water rule.
	if idx.NextID != 3 {
		t.Errorf("NextID = %d, want 3 (must preserve high-water mark)", idx.NextID)
	}
	_ = c1
	_ = index.SchemaVersion
}
