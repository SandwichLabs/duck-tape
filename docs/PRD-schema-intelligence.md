# PRD — Duck Tape Schema Intelligence

**Status:** Draft for issue slicing
**Related:** `docs/LSP_PLAN.md` (original design exploration)
**Owner:** TBD
**Target binary:** `dt` (no new binaries; all surfaces ship inside `dt`)

---

## 1. Summary

Add schema-aware intelligence to Duck Tape, built as **one shared core** exposed
through **three surfaces**:

1. **Track 1 — Developer LSP** (`dt lsp`): completion, hover, diagnostics for
   humans writing SQL in Neovim/VS Code, driven by an in-project
   `.ducktape.yaml`.
2. **Track 2 — Context cache + validation CLI**: a persisted schema cache and
   new `dt validate` / `dt explain` (+ `dt describe`, `dt cache`) commands for
   scripting and terminal work.
3. **Track 3 — MCP server** (`dt mcp`): agent-native tools (schema, validate,
   explain, run) over MCP, configured by the same `.ducktape.yaml`.

The design principle established during scoping: **LSP is the wrong protocol for
agents and CLI users; the capabilities under it (schema cache + non-executing
validation) are right for all three audiences.** Build the core once; skin it
three ways.

### 1.1 Why one core, three surfaces

| Capability (shared core) | Track 1 (LSP) | Track 2 (CLI) | Track 3 (MCP) |
|---|---|---|---|
| Project config discovery (`.ducktape.yaml`) | root_dir + settings | project context | project context |
| Schema cache (persisted, fingerprinted) | completion/hover source | `dt context`, `dt cache` | `list_schemas`, `describe` |
| Prepare-based validation (no execution) | diagnostics | `dt validate` | `validate` tool |
| Explain / cost | (future codelens) | `dt explain` | `explain` tool |
| Read-only federated engine | yes | yes | yes |

---

## 2. Goals & non-goals

### Goals
- A single project-local config (`.ducktape.yaml`) that all three surfaces read.
- A persisted, fingerprinted schema cache spanning the workspace DB + all
  attached connections (Postgres/MySQL/SQLite/HTTPFS/files).
- Non-executing validation (`prepare`) and `EXPLAIN` exposed as first-class
  capabilities — important for data-lake work where running to find a typo is
  expensive.
- Read-only-by-default connections so no surface ever locks an external engine.
- Agent-native access via MCP so a coding agent (e.g. Claude Code) can analyze a
  data lake with grounded schema and cheap validation.

### Non-goals (this initiative)
- Query execution UX changes beyond what validation/explain require.
- Result visualization, notebooks, or a TUI.
- Multi-dialect parsing engines (DuckDB `ATTACH` already federates dialects).
- Auth/secret-manager integration beyond env-var interpolation.
- A bespoke VS Code extension (generic LSP client config is sufficient for v1).

---

## 3. Personas & primary use cases

- **P1 — SQL author in an editor.** Writes `.sql` files against a project's
  attached databases; wants completion/hover/diagnostics. → Track 1.
- **P2 — Shell/data-pipeline author.** Scripts `dt` in Makefiles/CI; wants to
  validate a query before an expensive run and dump schema context. → Track 2.
- **P3 — Coding agent (Claude Code).** Analyzes data-lake data via `dt` to
  produce reports; wants grounded schema up front and cheap validation to avoid
  scanning TB of data on a typo. → Track 3 (and Track 2 verbs via bash).

---

## 4. `.ducktape.yaml` — project configuration (foundational)

A project-local file discovered by walking up from the working directory
(like `.git`). It is the single source of truth for all three surfaces and
supersedes/extends the global `~/.dt/config.yaml`.

### 4.1 Discovery & precedence
- Search order: nearest `.ducktape.yaml` walking up from cwd → stop at repo root
  or filesystem root.
- If found, it defines the active workspace/connections for `dt` in that tree.
- `extends:` may reference a global workspace name to inherit its connections;
  inline `connections:` override/augment.
- Absent file → fall back to today's global `~/.dt` behavior (backward
  compatible).

### 4.2 Proposed schema
```yaml
version: 1
workspace: analytics            # logical name; also cache namespace
extends: dev                    # optional: inherit from global ~/.dt workspace

connections:
  - name: prod_pg
    type: POSTGRES
    conn_string: "${PROD_PG_URL}"   # env interpolation; never commit secrets
    read_only: true                 # default true
  - name: lake
    type: HTTPFS
    conn_string: "s3://bucket/warehouse"
  - name: local_events
    type: SQLITE
    conn_string: "./data/events.db"

extensions: [httpfs, parquet, json, spatial]

cache:
  path: .ducktape/cache.json      # project-local; gitignored
  ttl_seconds: 300                # background refresh cadence
  scope: all                      # or a list of connection names

lsp:
  connections: [prod_pg, lake]    # subset exposed to hinting (default: all)

mcp:
  enabled: true
  tools: [list_schemas, describe, validate, explain, run]
  run:
    max_rows: 1000                # safety cap on the run tool
    allow_write: false
```

### 4.3 Requirements
- Env-var interpolation (`${VAR}`) in `conn_string`; a missing var is a clear
  error, never a silent empty string.
- Secrets must never be logged or written into the cache file.
- `.ducktape/` cache dir added to `.gitignore` guidance; `.ducktape.yaml` itself
  is safe to commit (no inline secrets expected).
- Validation of the file with actionable error messages (unknown keys, bad
  types) — do not silently ignore.

---

## 5. Shared core — packages

```
project/                # .ducktape.yaml discovery, parse, env-interp, merge with global
schema/
  introspect.go         # queries moved from cmd/context.go (SHOW ALL TABLES, information_schema, duckdb_functions)
  cache.go              # Cache model, fingerprint, JSON load/save, invalidation
engine/
  validate.go           # PrepareContext-based validation + DuckDB error → position parser
  explain.go            # EXPLAIN / EXPLAIN ANALYZE (opt-in) wrappers
cmd/database.go         # + WithReadOnly option (access_mode=read_only); :memory: forced RW
```

Track packages (`lsp/`, `mcp/`) depend only on the shared core, never on each
other.

### 5.1 Connection safety (shared)
- Attached connections keep existing `ReadWriteMode()` → `, READ_ONLY` default;
  add `META_JOURNAL_MODE 'WAL'`, `META_BUSY_TIMEOUT 500` for SQLite/MySQL
  attachments to avoid starvation.
- Workspace `dt.db` opened `access_mode=read_only` in LSP/MCP/validate paths;
  `:memory:` forced read-write.
- Fix the known bug: `connection.ConnectionConfigForm()` does not set
  `EnableWrite` from the form (`PRODUCTION_READINESS.md` §1.1).

---

## 6. Track 1 — Developer LSP (`dt lsp`)

**Depends on:** shared core (§4, §5).
**Library:** `github.com/tliron/glsp` (stdio JSON-RPC + `protocol_3_16`).

### Functional requirements
- `dt lsp` starts a stdio LSP server; completes `initialize`/`shutdown`;
  advertises incremental text sync, completion (triggers `.`, ` `), hover,
  signature help (triggers `(`, `,`), and diagnostics.
- Discovers `.ducktape.yaml` from the LSP `rootUri`; opens the federated engine
  read-only; builds/loads the schema cache.
- **Diagnostics:** statement splitter isolates the statement under edit →
  `PrepareContext` on complete statements → parse DuckDB error line/col → publish
  `Diagnostic`. Incomplete/actively-typed statements never produce errors.
- **Completion (Phase A):** clause heuristics — tables after `FROM`/`JOIN`,
  columns in `SELECT`/`WHERE`/`ON`, alias-resolved `alias.` → that table's
  columns. Cache-backed; Friendly SQL respected (validation via engine, not a
  generic grammar).
- **Completion (Phase B, optional):** tree-sitter-sql CST
  (`smacker/go-tree-sitter`) for precise node context + relevance scoring.
- **Hover / signature help:** table→columns+types, column→type+owner,
  function→signature from `duckdb_functions()`.
- **Runtime reconfig:** `workspace/didChangeConfiguration` tears down
  connections, clears cache, rebinds (switch workspace/target without restart).
- **Extension watcher:** background poll of `duckdb_extensions()`; on
  `loaded:false→true`, merge new function signatures.
- **Editor docs:** copy-paste config for Neovim (`vim.lsp.config`, `cmd = {"dt","lsp"}`,
  `filetypes={"sql"}`, root markers `{".ducktape.yaml",".git"}`), Helix, VS Code.

### Acceptance
- Opening a `.sql` file in a project with `.ducktape.yaml` yields table/column
  completion from all in-scope attached DBs within ~10 ms (warm cache).
- A query with a nonexistent column shows an inline binder-error diagnostic.
- `SELECT * EXCLUDE (...)`, `GROUP BY ALL`, `FROM 'x.parquet'` produce **no**
  false-positive diagnostics.

---

## 7. Track 2 — Context cache + validation CLI

**Depends on:** shared core (§4, §5). **This track delivers the shared core’s
CLI surface and should land first — it unblocks Tracks 1 and 3.**

### Commands
- `dt context [--json] [--connections a,b] [--tables 'glob'] [--summary]`
  Refactored onto the schema cache; adds `--json` (structured) and scoping so an
  agent/human can pull a slice instead of the whole lake schema. Backward
  compatible default output.
- `dt validate "<sql>"` (alias `dt check`) — bind/prepare only, **executes
  nothing**; prints errors with line/col (exit non-zero on failure). Reads whole
  buffer or `-f file.sql`.
- `dt explain "<sql>" [--analyze]` — `EXPLAIN`; `--analyze` gated behind explicit
  opt-in (it executes). Surface partition pruning / estimated cost.
- `dt describe '<file-or-glob>'` — schema of ad-hoc Parquet/CSV/JSON/Iceberg via
  `DESCRIBE SELECT * FROM '<path>'` without a full scan.
- `dt cache {refresh|show|clear}` — manage the persisted cache;
  `show [--json]` prints it; `refresh` forces rebuild; `clear` deletes it.

### Acceptance
- `dt validate` on a query referencing a missing column exits non-zero with a
  useful message and does not scan data.
- `dt context --json --connections lake` returns only the lake catalog, valid
  JSON, from cache when fingerprint matches (no remote round-trip).
- `dt cache refresh` rebuilds and updates fingerprint/timestamp.

---

## 8. Track 3 — MCP server (`dt mcp`)

**Depends on:** shared core (§4, §5). **Library:** a Go MCP library — default
`github.com/ThinkInAIXYZ/go-mcp` (the "go-mcp" named in scoping); confirm vs.
`mark3labs/mcp-go` during A-track spike.

### Functional requirements
- `dt mcp` starts an MCP **stdio** server; loads `.ducktape.yaml` from cwd/root;
  opens the federated engine read-only; shares the schema cache with the other
  surfaces.
- Tools (gated by `mcp.tools` in config):
  - `list_schemas` — catalogs/schemas/tables (cache-backed, scopable).
  - `describe_table` / `describe_file` — columns+types for a table or a file glob.
  - `validate_sql` — prepare-only; returns ok/errors with positions. No execution.
  - `explain_sql` — plan/cost.
  - `run_query` — execute with a `max_rows` cap and `allow_write:false` default;
    returns rows as JSON. Off unless enabled.
- Never emit secrets in tool output or logs.
- **Client docs:** how to register `dt mcp` in Claude Code / `claude_desktop_config`
  (command `dt`, args `["mcp"]`), and how `.ducktape.yaml` scopes it.

### Acceptance
- A coding agent can call `list_schemas` → `validate_sql` → `run_query` to
  produce a report over an attached lake, with `validate_sql` catching a bad
  column before any scan, and `run_query` honoring `max_rows`.

---

## 9. Cross-cutting requirements

- **Secrets:** env interpolation only; never log/serialize connection strings;
  redact in errors.
- **Build/CI:** `go-duckdb` CGO already in the release matrix; adding
  `glsp`/`go-mcp` is pure-Go. Track 1 Phase B (`go-tree-sitter`) adds CGO — keep
  it optional/behind a build consideration so the core stays buildable without it.
- **Caching correctness:** fingerprint = hash(sorted in-scope connection set +
  `dt.db` mtime + extension set). Stale-cache is the sharpest failure mode —
  background refresh + explicit `dt cache refresh` + fingerprint mismatch rebuild.
- **Backward compatibility:** absent `.ducktape.yaml` → current global-config
  behavior unchanged. New commands are additive.
- **Testing:** unit tests for splitter, error-position parser, fingerprint,
  config discovery/interpolation; integration tests spinning a DuckDB with an
  attached SQLite + a Parquet fixture covering validate/explain/describe/cache.

---

## 10. Suggested epic → issue breakdown

Dependency order: **A (foundation) → B (CLI) → {C (LSP), D (MCP)} in parallel.**

### Epic A — Shared foundation
- **A1** `project` package: `.ducktape.yaml` discovery (walk-up), parse,
  `${ENV}` interpolation, `extends` merge with global config, schema validation.
- **A2** `WithReadOnly` on `DatabaseClient` (`access_mode=read_only`; `:memory:`
  forced RW); SQLite/MySQL attach WAL/busy_timeout options; fix
  `ConnectionConfigForm` `EnableWrite` bug.
- **A3** `schema` package: move introspection out of `cmd/context.go`; define
  `Cache` model (tables/columns/functions/extensions).
- **A4** Cache persistence: JSON load/save, fingerprint, invalidation rules,
  project-local `.ducktape/cache.json`.
- **A5** `engine.Validate`: prepare-only validation + DuckDB error→position
  parser (regex suite + whole-statement fallback).
- **A6** `engine.Explain`: `EXPLAIN` / `EXPLAIN ANALYZE` (opt-in) wrappers.

### Epic B — CLI surface (Track 2)
- **B1** Refactor `dt context` onto cache; add `--json`, `--connections`,
  `--tables` scoping (backward-compatible default).
- **B2** `dt validate` / `dt check` command.
- **B3** `dt explain` command (`--analyze` gated).
- **B4** `dt describe '<file/glob>'` command.
- **B5** `dt cache {refresh|show|clear}` commands.

### Epic C — Developer LSP (Track 1)
- **C1** `dt lsp` scaffold: glsp stdio, initialize/shutdown, capabilities,
  `.ducktape.yaml` binding.
- **C2** Document sync + statement splitter (`lsp/splitter.go`).
- **C3** Diagnostics via `engine.Validate` → `publishDiagnostics`.
- **C4** Completion Phase A (clause heuristics, alias resolution).
- **C5** Hover + signature help.
- **C6** `didChangeConfiguration` runtime reconfig + `duckdb_extensions()` watcher.
- **C7** *(optional)* Completion Phase B: tree-sitter CST + relevance scoring.
- **C8** Editor integration docs (`docs/editors/`: nvim, helix, vscode).

### Epic D — MCP server (Track 3)
- **D1** `dt mcp` scaffold: go-mcp stdio server + `.ducktape.yaml` load +
  read-only engine.
- **D2** Tools: `list_schemas`, `describe_table`, `describe_file` (cache-backed).
- **D3** Tools: `validate_sql`, `explain_sql`.
- **D4** Tool: `run_query` (max_rows cap, `allow_write:false` default).
- **D5** `mcp:` config block + client setup docs (Claude Code / desktop).

### Epic E — Cross-cutting
- **E1** Secrets handling + redaction pass across all surfaces.
- **E2** Test fixtures + integration harness (DuckDB + SQLite + Parquet).
- **E3** CI/build review (CGO matrix; tree-sitter optionality).
- **E4** Top-level docs: README section + `.ducktape.yaml` reference.

---

## 11. Open decisions
- MCP library: `ThinkInAIXYZ/go-mcp` vs `mark3labs/mcp-go` (spike in A/D1).
- LSP completion: ship Phase A only for v1, defer tree-sitter (C7)?
- Cache location: always project-local `.ducktape/`, or fall back to
  `~/.dt/<workspace>/` when no project file exists? (Proposed: project-local when
  `.ducktape.yaml` present, else global.)
- `run_query` in MCP: ship in v1 or start read/validate-only until safety caps
  are proven?
- Should `.ducktape.yaml` fully replace global workspaces long-term, or coexist?

---

## 12. Milestones
- **M1 (Foundation + CLI):** Epics A + B → validation/cache/describe usable from
  the terminal and by agents via bash. Highest leverage; unblocks everything.
- **M2 (Agent-native):** Epic D → `dt mcp` for coding-agent workflows.
- **M3 (Editor):** Epic C → `dt lsp` for human SQL authoring.
- **M4 (Polish):** Epic E + LSP Phase B.
