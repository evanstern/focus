package mcp

import (
	"context"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools wires every tool the design doc enumerates into srv.
// One tool per design-doc bullet under §"MCP server"; nothing speculative.
func registerTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_board",
		Description: "Return active + backlog card lists plus epics for the .focus/ board nearest the current working directory.",
	}, toolBoard)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_list",
		Description: "Filterable flat list of cards. Filters: status, project, priority, epic, owner, tag, type. Empty filter returns all cards.",
	}, toolList)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_show",
		Description: "Fetch a single card with full frontmatter and body by id.",
	}, toolShow)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_new",
		Description: "Create a new card. Returns the assigned id and uuid.",
	}, toolNew)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_activate",
		Description: "Move a card from backlog to active. Enforces the board's WIP limit unless force=true.",
	}, toolActivate)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_park",
		Description: "Move an active card back to backlog.",
	}, toolPark)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_done",
		Description: "Mark an active card done. Does NOT prompt on contract; use focus_show first if you want to verify the contract programmatically.",
	}, toolDone)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_kill",
		Description: "Move any card to archived. Always succeeds.",
	}, toolKill)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_revive",
		Description: "Move an archived card back to backlog.",
	}, toolRevive)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_edit_body",
		Description: "Replace a card's markdown body. Frontmatter is preserved verbatim.",
	}, toolEditBody)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_epic_list",
		Description: "Summary of every epic in this board with child-card progress.",
	}, toolEpicList)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_epic_show",
		Description: "Detail + child-status histogram for a single epic by id.",
	}, toolEpicShow)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_epic_add",
		Description: "Set the epic field on a card to point at a given epic id.",
	}, toolEpicAdd)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "focus_reindex",
		Description: "Rebuild .focus/index.json from the on-disk cards/. Preserves the next_id high-water mark.",
	}, toolReindex)
}

// EmptyArgs is the input type for tools that take no arguments.
// AddTool requires the In type to render as an object schema, so we
// use an empty struct — that produces "type":"object" with no
// properties.
type EmptyArgs struct{}

// IDArgs is the input type for tools that accept exactly one id.
type IDArgs struct {
	ID int `json:"id" jsonschema:"the card id (board-local integer)"`
}

// IDForceArgs adds the force flag for transitions that allow it.
type IDForceArgs struct {
	ID    int  `json:"id" jsonschema:"the card id (board-local integer)"`
	Force bool `json:"force,omitempty" jsonschema:"bypass from-status validation"`
}

// NewArgs is the input type for focus_new. Mirrors NewCardOpts but
// uses primitives the JSON schema generator can describe.
type NewArgs struct {
	Title    string `json:"title" jsonschema:"card title (free-form)"`
	Project  string `json:"project,omitempty" jsonschema:"project tag; defaults to the board's parent dir name"`
	Priority string `json:"priority,omitempty" jsonschema:"priority p0|p1|p2|p3; defaults to p2"`
	Type     string `json:"type,omitempty" jsonschema:"card or epic; defaults to card"`
	Epic     int    `json:"epic,omitempty" jsonschema:"parent epic id (omit for none)"`
	Slug     string `json:"slug,omitempty" jsonschema:"override the auto-derived folder slug"`
}

// EditBodyArgs replaces a card's markdown body.
type EditBodyArgs struct {
	ID   int    `json:"id"`
	Body string `json:"body" jsonschema:"new markdown body; replaces the existing body verbatim"`
}

// EpicAddArgs links a card to an epic.
type EpicAddArgs struct {
	EpicID int  `json:"epic_id"`
	CardID int  `json:"card_id"`
	Force  bool `json:"force,omitempty" jsonschema:"skip the epic-id existence check"`
}

// ListArgs filters a focus_list call. All fields are optional.
type ListArgs struct {
	Status   string `json:"status,omitempty"`
	Project  string `json:"project,omitempty"`
	Priority string `json:"priority,omitempty"`
	Owner    string `json:"owner,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Type     string `json:"type,omitempty"`
	Epic     int    `json:"epic,omitempty"`
}

// CardSummary is the per-card payload shape returned by every list-y
// tool. We mirror the index.Entry fields directly because that's the
// shape internal/board already produces; keeping them aligned avoids
// an extra translation layer.
type CardSummary struct {
	ID       int      `json:"id"`
	UUID     string   `json:"uuid"`
	Title    string   `json:"title"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Priority string   `json:"priority"`
	Project  string   `json:"project"`
	Epic     *int     `json:"epic,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Owner    string   `json:"owner,omitempty"`
	Created  string   `json:"created"`
	Dir      string   `json:"dir"`
}

func summaryFromEntry(e index.Entry) CardSummary {
	return CardSummary{
		ID:       e.ID,
		UUID:     e.UUID,
		Title:    e.Title,
		Type:     string(e.Type),
		Status:   string(e.Status),
		Priority: string(e.Priority),
		Project:  e.Project,
		Epic:     e.Epic,
		Tags:     e.Tags,
		Owner:    e.Owner,
		Created:  e.Created,
		Dir:      e.Dir,
	}
}

// BoardResult is the focus_board output: active, backlog, epics.
type BoardResult struct {
	Active  []CardSummary `json:"active"`
	Backlog []CardSummary `json:"backlog"`
	Epics   []CardSummary `json:"epics"`
}

func toolBoard(_ context.Context, _ *mcpsdk.CallToolRequest, _ EmptyArgs) (*mcpsdk.CallToolResult, BoardResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, BoardResult{}, err
	}
	v, err := b.Board()
	if err != nil {
		return nil, BoardResult{}, err
	}
	out := BoardResult{
		Active:  []CardSummary{},
		Backlog: []CardSummary{},
		Epics:   []CardSummary{},
	}
	for _, e := range v.Active {
		out.Active = append(out.Active, summaryFromEntry(e))
	}
	for _, e := range v.Backlog {
		out.Backlog = append(out.Backlog, summaryFromEntry(e))
	}
	for _, e := range v.Epics {
		out.Epics = append(out.Epics, summaryFromEntry(e))
	}
	return nil, out, nil
}

// ListResult wraps a flat card list.
type ListResult struct {
	Cards []CardSummary `json:"cards"`
}

func toolList(_ context.Context, _ *mcpsdk.CallToolRequest, args ListArgs) (*mcpsdk.CallToolResult, ListResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, ListResult{}, err
	}
	opts := board.ListOpts{
		Status:   card.Status(args.Status),
		Project:  args.Project,
		Priority: card.Priority(args.Priority),
		Owner:    args.Owner,
		Tag:      args.Tag,
		Type:     card.Type(args.Type),
	}
	if args.Epic > 0 {
		eid := args.Epic
		opts.Epic = &eid
	}
	entries, err := b.List(opts)
	if err != nil {
		return nil, ListResult{}, err
	}
	out := ListResult{Cards: []CardSummary{}}
	for _, e := range entries {
		out.Cards = append(out.Cards, summaryFromEntry(e))
	}
	return nil, out, nil
}

// ShowResult is focus_show: full card frontmatter + body.
type ShowResult struct {
	CardSummary
	Description string   `json:"description,omitempty"`
	Area        string   `json:"area,omitempty"`
	Contract    []string `json:"contract,omitempty"`
	DependsOn   []int    `json:"depends_on,omitempty"`
	Body        string   `json:"body"`
}

func toolShow(_ context.Context, _ *mcpsdk.CallToolRequest, args IDArgs) (*mcpsdk.CallToolResult, ShowResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, ShowResult{}, err
	}
	c, dir, err := b.LoadCard(args.ID)
	if err != nil {
		return nil, ShowResult{}, err
	}
	cs := CardSummary{
		ID:       c.ID,
		UUID:     c.UUID,
		Title:    c.Title,
		Type:     string(c.Type),
		Status:   string(c.Status),
		Priority: string(c.Priority),
		Project:  c.Project,
		Epic:     c.Epic,
		Tags:     c.Tags,
		Owner:    c.Owner,
		Dir:      dir,
	}
	if !c.Created.IsZero() {
		cs.Created = c.Created.Format("2006-01-02")
	}
	return nil, ShowResult{
		CardSummary: cs,
		Description: c.Description,
		Area:        c.Area,
		Contract:    c.Contract,
		DependsOn:   c.DependsOn,
		Body:        c.Body,
	}, nil
}

// NewResult is focus_new's output: the assigned id, uuid, and dir.
type NewResult struct {
	ID   int    `json:"id"`
	UUID string `json:"uuid"`
	Dir  string `json:"dir"`
}

func toolNew(_ context.Context, _ *mcpsdk.CallToolRequest, args NewArgs) (*mcpsdk.CallToolResult, NewResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, NewResult{}, err
	}
	opts := board.NewCardOpts{
		Project:  args.Project,
		Priority: card.Priority(args.Priority),
		Type:     card.Type(args.Type),
		Slug:     args.Slug,
	}
	if args.Epic > 0 {
		eid := args.Epic
		opts.Epic = &eid
	}
	c, dir, err := b.NewCard(args.Title, opts)
	if err != nil {
		return nil, NewResult{}, err
	}
	return nil, NewResult{ID: c.ID, UUID: c.UUID, Dir: dir}, nil
}

// TransitionResult is the standard output for transition tools.
type TransitionResult struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

func toolActivate(_ context.Context, _ *mcpsdk.CallToolRequest, args IDForceArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	return doTransition((*board.Board).Activate, args)
}

func toolPark(_ context.Context, _ *mcpsdk.CallToolRequest, args IDForceArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	return doTransition((*board.Board).Park, args)
}

func toolDone(_ context.Context, _ *mcpsdk.CallToolRequest, args IDForceArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	return doTransition((*board.Board).Done, args)
}

func toolKill(_ context.Context, _ *mcpsdk.CallToolRequest, args IDForceArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	return doTransition((*board.Board).Kill, args)
}

func toolRevive(_ context.Context, _ *mcpsdk.CallToolRequest, args IDForceArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	return doTransition((*board.Board).Revive, args)
}

func doTransition(op func(*board.Board, int, bool) (*card.Card, error), args IDForceArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, TransitionResult{}, err
	}
	c, err := op(b, args.ID, args.Force)
	if err != nil {
		return nil, TransitionResult{}, err
	}
	return nil, TransitionResult{ID: c.ID, Status: string(c.Status)}, nil
}

func toolEditBody(_ context.Context, _ *mcpsdk.CallToolRequest, args EditBodyArgs) (*mcpsdk.CallToolResult, TransitionResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, TransitionResult{}, err
	}
	if err := b.SetBody(args.ID, args.Body); err != nil {
		return nil, TransitionResult{}, err
	}
	c, _, err := b.LoadCard(args.ID)
	if err != nil {
		return nil, TransitionResult{}, err
	}
	return nil, TransitionResult{ID: c.ID, Status: string(c.Status)}, nil
}

// EpicListResult wraps the per-epic progress summary.
type EpicListResult struct {
	Epics []EpicProgress `json:"epics"`
}

// EpicProgress is the summary returned for one epic.
type EpicProgress struct {
	CardSummary
	Active   int `json:"active"`
	Backlog  int `json:"backlog"`
	Done     int `json:"done"`
	Archived int `json:"archived"`
	Total    int `json:"total"`
}

func progressFromBoard(p board.EpicProgress) EpicProgress {
	return EpicProgress{
		CardSummary: summaryFromEntry(p.Epic),
		Active:      p.Active,
		Backlog:     p.Backlog,
		Done:        p.Done,
		Archived:    p.Archive,
		Total:       p.Total(),
	}
}

func toolEpicList(_ context.Context, _ *mcpsdk.CallToolRequest, _ EmptyArgs) (*mcpsdk.CallToolResult, EpicListResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, EpicListResult{}, err
	}
	eps, err := b.EpicList()
	if err != nil {
		return nil, EpicListResult{}, err
	}
	out := EpicListResult{Epics: []EpicProgress{}}
	for _, p := range eps {
		out.Epics = append(out.Epics, progressFromBoard(p))
	}
	return nil, out, nil
}

func toolEpicShow(_ context.Context, _ *mcpsdk.CallToolRequest, args IDArgs) (*mcpsdk.CallToolResult, EpicProgress, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, EpicProgress{}, err
	}
	p, err := b.EpicShow(args.ID)
	if err != nil {
		return nil, EpicProgress{}, err
	}
	return nil, progressFromBoard(*p), nil
}

// EpicAddResult confirms the link.
type EpicAddResult struct {
	EpicID int `json:"epic_id"`
	CardID int `json:"card_id"`
}

func toolEpicAdd(_ context.Context, _ *mcpsdk.CallToolRequest, args EpicAddArgs) (*mcpsdk.CallToolResult, EpicAddResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, EpicAddResult{}, err
	}
	if err := b.EpicAdd(args.EpicID, args.CardID, args.Force); err != nil {
		return nil, EpicAddResult{}, err
	}
	return nil, EpicAddResult{EpicID: args.EpicID, CardID: args.CardID}, nil
}

// ReindexResult reports the rebuilt index size + next_id.
type ReindexResult struct {
	Cards  int `json:"cards"`
	NextID int `json:"next_id"`
}

func toolReindex(_ context.Context, _ *mcpsdk.CallToolRequest, _ EmptyArgs) (*mcpsdk.CallToolResult, ReindexResult, error) {
	b, err := resolveBoard()
	if err != nil {
		return nil, ReindexResult{}, err
	}
	idx, err := b.Reindex()
	if err != nil {
		return nil, ReindexResult{}, err
	}
	return nil, ReindexResult{Cards: len(idx.Cards), NextID: idx.NextID}, nil
}
