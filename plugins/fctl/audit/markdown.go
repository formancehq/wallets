package audit

import (
	"fmt"
	"strings"
)

// Markdown renders the generated tables the inventory document links to. It is
// written to docs/operations.generated.md and committed, so a change to
// openapi.yaml shows up as a reviewable diff rather than as drifting prose.
func (r *Report) Markdown() string {
	var b strings.Builder

	b.WriteString("# Wallets operation tables (generated)\n\n")
	b.WriteString("Do not edit. Regenerate with `just fctl-audit` from the repository root.\n\n")
	fmt.Fprintf(&b, "Source: `%s` (SHA-256 `%s`), based on product revision `%s`.\n", r.SpecDocument, r.SpecSHA256, r.BaseRevision)
	fmt.Fprintf(&b, "Legacy fctl baseline: `%s`.\n\n", r.BaselineRevision)

	b.WriteString("## 1. Operations by family\n\n")
	for _, family := range Families {
		records := r.ByFamily()[family]
		if len(records) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s (%d)\n\n", family, len(records))
		b.WriteString("| operationId | method | path | scopes | success | request body | legacy fctl command | blockers |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
		for _, rec := range records {
			fmt.Fprintf(&b, "| `%s` | %s | `%s` | %s | %s | %s | %s | %s |\n",
				rec.OperationID,
				rec.Method,
				rec.Path,
				code(rec.Scopes),
				successCell(rec.Successes),
				requestBodyCell(rec.Operation),
				code(rec.BaselineCommands),
				code(rec.Blockers),
			)
		}
		b.WriteString("\n")
	}

	b.WriteString("## 2. Risk matrix\n\n")
	b.WriteString("| operationId | mutating | moves funds | idempotency key | paginated | multiple success shapes |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, rec := range r.Operations {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n",
			rec.OperationID,
			mark(rec.Mutating()),
			mark(rec.Risk.Moves),
			mark(rec.Risk.Idempotent),
			mark(rec.Risk.Paginated),
			mark(rec.Risk.MultiSuccess),
		)
	}
	fmt.Fprintf(&b, "\nNo operation on this surface returns secret or display-once material, "+
		"destroys caller state, or streams: those four counts are %d, %d, %d and %d.\n\n",
		r.Totals.SecretBearing, r.Totals.DisplayOnce, r.Totals.Destructive, r.Totals.Streaming)

	b.WriteString("## 3. Legacy fctl baseline\n\n")
	fmt.Fprintf(&b, "%d constructors under `cmd/wallets/` at `%s`: %d grouping-only, %d executable leaves.\n\n",
		r.Totals.BaselineCommands, r.Totals.BaselineRevisionShort(), r.Totals.BaselineGroupings, r.Totals.BaselineLeaves)
	b.WriteString("| command | aliases | kind | operationId | source |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, c := range r.Baseline {
		kind := "leaf"
		if c.Grouping {
			kind = "grouping"
		}
		ops := code(c.Ops)
		if c.Grouping {
			ops = "—"
		} else if len(c.Ops) == 0 {
			ops = "excluded: " + c.Exclusion
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | `%s` |\n",
			c.Path, code(c.Aliases), kind, ops, c.Source)
	}
	b.WriteString("\n")

	b.WriteString("## 4. Operations with no legacy precedent\n\n")
	b.WriteString("| operationId | method | path | family |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, rec := range r.Operations {
		if len(rec.BaselineCommands) > 0 {
			continue
		}
		fmt.Fprintf(&b, "| `%s` | %s | `%s` | %s |\n", rec.OperationID, rec.Method, rec.Path, rec.Family)
	}
	b.WriteString("\n")

	b.WriteString("## 5. Blockers\n\n")
	for _, blocker := range r.Blockers {
		scope := "whole surface"
		if len(blocker.OperationIDs) > 0 {
			scope = strings.Join(backtick(blocker.OperationIDs), ", ")
		}
		fmt.Fprintf(&b, "### %s — %s\n\n", blocker.ID, blocker.Title)
		fmt.Fprintf(&b, "- Applies to: %s\n", scope)
		fmt.Fprintf(&b, "- Evidence: %s\n", blocker.Evidence)
		fmt.Fprintf(&b, "- Consequence: %s\n", blocker.Consequence)
		fmt.Fprintf(&b, "- Fix in: %s\n\n", blocker.FixIn)
	}

	b.WriteString("## 6. Spec-versus-server divergences\n\n")
	for _, d := range r.Divergences {
		scope := "document-level"
		if len(d.OperationIDs) > 0 {
			scope = strings.Join(backtick(d.OperationIDs), ", ")
		}
		fmt.Fprintf(&b, "### %s — %s\n\n", d.ID, d.Title)
		fmt.Fprintf(&b, "- Applies to: %s\n", scope)
		fmt.Fprintf(&b, "- Spec says: %s\n", d.Spec)
		fmt.Fprintf(&b, "- Server does: %s\n\n", d.Server)
	}

	b.WriteString("## 7. Derived totals\n\n")
	t := r.Totals
	rows := [][2]any{
		{"operations declared", t.Operations},
		{"unique operationIds", t.UniqueOperationIDs},
		{"deprecated", t.DeprecatedOperations},
		{"declaring a security block", t.OperationsWithSecurity},
		{"declaring wallets:read", t.ReadScoped},
		{"declaring wallets:write", t.WriteScoped},
		{"legacy fctl constructors", t.BaselineCommands},
		{"legacy grouping-only commands", t.BaselineGroupings},
		{"legacy executable leaves", t.BaselineLeaves},
		{"leaves mapped to an operation", t.BaselineMapped},
		{"leaves excluded with evidence", t.BaselineExcluded},
		{"distinct operations the baseline reaches", t.BaselineTargets},
		{"operations with a legacy precedent", t.WithBaseline},
		{"operations without a legacy precedent", t.WithoutBaseline},
		{"operations carrying an operation-scoped blocker", t.Blocked},
		{"operations with no operation-scoped blocker", t.Admissible},
		{"module-level blockers", t.ModuleBlockers},
		{"recorded divergences", t.Divergences},
		{"mutating operations", t.Mutating},
		{"operations accepting an idempotency key", t.Idempotent},
		{"operations declaring cursor pagination", t.Paginated},
		{"operations that move funds", t.FundMoving},
		{"operations with multiple success shapes", t.MultiSuccess},
		{"operations returning secret material", t.SecretBearing},
		{"display-once operations", t.DisplayOnce},
		{"destructive operations", t.Destructive},
		{"streaming operations", t.Streaming},
	}
	b.WriteString("| quantity | value |\n| --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | %v |\n", row[0], row[1])
	}
	b.WriteString("\nThe last column of §7 is a count of proven facts. None of it is an\n")
	b.WriteString("acceptance claim: no runtime, component, OCI install or dual-host gate is\n")
	b.WriteString("satisfied by this inventory.\n")

	return b.String()
}

// BaselineRevisionShort is on Totals so the template can reach it without the
// report; it renders the pinned fctl revision the way commit messages do.
func (Totals) BaselineRevisionShort() string { return BaselineRevision[:7] }

func successCell(successes []Success) string {
	parts := make([]string, 0, len(successes))
	for _, s := range successes {
		if s.Body == "" {
			parts = append(parts, s.Code+" (no body)")
			continue
		}
		parts = append(parts, fmt.Sprintf("%s `%s`", s.Code, s.Body))
	}
	return strings.Join(parts, " / ")
}

func requestBodyCell(op Operation) string {
	switch {
	case op.RequestBody != "":
		return "`" + op.RequestBody + "`"
	case op.RequestBodyInline:
		return "inline (D6)"
	default:
		return "—"
	}
}

func code(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	return strings.Join(backtick(values), ", ")
}

func backtick(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, "`"+v+"`")
	}
	return out
}

func mark(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
