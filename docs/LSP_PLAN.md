# Duck Tape LSP — Implementation Plan

**Status:** Proposed
**Author:** design draft
**Scope:** Add a schema-aware SQL Language Server to `dt`, driven by the
workspace's DuckDB database and its attached (cached) connections.

---

## 1. Goal

Give any LSP-capable editor (Neovim, VS Code, Helix, Zed, Emacs) real-time SQL
intelligence — completion, hover, and diagnostics — for the **exact** set of
databases a Duck Tape workspace has attached: the local workspace `dt.db` plus
every `ATTACH`ed connection (Postgres, MySQL, SQLite, HTTPFS/S3, Parquet/CSV/JSON
replacement scans).

The differentiator is that Duck Tape already federates every engine behind a
single embedded DuckDB instance. The language server does not need a
per-dialect adapter — it introspects and validates against **one** DuckDB
catalog that already contains every attached schema. The remaining work is
LSP plumbing, a persisted **schema cache**, and safe (read-only) connection
handling.

---

## 2. Why this differs from the reference (Rust) plan

The uploaded `DuckDB_Neovim_LSP_Plan` describes a greenfield **Rust /
`tower-lsp`** server with a `DatabaseAdapter` trait to reach Postgres, MySQL,
etc. That plan is sound in isolation, but Duck Tape is a different starting
point, and three of its assumptions invert here:

| Reference plan assumption | Duck Tape reality | Consequence |
|---|---|---|
| New standalone Rust binary | `dt` is a Go/Cobra binary with `go-duckdb` (CGO) already linked | Ship the LSP as `dt lsp` (a subcommand), not a separate binary. No new toolchain, no CGO cross-compile pain — the engine is already in-process. |
| Per-engine `DatabaseAdapter` (a Postgres adapter, a MySQL adapter…) | DuckDB `ATTACH` already federates Postgres/MySQL/SQLite/files into one catalog | **No adapters needed.** Introspect + validate against the single DuckDB instance; every attached engine shows up as another `catalog.schema.table`. This is a large simplification. |
| Must design read-only locking from scratch | `connection.ConnectionConfig.ReadWriteMode()` already appends `, READ_ONLY` unless `EnableWrite` | Attachment safety is mostly solved; we only need to also open the **workspace `dt.db` itself** read-only in LSP mode. |
| `duckdb_prepare` via raw C API | Go `database/sql` `PrepareContext` on `go-duckdb` | Same validation strategy, idiomatic Go — DuckDB's binder/catalog checks the query, so DuckDB "Friendly SQL" (`GROUP BY ALL`, `SELECT * EXCLUDE`, `FROM 'file.parquet'`, trailing commas) validates correctly with zero false positives. |

We keep the reference plan's genuinely portable ideas: **read-only by default**,
the **hybrid parser** (error-tolerant CST for half-typed input + real engine
`prepare` for complete statements), the **lazy, thread-safe schema cache**, and
the **extension-lifecycle watcher**.

**Language decision:** stay in Go. Rewriting in Rust would fork the engine
integration, config, and connection logic that already exist. Go startup is
fine for a long-lived stdio server (the ~150–300 ms Node.js concern in the
reference does not apply to a compiled Go binary that starts once and stays
resident).

---

## 3. Architecture overview

```
  Editor (Neovim / VS Code / Helix …)
        │  JSON-RPC 2.0 over stdio
        ▼
  dt lsp                         ← new Cobra subcommand
        │
        ▼
  lsp.Server (glsp handler)
   ├── Document store            ← didOpen/didChange text sync
   ├── Diagnostics engine        ← splitter → PrepareContext → parse error → Diagnostic
   ├── Completion / Hover        ← cursor context → SchemaCache lookup
   └── schema.Cache  ◄───────────┐
        │  reads                 │ refresh (lazy + on DDL + background)
        ▼                        │
  DatabaseClient (READ_ONLY)  ───┘
   ├── workspace dt.db  (access_mode=read_only)
   └── ATTACH <conn> ... (, READ_ONLY)   ← Postgres / MySQL / SQLite / HTTPFS
```

The `DatabaseClient` (`cmd/database.go`) and workspace/connection resolution
(`workspace`, `connection` packages) are reused verbatim, with one new
read-only option (§5.2).

---

## 4. The schema cache (the "cached schemas / dbs" core)

This is the heart of the feature and the user's explicit ask.

### 4.1 What is cached

For the workspace DB **and every attached connection**, cache:

- **Catalogs / schemas** — `information_schema.schemata`
- **Tables & views** — `SHOW ALL TABLES` (already used by `dt context`) →
  `database, schema, name, column_names, column_types, temporary`
- **Columns** — from the same `SHOW ALL TABLES` payload (names + types), with
  `information_schema.columns` as the fallback for nullability/defaults
- **Functions / macros / UDFs** — `duckdb_functions()`
- **Loaded extensions** — `duckdb_extensions()` (`extension_name, loaded, installed`)

### 4.2 In-memory model

```go
package schema

type Column struct { Name, DataType string; Nullable bool }
type Table  struct { Catalog, Schema, Name, Type string; Columns []Column }
type Func   struct { Name string; Params []string; ParamTypes []string; Return string }

type Cache struct {
    mu         sync.RWMutex
    Workspace  string
    Tables     map[string]Table   // key: catalog.schema.name (lower-cased)
    Functions  map[string]Func
    Extensions map[string]bool    // name -> loaded
    BuiltAt    time.Time
    Fingerprint string            // hash of attached connection set
}
```

The DuckDB root connection is opened once; per-request introspection uses
short-lived `database/sql` connections from the pooled `*sql.DB` (DuckDB is
multi-reader-safe within one process), matching reference recommendation #3.

### 4.3 Persistence — the "cached" differentiator

Introspecting a **remote** Postgres/MySQL over the network on every editor
start is slow. Duck Tape already snapshots schemas conceptually (`dt context`);
we make it durable:

- Serialize `Cache` to `~/.dt/<workspace>/.schema_cache.json` (gitignored).
- On `dt lsp` startup: if a cache file exists **and** its `Fingerprint`
  (hash of the sorted attached-connection set + DB mtime) matches the current
  workspace, load it instantly and serve completions in **<10 ms** with **zero**
  remote round-trips. Kick off a background refresh to reconcile drift.
- If the fingerprint differs (a connection was added/removed, or `dt.db`
  changed), rebuild.
- Expose it as a first-class command so it's useful outside the editor too:
  - `dt lsp cache refresh` — force a full rebuild now.
  - `dt lsp cache show` — print the cached schema (JSON/markdown).
  - `dt context` is refactored to **read this same cache**, so the LLM-context
    command and the LSP share one introspection path (see §7 DRY note).

### 4.4 Invalidation & refresh

- **Lazy build:** first `completion`/`hover` request triggers the build if the
  in-memory cache is empty (keeps startup instant).
- **DDL-aware:** when the editor buffer executes/contains a completed
  `CREATE`/`DROP`/`ALTER`/`ATTACH`/`DETACH` (detected in the diagnostics pass),
  mark tables dirty and re-introspect the affected catalog.
- **Extension watcher:** a background goroutine polls `duckdb_extensions()` on an
  interval; when an extension flips `loaded=false → true` (e.g. `spatial`, `fts`,
  `httpfs` autoloaded on first use), fetch its new `duckdb_functions()` rows and
  merge them so completion/signature help work without hardcoding.
- **Config change:** `workspace/didChangeConfiguration` (switching workspace or
  DB target at runtime) closes connections, clears the cache, reconnects.

---

## 5. Connection & locking strategy

### 5.1 Attachments — already safe

Attached connections go through the existing boot query in
`cmd/database.go:115`, which uses `attachment.ReadWriteMode()` →
`, READ_ONLY` by default. The LSP inherits this untouched. For attached
SQLite/MySQL we additionally pass the reference plan's anti-starvation options
inside `ATTACH` (`META_JOURNAL_MODE 'WAL'`, `META_BUSY_TIMEOUT 500`) so the
server never blocks an external pipeline.

### 5.2 The workspace `dt.db` — one change required

Today `InitDatabaseClient` opens the workspace DB read-write
(`<path>?threads=N`). A running `dt query` or app holding a read-write handle to
the same file would lock-conflict with the LSP. Fix:

- Add `WithReadOnly(bool)` to `DatabaseClient` and, in LSP mode, build the
  connString as `<path>?threads=N&access_mode=read_only`.
- `:memory:` targets cannot be read-only → force read-write (they are
  process-isolated, so no lock risk), mirroring the reference plan's in-memory
  rule.

| Target | LSP open mode | Lock risk |
|---|---|---|
| Workspace `dt.db` (file) | `access_mode=read_only` | none (multi-reader) |
| `:memory:` | read-write | none (process-local) |
| Attached Postgres/MySQL/SQLite | `, READ_ONLY` (+ WAL/busy_timeout) | none |

---

## 6. Parser & diagnostics (hybrid model)

Mirror the reference plan, adapted to Go:

1. **Statement splitter** — a small scope-tracking tokenizer (parens, quotes,
   `$$` blocks, `;`) isolates the statement under the cursor so one malformed
   query doesn't poison diagnostics for the rest of the buffer.
2. **Complete statement → validate on the real engine.** Run
   `db.PrepareContext(ctx, stmt)` on the read-only DuckDB connection (never
   execute — prepare only). DuckDB's binder resolves catalog/columns and accepts
   Friendly SQL, so diagnostics are true positives.
3. **Prepare error → LSP Diagnostic.** Parse `go-duckdb`'s error string for
   line/column. DuckDB emits `LINE n:` and a caret (`^`) offset; extract them
   with a regex (e.g. the reference's
   `Parser\s+Error:.*?at or near "…" LINE (\d+)`), plus binder-error variants
   ("Catalog Error", "Binder Error"). Map to a `protocol.Diagnostic` range and
   `publishDiagnostics`.
4. **Incomplete statement → don't error; complete instead.** While typing,
   `prepare` will fail continuously — that's expected and must **not** surface as
   a diagnostic. For these we need only cursor context for completion (§6.1).

### 6.1 Completion context — phased

- **Phase A (MVP, no tree-sitter):** lightweight lexer + clause heuristics.
  Detect whether the cursor is after `FROM`/`JOIN` (→ suggest tables), inside a
  `SELECT`/`WHERE`/`ON` projection (→ columns), or after `alias.` (→ that table's
  columns). Resolve aliases by scanning the statement's `FROM … AS x` bindings.
  This covers ~80% of value with no new dependency.
- **Phase B (richer):** add `tree-sitter-sql` via `github.com/smacker/go-tree-sitter`
  for an error-tolerant CST, enabling precise node-context scoring
  (`projection_list` vs `from_clause` vs `where_clause`) and the reference plan's
  relevance-weighted ranking (alias-matched columns boosted to the top).

### 6.2 Hover & signature help

- Hover over a table → columns + types from the cache; over a column → its type
  and owning table; over a function → signature from `duckdb_functions()`.
- Signature help inside `func(` → parameter list from the cached `Func`.

---

## 7. Code layout & reuse

New package `lsp/` plus a shared `schema/` package extracted from existing code:

```
cmd/lsp.go                 # `dt lsp` + `dt lsp cache {refresh,show}` Cobra commands
lsp/server.go              # glsp handler wiring, initialize/shutdown, capabilities
lsp/document.go            # text document sync + in-memory buffer store
lsp/diagnostics.go         # splitter → prepare → error-parse → publishDiagnostics
lsp/completion.go          # completion provider (Phase A heuristics, later CST)
lsp/hover.go               # hover + signatureHelp
lsp/splitter.go            # statement splitter / scope tracker
schema/cache.go            # Cache type, persistence (load/save/fingerprint)
schema/introspect.go       # introspection queries (moved from cmd/context.go)
```

**DRY refactor:** `getSchemaMarkdown`, `getTableNames`, and the `SHOW ALL TABLES`
logic currently living in `cmd/context.go` move into `schema/introspect.go`.
Both `dt context` and `dt lsp` then consume one introspection + cache path.
(This also lets `dt context` benefit from the persisted cache for instant LLM
context on slow remote DBs.)

**Client library:** recommend `github.com/tliron/glsp` (batteries-included
Go LSP framework: JSON-RPC, stdio transport, `protocol_3_16` types, handler
struct). Alternative: the lower-level `go.lsp.dev/protocol` + `jsonrpc2`.
Recommendation: **glsp** for velocity; it maps cleanly onto Cobra's `Run`.

**Advertised capabilities (initialize):** `textDocumentSync: Incremental`,
`completionProvider` (triggers `.`, ` `), `hoverProvider`, `diagnosticProvider`
(or push via `publishDiagnostics`), `signatureHelpProvider` (trigger `(`, `,`).

---

## 8. Configuration

Extend the workspace config block (viper, `~/.dt/config.yaml`) with an optional
`lsp` section — no breaking changes:

```yaml
dev:
  dbLocation: /home/me/.dt/dev/dt.db
  connections: { … existing … }
  lsp:
    read_only: true                     # default true; :memory: forced false
    auto_load_extensions: [parquet, httpfs, json]
    cache_ttl_seconds: 300              # background refresh cadence
    connections: [my_pg, my_sqlite]     # which attachments to expose (default: all)
```

Runtime switching (the reference plan's `DuckDbSwitchTarget`) is handled via
`workspace/didChangeConfiguration`: the server tears down connections, clears
the cache, and rebinds to the new workspace/DB.

---

## 9. Editor integration

Because it's a standard stdio LSP, any client works by spawning `dt lsp`.

- **Neovim** (0.11+ `vim.lsp.config`): register `cmd = { "dt", "lsp" }`,
  `filetypes = { "sql" }`, `root_dir` markers `{ ".git", "config.yaml" }`, plus a
  `:DtSwitchWorkspace` user command that sends `didChangeConfiguration`. (Adapt
  the Lua from the reference plan; swap the binary for `dt lsp` and drop the
  client-side `.sqllsrc` parsing since Duck Tape already owns config discovery.)
- **VS Code / Helix / Zed:** document the `dt lsp` command + `sql` filetype
  mapping. A thin VS Code extension is a possible follow-up, not required.

Ship a `docs/editors/` folder with copy-paste snippets for nvim + Helix.

---

## 10. Phased roadmap

| Phase | Deliverable | Reuses / adds |
|---|---|---|
| **0. Scaffolding** | `dt lsp` starts a glsp stdio server, completes `initialize`/`shutdown`, advertises sync capability. | new `cmd/lsp.go`, `lsp/server.go`, glsp dep |
| **1. Read-only engine binding** | LSP opens workspace `dt.db` read-only + attachments; `WithReadOnly` option added. | `cmd/database.go` (+`WithReadOnly`), existing connection logic |
| **2. Diagnostics loop** | Splitter + `PrepareContext` validation + error-position parsing → `publishDiagnostics`. | `lsp/splitter.go`, `lsp/diagnostics.go` |
| **3. Schema cache + completion/hover** | `schema.Cache` (in-mem + persisted JSON + fingerprint), Phase-A completion, hover. `dt context` refactored onto shared cache. | new `schema/` pkg, refactor `cmd/context.go` |
| **4. Extension lifecycle + CST** | `duckdb_extensions()` watcher; tree-sitter CST for precise context + relevance scoring; signature help. | background goroutine, `go-tree-sitter` |
| **5. Polish** | Config `lsp:` block, `didChangeConfiguration` runtime switching, editor docs (nvim/Helix), tests. | config, `docs/editors/` |

MVP = Phases 0–3 (validation + schema-aware completion against cached,
read-only attached DBs). Phases 4–5 are enhancement.

---

## 11. Risks & open questions

- **CGO / static binary:** `go-duckdb` is CGO; `dt` already ships this way via
  goreleaser, so `dt lsp` adds no new build constraint. Confirm release matrix
  still builds with the new packages.
- **Remote introspection latency & auth:** first cache build against a large
  remote Postgres can be slow and needs live credentials. Persisted cache
  mitigates repeat cost; document that `dt lsp cache refresh` requires the
  connection to be reachable.
- **`SHOW ALL TABLES` scale:** on very large catalogs, page/scope introspection
  to the connections listed in `lsp.connections` rather than all attachments.
- **Diagnostics precision:** DuckDB error strings vary by error class; the
  line/column parser needs a small regex suite + a fallback that highlights the
  whole statement when position can't be extracted.
- **Incomplete-statement noise:** ensure prepare-failures on the actively-typed
  statement are suppressed (only complete statements produce diagnostics).
- **Existing bug to fold in:** `connection.ConnectionConfigForm()` doesn't set
  `EnableWrite` (noted in `PRODUCTION_READINESS.md`); since the LSP relies on
  `ReadWriteMode()`, fix it as part of Phase 1.

---

## 12. Summary

Duck Tape is unusually well-positioned for this feature: the embedded DuckDB
engine already federates every supported database into one catalog, connections
already default to read-only, and schema introspection already exists in
`dt context`. The plan therefore ships the language server **inside** `dt` as
`dt lsp`, reuses the connection/engine layer, and centers on a **persisted,
fingerprinted schema cache** so the editor gets instant, schema-aware SQL
intelligence for the workspace's cached databases — without ever locking or
re-hammering a remote engine.
