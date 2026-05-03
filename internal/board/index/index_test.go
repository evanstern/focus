package index

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/evanstern/focus/internal/board/card"
)

func TestLoadOrEmptyMissing(t *testing.T) {
	dir := t.TempDir()
	idx, err := LoadOrEmpty(dir)
	if err != nil {
		t.Fatalf("LoadOrEmpty: %v", err)
	}
	if idx.NextID != 1 {
		t.Errorf("fresh index NextID = %d, want 1", idx.NextID)
	}
	if len(idx.Cards) != 0 {
		t.Errorf("fresh index has %d cards", len(idx.Cards))
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	c := sampleCard()
	idx := &Index{
		NextID: 2,
		Cards:  []Entry{EntryFromCard(c, "cards/0001-test")},
	}
	if err := Save(dir, idx); err != nil {
		t.Fatalf("Save: %v", err)
	}

	idx2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if idx2.NextID != 2 {
		t.Errorf("NextID round-trip: got %d", idx2.NextID)
	}
	if len(idx2.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(idx2.Cards))
	}
	if idx2.Cards[0].Title != "Test card" {
		t.Errorf("title round-trip: %q", idx2.Cards[0].Title)
	}
	if idx2.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version not stamped: %d", idx2.SchemaVersion)
	}
	if idx2.GeneratedAt.IsZero() {
		t.Error("GeneratedAt not stamped")
	}
}

func TestAllocateIDMonotonic(t *testing.T) {
	idx := &Index{NextID: 5}
	if id := idx.AllocateID(); id != 5 {
		t.Errorf("first allocate = %d, want 5", id)
	}
	if id := idx.AllocateID(); id != 6 {
		t.Errorf("second allocate = %d, want 6", id)
	}
	if idx.NextID != 7 {
		t.Errorf("NextID after two allocations = %d, want 7", idx.NextID)
	}
}

func TestUpsertReplaces(t *testing.T) {
	idx := &Index{Cards: []Entry{{ID: 1, Title: "old"}, {ID: 2, Title: "two"}}}
	idx.Upsert(Entry{ID: 1, Title: "new"})
	if len(idx.Cards) != 2 {
		t.Errorf("len = %d, want 2 (upsert should replace, not append)", len(idx.Cards))
	}
	if idx.Find(1).Title != "new" {
		t.Error("upsert didn't replace title")
	}
}

func TestUpsertAppends(t *testing.T) {
	idx := &Index{Cards: []Entry{{ID: 1, Title: "one"}}}
	idx.Upsert(Entry{ID: 2, Title: "two"})
	if len(idx.Cards) != 2 {
		t.Errorf("len = %d, want 2", len(idx.Cards))
	}
}

func TestRemovePreservesNextID(t *testing.T) {
	idx := &Index{NextID: 5, Cards: []Entry{{ID: 4}}}
	if !idx.Remove(4) {
		t.Error("Remove returned false on existing id")
	}
	if idx.NextID != 5 {
		t.Errorf("NextID = %d after remove, want 5 (ids never reused)", idx.NextID)
	}
}

func TestMaxID(t *testing.T) {
	idx := &Index{Cards: []Entry{{ID: 3}, {ID: 7}, {ID: 1}}}
	if got := idx.MaxID(); got != 7 {
		t.Errorf("MaxID = %d, want 7", got)
	}
	empty := &Index{}
	if got := empty.MaxID(); got != 0 {
		t.Errorf("MaxID(empty) = %d, want 0", got)
	}
}

func TestSaveSortsByID(t *testing.T) {
	dir := t.TempDir()
	idx := &Index{
		NextID: 4,
		Cards: []Entry{
			{ID: 3, Title: "three", UUID: "u3", Type: card.TypeCard, Status: card.StatusBacklog, Priority: card.PriorityP2, Project: "p", Created: "2026-01-01"},
			{ID: 1, Title: "one", UUID: "u1", Type: card.TypeCard, Status: card.StatusBacklog, Priority: card.PriorityP2, Project: "p", Created: "2026-01-01"},
			{ID: 2, Title: "two", UUID: "u2", Type: card.TypeCard, Status: card.StatusBacklog, Priority: card.PriorityP2, Project: "p", Created: "2026-01-01"},
		},
	}
	if err := Save(dir, idx); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for i, e := range loaded.Cards {
		if e.ID != i+1 {
			t.Errorf("card[%d].ID = %d, want %d (Save should sort by id)", i, e.ID, i+1)
		}
	}
}

func TestAtomicSaveSurvivesConcurrency(t *testing.T) {
	// Two goroutines each load → modify → save in a loop. Without flock
	// the writes race; the assertion here is only that the file always
	// parses, never that the final content is deterministic. Atomic
	// rename guarantees we never see a torn file.
	dir := t.TempDir()
	if err := Save(dir, &Index{NextID: 1}); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				idx, err := Load(dir)
				if err != nil {
					t.Errorf("worker %d Load: %v", w, err)
					return
				}
				idx.Upsert(Entry{
					ID:       w*100 + i + 1,
					UUID:     "u",
					Title:    "t",
					Type:     card.TypeCard,
					Status:   card.StatusBacklog,
					Priority: card.PriorityP2,
					Project:  "p",
					Created:  "2026-01-01",
				})
				if err := Save(dir, idx); err != nil {
					t.Errorf("worker %d Save: %v", w, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	// Final read should still parse cleanly. We don't assert on count;
	// concurrent read-modify-write WITHOUT the flock is the very thing
	// the lock package will guard against. This test only proves the
	// file is never corrupt.
	if _, err := Load(dir); err != nil {
		t.Errorf("final Load after concurrent saves: %v", err)
	}
}

func TestLoadRejectsUnknownSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	bad := []byte(`{"schema_version":99,"next_id":1,"cards":[]}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), bad, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load should reject schema_version mismatch")
	}
}

func TestEntryFromCardDropsBody(t *testing.T) {
	c := sampleCard()
	c.Body = "lots of body content"
	e := EntryFromCard(c, "cards/0001-test")
	// Entry has no body field at all — this is just a smoke test that
	// EntryFromCard doesn't accidentally start carrying body bytes.
	if e.ID != c.ID {
		t.Errorf("ID transfer: got %d", e.ID)
	}
	if e.Dir != "cards/0001-test" {
		t.Errorf("Dir = %q", e.Dir)
	}
}

func sampleCard() *card.Card {
	return &card.Card{
		SchemaVersion: card.SchemaVersion,
		ID:            1,
		UUID:          "abc",
		Title:         "Test card",
		Type:          card.TypeCard,
		Status:        card.StatusBacklog,
		Priority:      card.PriorityP2,
		Project:       "test",
		Created:       time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	}
}
