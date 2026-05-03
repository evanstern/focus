package board

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
	"github.com/evanstern/focus/internal/board/lock"
	"github.com/google/uuid"
)

// NewCardOpts captures the optional fields callers may set when
// creating a card. Empty fields fall back to the defaults: priority
// p2, type card, status backlog, project = board's parent dir name.
type NewCardOpts struct {
	Project  string
	Priority card.Priority
	Type     card.Type
	Epic     *int
	Slug     string
}

// NewCard creates a new card on disk and updates the index. Acquires
// the .focus/.lock for the duration of allocate-id → write-card →
// write-index so concurrent `focus new` (or MCP equivalent) calls
// can't double-allocate or corrupt the index.
//
// Returns the in-memory Card with id and uuid populated, plus the
// directory name relative to .focus/cards/.
func (b *Board) NewCard(title string, opts NewCardOpts) (*card.Card, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, "", errors.New("title is required")
	}

	cardType := opts.Type
	if cardType == "" {
		cardType = card.TypeCard
	}
	priority := opts.Priority
	if priority == "" {
		priority = card.PriorityP2
	}
	project := opts.Project
	if project == "" {
		project = filepath.Base(b.Root)
	}

	slug := opts.Slug
	if slug == "" {
		s, err := card.Slugify(title)
		if err != nil {
			return nil, "", err
		}
		slug = s
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, "", fmt.Errorf("uuid generation failed (clock skew?): %w", err)
	}

	c := &card.Card{
		SchemaVersion: card.SchemaVersion,
		UUID:          id.String(),
		Title:         title,
		Type:          cardType,
		Status:        card.StatusBacklog,
		Priority:      priority,
		Project:       project,
		Created:       time.Now().UTC().Truncate(24 * time.Hour),
		Epic:          opts.Epic,
	}

	var dirName string
	err = lock.With(b.Dir, func() error {
		idx, err := index.LoadOrEmpty(b.Dir)
		if err != nil {
			return err
		}
		c.ID = idx.AllocateID()

		dirName = card.DirName(c.ID, slug)
		dirPath := filepath.Join(b.CardsDir(), dirName)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dirPath, err)
		}

		filePath := filepath.Join(dirPath, CardFileName)
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("card file %s already exists", filePath)
		}

		data, err := card.Marshal(c)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", filePath, err)
		}

		idx.Upsert(index.EntryFromCard(c, filepath.ToSlash(filepath.Join(CardsDirName, dirName))))
		return index.Save(b.Dir, idx)
	})
	if err != nil {
		return nil, "", err
	}
	return c, dirName, nil
}

// LoadCard reads a card by id from disk, including its body. Used by
// `focus show`, `focus edit`, and any MCP tool that needs body
// content. Read-only — no lock taken.
func (b *Board) LoadCard(id int) (*card.Card, string, error) {
	dirName, err := b.FindCardDir(id)
	if err != nil {
		return nil, "", err
	}
	path := b.CardFile(dirName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	c, err := card.Parse(data)
	if err != nil {
		return nil, "", fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, "", fmt.Errorf("%s: %w", path, err)
	}
	return c, dirName, nil
}

// SaveCard writes a Card back to its INDEX.md and updates the index
// row. Caller must hold the lock — this is a building block for the
// transition operations below, not a public mutation primitive.
func (b *Board) saveCardLocked(c *card.Card, dirName string, idx *index.Index) error {
	data, err := card.Marshal(c)
	if err != nil {
		return err
	}
	path := b.CardFile(dirName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	idx.Upsert(index.EntryFromCard(c, filepath.ToSlash(filepath.Join(CardsDirName, dirName))))
	return nil
}

// transition moves a card to the target status with a caller-supplied
// from-status check. Acquires the lock for the read-modify-write
// cycle. Force=true bypasses the from-status check entirely.
func (b *Board) transition(id int, to card.Status, allowedFrom func(card.Status) error, force bool) (*card.Card, error) {
	var result *card.Card
	err := lock.With(b.Dir, func() error {
		c, dirName, err := b.LoadCard(id)
		if err != nil {
			return err
		}
		if !force && allowedFrom != nil {
			if err := allowedFrom(c.Status); err != nil {
				return err
			}
		}
		c.Status = to

		idx, err := index.LoadOrEmpty(b.Dir)
		if err != nil {
			return err
		}
		if err := b.saveCardLocked(c, dirName, idx); err != nil {
			return err
		}
		if err := index.Save(b.Dir, idx); err != nil {
			return err
		}
		result = c
		return nil
	})
	return result, err
}

// ErrWIPLimit is returned by Activate when activating would exceed
// the board's WIP limit and force=false. CLI prints a hint about
// --force; MCP surfaces it as a tool error.
var ErrWIPLimit = errors.New("WIP limit reached")

// Activate transitions backlog → active. Enforces the board's WIP
// limit unless force=true. The check counts active cards (excluding
// the one being activated) under the same lock as the transition so
// two concurrent `focus activate` calls can't both squeak past the
// limit.
func (b *Board) Activate(id int, force bool) (*card.Card, error) {
	cfg, err := b.LoadConfig()
	if err != nil {
		return nil, err
	}
	limit := cfg.EffectiveWIPLimit()

	var result *card.Card
	err = lock.With(b.Dir, func() error {
		c, dirName, err := b.LoadCard(id)
		if err != nil {
			return err
		}
		if c.Status != card.StatusBacklog && !force {
			return fmt.Errorf("can't activate: card is %s, must be backlog", c.Status)
		}
		idx, err := index.LoadOrEmpty(b.Dir)
		if err != nil {
			return err
		}
		if !force {
			active := 0
			for _, e := range idx.Cards {
				if e.Status == card.StatusActive && e.Type != card.TypeEpic && e.ID != id {
					active++
				}
			}
			if active >= limit {
				return fmt.Errorf("%w (%d active, limit %d); pass --force to override", ErrWIPLimit, active, limit)
			}
		}
		c.Status = card.StatusActive
		if err := b.saveCardLocked(c, dirName, idx); err != nil {
			return err
		}
		if err := index.Save(b.Dir, idx); err != nil {
			return err
		}
		result = c
		return nil
	})
	return result, err
}

// Park transitions active → backlog.
func (b *Board) Park(id int, force bool) (*card.Card, error) {
	return b.transition(id, card.StatusBacklog, func(from card.Status) error {
		if from != card.StatusActive {
			return fmt.Errorf("can't park: card is %s, must be active", from)
		}
		return nil
	}, force)
}

// Done transitions active → done. Contract enforcement is the
// caller's job (the CLI prompts on tty; the MCP just transitions);
// pass force=true to skip validation entirely.
func (b *Board) Done(id int, force bool) (*card.Card, error) {
	return b.transition(id, card.StatusDone, func(from card.Status) error {
		if from != card.StatusActive {
			return fmt.Errorf("can't mark done: card is %s, must be active", from)
		}
		return nil
	}, force)
}

// Kill transitions any status → archived. The "force" flag is ignored
// because kill is always allowed by design.
func (b *Board) Kill(id int, _ bool) (*card.Card, error) {
	return b.transition(id, card.StatusArchived, nil, true)
}

// Revive transitions archived → backlog.
func (b *Board) Revive(id int, force bool) (*card.Card, error) {
	return b.transition(id, card.StatusBacklog, func(from card.Status) error {
		if from != card.StatusArchived {
			return fmt.Errorf("can't revive: card is %s, must be archived", from)
		}
		return nil
	}, force)
}

// EpicAdd sets the epic field on a card to point at the given epic
// id. Validates that epicID names an existing card with type:epic
// unless force is true.
func (b *Board) EpicAdd(epicID, cardID int, force bool) error {
	return lock.With(b.Dir, func() error {
		idx, err := index.LoadOrEmpty(b.Dir)
		if err != nil {
			return err
		}

		if !force {
			ep := idx.Find(epicID)
			if ep == nil {
				return fmt.Errorf("epic %d not found", epicID)
			}
			if ep.Type != card.TypeEpic {
				return fmt.Errorf("card %d is not an epic (type=%s)", epicID, ep.Type)
			}
		}

		c, dirName, err := b.LoadCard(cardID)
		if err != nil {
			return err
		}
		eid := epicID
		c.Epic = &eid
		if err := b.saveCardLocked(c, dirName, idx); err != nil {
			return err
		}
		return index.Save(b.Dir, idx)
	})
}

// Reindex walks .focus/cards/ and rewrites index.json from scratch.
// Use after hand-edits, git merges, or any state-bypassing
// operation. Preserves the previous next_id high-water mark per
// designs/focus-v2.md §"Recovery".
func (b *Board) Reindex() (*index.Index, error) {
	var result *index.Index
	err := lock.With(b.Dir, func() error {
		entries, err := os.ReadDir(b.CardsDir())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				entries = nil
			} else {
				return err
			}
		}

		newIdx := &index.Index{}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(b.CardsDir(), e.Name(), CardFileName)
			data, err := os.ReadFile(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return fmt.Errorf("read %s: %w", path, err)
			}
			c, err := card.Parse(data)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
			if err := c.Validate(); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			newIdx.Cards = append(newIdx.Cards,
				index.EntryFromCard(c, filepath.ToSlash(filepath.Join(CardsDirName, e.Name()))),
			)
		}

		newIdx.NextID = newIdx.MaxID() + 1

		if old, err := index.Load(b.Dir); err == nil {
			if old.NextID > newIdx.NextID {
				newIdx.NextID = old.NextID
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if err := index.Save(b.Dir, newIdx); err != nil {
			return err
		}
		result = newIdx
		return nil
	})
	return result, err
}

// ListOpts narrows a List call. Empty fields apply no filter; the
// CLI translates --project, --priority, etc. into these fields.
type ListOpts struct {
	Status   card.Status
	Project  string
	Priority card.Priority
	Epic     *int
	Owner    string
	Tag      string
	Type     card.Type
}

// List returns the cards matching opts, ordered by id. Read-only;
// callers do not need to hold the lock.
func (b *Board) List(opts ListOpts) ([]index.Entry, error) {
	idx, err := index.LoadOrEmpty(b.Dir)
	if err != nil {
		return nil, err
	}
	out := make([]index.Entry, 0, len(idx.Cards))
	for _, e := range idx.Cards {
		if opts.Status != "" && e.Status != opts.Status {
			continue
		}
		if opts.Project != "" && e.Project != opts.Project {
			continue
		}
		if opts.Priority != "" && e.Priority != opts.Priority {
			continue
		}
		if opts.Epic != nil {
			if e.Epic == nil || *e.Epic != *opts.Epic {
				continue
			}
		}
		if opts.Owner != "" && e.Owner != opts.Owner {
			continue
		}
		if opts.Type != "" && e.Type != opts.Type {
			continue
		}
		if opts.Tag != "" {
			found := false
			for _, t := range e.Tags {
				if t == opts.Tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// BoardView is the active+backlog snapshot used by `focus board` and
// the TUI's default view. Done and archived are excluded by design.
type BoardView struct {
	Active  []index.Entry
	Backlog []index.Entry
	Epics   []index.Entry
}

// Board returns the default board view: active cards, backlog cards,
// and epics. Read-only.
func (b *Board) Board() (*BoardView, error) {
	idx, err := index.LoadOrEmpty(b.Dir)
	if err != nil {
		return nil, err
	}
	v := &BoardView{}
	for _, e := range idx.Cards {
		if e.Type == card.TypeEpic {
			if e.Status != card.StatusArchived && e.Status != card.StatusDone {
				v.Epics = append(v.Epics, e)
			}
			continue
		}
		switch e.Status {
		case card.StatusActive:
			v.Active = append(v.Active, e)
		case card.StatusBacklog:
			v.Backlog = append(v.Backlog, e)
		}
	}
	sortByPrioThenID(v.Active)
	sortByPrioThenID(v.Backlog)
	sort.Slice(v.Epics, func(i, j int) bool { return v.Epics[i].ID < v.Epics[j].ID })
	return v, nil
}

// EpicProgress is the progress summary for one epic: how many child
// cards are in each status. Used by `focus epic <id>`.
type EpicProgress struct {
	Epic    index.Entry
	Active  int
	Backlog int
	Done    int
	Archive int
}

// Total returns the combined child-card count.
func (p EpicProgress) Total() int {
	return p.Active + p.Backlog + p.Done + p.Archive
}

// EpicShow returns the epic itself plus a child-status histogram.
// Errors if id doesn't refer to an existing epic.
func (b *Board) EpicShow(id int) (*EpicProgress, error) {
	idx, err := index.LoadOrEmpty(b.Dir)
	if err != nil {
		return nil, err
	}
	e := idx.Find(id)
	if e == nil {
		return nil, fmt.Errorf("epic %d not found", id)
	}
	if e.Type != card.TypeEpic {
		return nil, fmt.Errorf("card %d is not an epic", id)
	}
	p := &EpicProgress{Epic: *e}
	for _, c := range idx.Cards {
		if c.Epic == nil || *c.Epic != id {
			continue
		}
		switch c.Status {
		case card.StatusActive:
			p.Active++
		case card.StatusBacklog:
			p.Backlog++
		case card.StatusDone:
			p.Done++
		case card.StatusArchived:
			p.Archive++
		}
	}
	return p, nil
}

// EpicList returns all epics in the board ordered by id, regardless
// of status. Useful for `focus epic list`.
func (b *Board) EpicList() ([]EpicProgress, error) {
	idx, err := index.LoadOrEmpty(b.Dir)
	if err != nil {
		return nil, err
	}
	var out []EpicProgress
	for _, e := range idx.Cards {
		if e.Type != card.TypeEpic {
			continue
		}
		p := EpicProgress{Epic: e}
		for _, c := range idx.Cards {
			if c.Epic == nil || *c.Epic != e.ID {
				continue
			}
			switch c.Status {
			case card.StatusActive:
				p.Active++
			case card.StatusBacklog:
				p.Backlog++
			case card.StatusDone:
				p.Done++
			case card.StatusArchived:
				p.Archive++
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Epic.ID < out[j].Epic.ID })
	return out, nil
}

// priorityRank gives a stable numeric ordering: p0 < p1 < p2 < p3.
// Lower number = higher priority. Used by board-view sorting.
func priorityRank(p card.Priority) int {
	switch p {
	case card.PriorityP0:
		return 0
	case card.PriorityP1:
		return 1
	case card.PriorityP2:
		return 2
	case card.PriorityP3:
		return 3
	}
	return 9
}

func sortByPrioThenID(es []index.Entry) {
	sort.Slice(es, func(i, j int) bool {
		ri, rj := priorityRank(es[i].Priority), priorityRank(es[j].Priority)
		if ri != rj {
			return ri < rj
		}
		return es[i].ID < es[j].ID
	})
}
