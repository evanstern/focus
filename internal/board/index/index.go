// Package index manages the .focus/index.json derived cache.
//
// The index is the read path for every command that doesn't load card
// bodies — `focus board`, `focus list`, `focus epic list`, the TUI's
// board view, etc. It holds frontmatter only; bodies are loaded on
// demand by `focus show`/`focus edit`. This is where the speed win
// over v1's bash implementation comes from.
//
// Writes go through Save which uses google/renameio/v2 for an atomic
// temp-file → fsync → rename, so a crash mid-write can never leave a
// truncated index on disk. All mutating callers must hold the
// .focus/.lock flock for the duration of the read-modify-write cycle
// (see internal/board/lock).
package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/evanstern/focus/internal/board/card"
	"github.com/google/renameio/v2"
)

// SchemaVersion is stamped on every index.json we write. Bumped only
// when the index file format changes in a backwards-incompatible way;
// independent of the card schema_version.
const SchemaVersion = 2

// FileName is the index file's name relative to .focus/.
const FileName = "index.json"

// Index is the on-disk shape of .focus/index.json.
type Index struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	NextID        int       `json:"next_id"`
	Cards         []Entry   `json:"cards"`
}

// Entry is one card's row in the index. Frontmatter only — never the
// body.
type Entry struct {
	ID          int           `json:"id"`
	UUID        string        `json:"uuid"`
	Title       string        `json:"title"`
	Type        card.Type     `json:"type"`
	Status      card.Status   `json:"status"`
	Priority    card.Priority `json:"priority"`
	Project     string        `json:"project"`
	Epic        *int          `json:"epic,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Owner       string        `json:"owner,omitempty"`
	Description string        `json:"description,omitempty"`
	Area        string        `json:"area,omitempty"`
	Created     string        `json:"created"`
	Dir         string        `json:"dir"`
}

// EntryFromCard converts a fully-parsed Card plus its on-disk dir
// (relative to the board root, e.g. "cards/0142-ship-the-feature")
// into an index Entry. The body is intentionally dropped.
func EntryFromCard(c *card.Card, dir string) Entry {
	e := Entry{
		ID:          c.ID,
		UUID:        c.UUID,
		Title:       c.Title,
		Type:        c.Type,
		Status:      c.Status,
		Priority:    c.Priority,
		Project:     c.Project,
		Epic:        c.Epic,
		Tags:        c.Tags,
		Owner:       c.Owner,
		Description: c.Description,
		Area:        c.Area,
		Dir:         dir,
	}
	if !c.Created.IsZero() {
		e.Created = c.Created.Format("2006-01-02")
	}
	return e
}

// Load reads .focus/index.json from a board's .focus/ directory.
// Returns os.ErrNotExist if no index has been written yet (first
// `focus new` is the typical writer); callers should treat that as an
// empty index, not a hard error.
func Load(focusDir string) (*Index, error) {
	path := filepath.Join(focusDir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if idx.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("%s schema_version %d unsupported (this binary speaks %d); run `focus reindex`", path, idx.SchemaVersion, SchemaVersion)
	}
	return &idx, nil
}

// Save writes the index to .focus/index.json atomically via
// renameio/v2 (temp file in the same dir → fsync → rename). The
// rename is atomic on POSIX, so concurrent readers either see the old
// snapshot or the new one — never a torn write.
//
// Save also stamps GeneratedAt to the current UTC time and sets
// SchemaVersion to the constant; callers don't need to fill those.
//
// Sorted by id for stable diffs and deterministic output across
// platforms.
func Save(focusDir string, idx *Index) error {
	idx.SchemaVersion = SchemaVersion
	idx.GeneratedAt = time.Now().UTC()
	sortByID(idx.Cards)

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	path := filepath.Join(focusDir, FileName)
	return renameio.WriteFile(path, data, 0o644)
}

// LoadOrEmpty returns the existing index or, if none has been written,
// a fresh empty Index with NextID=1. Convenience for command code
// that needs to read-then-modify; the alternative is each handler
// open-coding the os.ErrNotExist check.
func LoadOrEmpty(focusDir string) (*Index, error) {
	idx, err := Load(focusDir)
	if err == nil {
		return idx, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	return &Index{SchemaVersion: SchemaVersion, NextID: 1}, nil
}

// AllocateID returns the next available card id and increments NextID
// in place. The high-water-mark invariant (designs/focus-v2.md
// §"`next_id` allocation rule") is maintained by callers via Save.
func (idx *Index) AllocateID() int {
	if idx.NextID < 1 {
		idx.NextID = 1
	}
	id := idx.NextID
	idx.NextID++
	return id
}

// Upsert replaces the entry for e.ID if it exists, else appends it.
// Sorting is deferred to Save.
func (idx *Index) Upsert(e Entry) {
	for i := range idx.Cards {
		if idx.Cards[i].ID == e.ID {
			idx.Cards[i] = e
			return
		}
	}
	idx.Cards = append(idx.Cards, e)
}

// Find returns the entry with the given id, or nil if not present.
func (idx *Index) Find(id int) *Entry {
	for i := range idx.Cards {
		if idx.Cards[i].ID == id {
			return &idx.Cards[i]
		}
	}
	return nil
}

// Remove drops the entry with the given id from the index. It does NOT
// lower NextID — burned ids stay burned (designs/focus-v2.md
// §"`next_id` allocation rule"). Returns true if an entry was removed.
func (idx *Index) Remove(id int) bool {
	for i := range idx.Cards {
		if idx.Cards[i].ID == id {
			idx.Cards = append(idx.Cards[:i], idx.Cards[i+1:]...)
			return true
		}
	}
	return false
}

// MaxID returns the highest id present in the index, or 0 if empty.
// Used by Reindex to compute the next_id high-water mark.
func (idx *Index) MaxID() int {
	max := 0
	for i := range idx.Cards {
		if idx.Cards[i].ID > max {
			max = idx.Cards[i].ID
		}
	}
	return max
}

// sortByID is an in-place insertion sort. We avoid pulling in the
// stdlib sort package for this one use; insertion sort is fine at
// board sizes (typically <1000 cards) and the implementation is
// trivial enough to verify by inspection.
func sortByID(es []Entry) {
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && es[j-1].ID > es[j].ID; j-- {
			es[j-1], es[j] = es[j], es[j-1]
		}
	}
}
