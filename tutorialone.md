# Tutorial One: Building a Saved-Query Subsystem for DuckTape

This tutorial walks through designing and building a new subsystem for `dt`
that lets you **save a query under a name** and **rerun it later**, including
the connections it was originally run with. By the end you'll understand:

- How to model the domain type (`SavedQuery`) idiomatically
- How to build a **repository** package with a small interface and a
  filesystem-backed implementation
- Where saved queries should live on disk (inside the workspace folder)
- What **switch code** is needed in the `cmd` package: flag changes,
  new Cobra subcommands, and a refactor of the shared execution path
- How to test each layer with `testify`, matching the existing test style

We will *not* paste a finished implementation top-to-bottom. Instead, each
part explains the decision being made, shows the code that decision produces,
and points at the existing DuckTape code it needs to integrate with.

---

## Part 0 — What we're building (the UX contract)

Before writing any Go, pin down the command-line surface. Everything else
falls out of this:

```bash
# Run a query and save it under a name (note: --save takes a NAME now)
dt q "select * from pg.users where created_at > now() - interval 7 day" \
    -c pg --save weekly_users

# Rerun it later — same SQL, same connections, automatically
dt q run weekly_users

# See what you've saved in this workspace
dt q list

# Inspect one
dt q show weekly_users

# Delete one
dt q rm weekly_users
```

Two requirements drive the whole design:

1. **Queries are saved *per workspace*.** A query saved while `-w dev` is
   active must not show up under `-w prod`. DuckTape already scopes the
   DuckDB database and connections to a workspace folder
   (`~/.dt/<workspace>/` — see `config.WorkspacePath` in
   `config/config.go`), so saved queries belong in that same folder.
2. **A saved query remembers its connection IDs.** When you ran it with
   `-c pg -c stripe`, rerunning it must re-attach `pg` and `stripe` without
   you retyping them. The connection *configs* stay where they already live
   (viper config, managed by the `workspace` package) — we only store the
   *names*, and resolve them at run time exactly like `dt q` does today via
   `WithConnectionsByName`.

### A note on the word "repository"

The request calls for "a new repository" in the design-pattern sense: a type
that owns persistence for one kind of entity, behind an interface, so the
CLI layer never touches the filesystem directly. That's what we'll build.

---

## Part 1 — Design decisions (read this before typing)

### Where does the data live?

Three realistic options:

| Option | Pros | Cons |
| --- | --- | --- |
| Viper config (`config.yaml`), like connections | Consistent with connections | Multi-line SQL in one shared YAML file gets ugly fast; every save rewrites global config; merge conflicts between workspaces |
| A table inside the workspace's DuckDB file (`dt.db`) | Queryable | Saved queries become invisible to `git`, editors, and `grep`; harder to hand-edit |
| **One YAML file per query in `~/.dt/<workspace>/queries/`** | Human-readable, editable, diffable, naturally workspace-scoped, trivial `list` | A new persistence path to write |

We pick the third. The workspace folder already exists and is created by
`config.EnsureWorkspace` — we're extending an existing convention, not
inventing one. Layout:

```
~/.dt/
├── config.yaml            # global config + connections (unchanged)
└── dev/                   # workspace folder
    ├── dt.db              # workspace DuckDB database (unchanged)
    └── queries/           # NEW: one file per saved query
        ├── weekly_users.yaml
        └── churn_report.yaml
```

### What goes in a saved query file?

```yaml
name: weekly_users
sql: |
  select * from pg.users
  where created_at > now() - interval 7 day
connections:
  - pg
workspace: dev
created_at: 2026-08-05T14:03:22Z
updated_at: 2026-08-05T14:03:22Z
```

Store the **connection names**, not the connection configs. Configs can
contain credentials and already have a home; duplicating them here would
create two sources of truth and scatter secrets across more files.

### Package layout

DuckTape uses flat, single-purpose packages at the module root:
`connection/`, `workspace/`, `config/`, `cmd/`. We follow suit:

```
savedquery/
├── savedquery.go        # the SavedQuery domain type
├── repository.go        # Repository interface + filesystem implementation
└── repository_test.go
```

Why not put it in `workspace/`? Because `workspace/` is a thin wrapper over
viper, and this package is a filesystem store. Different persistence
mechanism, different package. `cmd/` will import both.

> **Go idiom checkpoint:** one package = one concern, named after what it
> *provides* (`savedquery`), not what it *is* (`models`, `utils`, `repo`).

---

## Part 2 — The domain type (`savedquery/savedquery.go`)

Start with the entity. Keep it a plain struct with YAML tags, mirroring how
`connection.ConnectionConfig` is defined in `connection/connection.go`:

```go
/*
Copyright © 2026 Zac Orndorff zac@orndorff.dev
*/
package savedquery

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// SavedQuery is a named SQL query bound to the workspace it was saved in
// and the connection names it was executed with.
type SavedQuery struct {
	Name        string    `yaml:"name"`
	SQL         string    `yaml:"sql"`
	Connections []string  `yaml:"connections"`
	Workspace   string    `yaml:"workspace"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
}
```

### Validating names

The name becomes a filename *and* a CLI argument, so constrain it. A
package-level `validName` regexp and a `Validate` method keep the rule in
one place:

```go
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

var ErrInvalidName = errors.New("query name must start with a letter or digit and contain only letters, digits, '-' and '_'")

func (q SavedQuery) Validate() error {
	if !validName.MatchString(q.Name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, q.Name)
	}
	if q.SQL == "" {
		return errors.New("query SQL must not be empty")
	}
	return nil
}
```

Why so strict? Three reasons:

1. It's a filename — no path separators, no `..`, no leading dots. This is
   also your defense against path traversal (`dt q rm ../../config` must
   be impossible).
2. It will be typed as a bare CLI arg — no shell metacharacters.
3. It must never collide with the subcommand names we're about to add
   (`run`, `list`, `show`, `rm`). The regexp doesn't prevent that by
   itself — Part 5 explains why it doesn't need to.

> **Go idiom checkpoint:** `ErrInvalidName` is a *sentinel error* wrapped
> with `%w`. Callers can test `errors.Is(err, savedquery.ErrInvalidName)`
> without string matching. We'll use the same trick for "not found".

---

## Part 3 — The repository (`savedquery/repository.go`)

### The interface

Small. Only what the CLI needs, nothing speculative:

```go
// Repository stores and retrieves saved queries for a single workspace.
type Repository interface {
	Save(q SavedQuery) error
	Get(name string) (SavedQuery, error)
	List() ([]SavedQuery, error)
	Delete(name string) error
}

var ErrNotFound = errors.New("saved query not found")
```

> **Go idiom checkpoint:** *"Accept interfaces, return structs."* The
> interface exists so `cmd` code and tests can depend on the behavior, but
> the constructor below returns the concrete `*FSRepository`. Don't return
> the interface from the constructor — that hides methods and makes
> extension awkward.

Note the interface is **per-workspace**: you construct a repository *for* a
workspace, rather than passing `workspace` into every method. That mirrors
how the rest of the codebase treats the active workspace as ambient context
(it's resolved once from viper at the top of each command's `Run`).

### The filesystem implementation

```go
// FSRepository stores each query as a YAML file inside a directory,
// conventionally ~/.dt/<workspace>/queries.
type FSRepository struct {
	dir       string
	workspace string
}

func NewFSRepository(workspace string) (*FSRepository, error) {
	dir := filepath.Join(config.WorkspacePath(workspace), "queries")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating queries dir: %w", err)
	}
	return &FSRepository{dir: dir, workspace: workspace}, nil
}
```

Two things to notice:

- We reuse `config.WorkspacePath` from `config/config.go` rather than
  rebuilding the `~/.dt/<workspace>` path — one source of truth for layout.
  (Check the import graph first: `config` imports `workspace`, and neither
  imports `savedquery`, so `savedquery → config` introduces no cycle.)
- `MkdirAll` in the constructor means every method can assume the directory
  exists. Constructors that return `(T, error)` are normal Go; don't panic
  and don't lazily create the dir in four different methods.

For tests you'll want a second constructor that skips the home-directory
convention:

```go
// NewFSRepositoryAt is like NewFSRepository but roots the store at an
// explicit directory. Used by tests and available for future flags.
func NewFSRepositoryAt(dir, workspace string) (*FSRepository, error) { ... }
```

### Save: marshal + atomic write

```go
func (r *FSRepository) path(name string) string {
	return filepath.Join(r.dir, name+".yaml")
}

func (r *FSRepository) Save(q SavedQuery) error {
	if err := q.Validate(); err != nil {
		return err
	}
	q.Workspace = r.workspace

	now := time.Now().UTC()
	if existing, err := r.Get(q.Name); err == nil {
		q.CreatedAt = existing.CreatedAt // overwriting: keep original creation time
	} else {
		q.CreatedAt = now
	}
	q.UpdatedAt = now

	data, err := yaml.Marshal(q)
	if err != nil {
		return fmt.Errorf("marshaling query %q: %w", q.Name, err)
	}

	// Write to a temp file in the same directory, then rename. Rename is
	// atomic on POSIX filesystems, so a crash mid-save never leaves a
	// half-written query file.
	tmp, err := os.CreateTemp(r.dir, q.Name+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), r.path(q.Name))
}
```

Use `gopkg.in/yaml.v3` for marshaling — it's already an indirect dependency
(check `go.sum`), so `go mod tidy` just promotes it to direct in `go.mod`.

### Get: translate `os.ErrNotExist` into your domain error

```go
func (r *FSRepository) Get(name string) (SavedQuery, error) {
	if !validName.MatchString(name) {
		return SavedQuery{}, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	data, err := os.ReadFile(r.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return SavedQuery{}, fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	if err != nil {
		return SavedQuery{}, err
	}
	var q SavedQuery
	if err := yaml.Unmarshal(data, &q); err != nil {
		return SavedQuery{}, fmt.Errorf("parsing %s: %w", r.path(name), err)
	}
	return q, nil
}
```

The name check at the top of `Get` (and `Delete`) is the security-relevant
line: it runs *before* the name is ever joined into a path.

> **Go idiom checkpoint:** callers should never see `os.ErrNotExist` leak
> out of a repository — that's an implementation detail. Translate it at
> the boundary. The `cmd` layer can then do
> `if errors.Is(err, savedquery.ErrNotFound)` and print a friendly message
> plus a `dt q list` hint.

### List and Delete

`List` is a `filepath.Glob(filepath.Join(r.dir, "*.yaml"))` (or
`os.ReadDir` + suffix filter), unmarshal each file, skip-and-`slog.Warn` on
corrupt entries rather than failing the whole listing, and sort by name so
output is deterministic. `Delete` validates the name, calls `os.Remove`,
and maps `os.ErrNotExist` to `ErrNotFound` the same way `Get` does.

---

## Part 4 — Testing the repository

Match the house style: external test package (`savedquery_test`), `testify`
assertions, table tests where they pay off. `t.TempDir()` gives you a real
filesystem that cleans itself up — no mocks needed for a filesystem repo:

```go
package savedquery_test

import (
	"errors"
	"testing"

	"github.com/SandwichLabs/duck-tape/savedquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndGetRoundTrip(t *testing.T) {
	repo, err := savedquery.NewFSRepositoryAt(t.TempDir(), "test_workspace")
	require.NoError(t, err)

	in := savedquery.SavedQuery{
		Name:        "weekly_users",
		SQL:         "select * from pg.users",
		Connections: []string{"pg"},
	}
	require.NoError(t, repo.Save(in))

	out, err := repo.Get("weekly_users")
	require.NoError(t, err)
	assert.Equal(t, in.SQL, out.SQL)
	assert.Equal(t, in.Connections, out.Connections)
	assert.Equal(t, "test_workspace", out.Workspace)
	assert.False(t, out.CreatedAt.IsZero())
}

func TestGetMissingReturnsErrNotFound(t *testing.T) {
	repo, err := savedquery.NewFSRepositoryAt(t.TempDir(), "test_workspace")
	require.NoError(t, err)

	_, err = repo.Get("nope")
	assert.True(t, errors.Is(err, savedquery.ErrNotFound))
}
```

Cases worth a table test:

- invalid names: `"../evil"`, `".hidden"`, `"has space"`, `""`, `"a/b"`
- overwrite preserves `CreatedAt`, bumps `UpdatedAt`
- `List` on an empty dir returns an empty slice (not an error, not `nil`
  if you promised otherwise — decide and test it)
- `Delete` then `Get` returns `ErrNotFound`

Run them the same way CI does: `task test` (see `Taskfile.yml`).

---

## Part 5 — The switch code: wiring into `cmd`

Now the part you asked about specifically — what has to change in the
Cobra layer. There are four pieces.

### 5.1 Change `--save` from a bool to a string

`cmd/query.go` already has a placeholder:

```go
// current code in init() — a bool that does nothing yet
queryCmd.Flags().BoolP("save", "s", false, "Save the query to the ducktape folder")
```

Replace it:

```go
queryCmd.Flags().StringP("save", "s", "", "Save this query under a name for later reuse (dt q run <name>)")
```

This is exactly the UX you sketched: `dt q "select ..." --save weekly_users`.
A string flag whose empty default means "don't save" is a common Cobra
pattern and avoids a second `--name` flag.

### 5.2 Extract the execution path so two commands can share it

Today, `queryCmd.Run` in `cmd/query.go` does everything inline: read
viper, build the `DatabaseClient` with the functional options from
`cmd/database.go`, prepare, execute, scan rows, print JSON lines. The
`run` subcommand needs the identical pipeline — so extract it *first*,
before adding features. Add to `cmd/query.go` (or a new `cmd/execute.go`):

```go
// executeQuery runs sql against the given workspace with the given
// connections attached, streaming JSON rows to out.
func executeQuery(out io.Writer, workspaceName, sqlText string, connectionNames, params []string) error {
	dbPath := viper.GetString(fmt.Sprintf("%s.dbLocation", workspaceName))

	client := NewDatabaseClient(
		WithNumThreads(4),
		WithWorkspace(workspaceName),
		WithDatabasePath(dbPath),
		WithConnectionsByName(connectionNames),
		InitDatabaseClient(),
	)
	// ... the existing OpenConnection / PrepareContext / scan loop,
	// writing each row with fmt.Fprintln(out, valueString)
	return nil
}
```

Then `queryCmd.Run` shrinks to: read flags → `executeQuery(...)` → maybe
save. Returning an `error` (instead of `cobra.CheckErr` inside the helper)
keeps the helper testable; the command's `Run` remains the only place that
calls `cobra.CheckErr`.

**Ordering decision:** save *after* a successful execution. If the SQL is
broken, you'll be glad it wasn't persisted:

```go
Run: func(cmd *cobra.Command, args []string) {
	workspaceName := viper.GetString("workspace")
	sqlText := args[0]
	connectionNames, _ := cmd.Flags().GetStringArray("connections")
	params, _ := cmd.Flags().GetStringArray("param")

	err := executeQuery(cmd.OutOrStdout(), workspaceName, sqlText, connectionNames, params)
	cobra.CheckErr(err)

	if saveName, _ := cmd.Flags().GetString("save"); saveName != "" {
		repo, err := savedquery.NewFSRepository(workspaceName)
		cobra.CheckErr(err)
		err = repo.Save(savedquery.SavedQuery{
			Name:        saveName,
			SQL:         sqlText,
			Connections: connectionNames,
		})
		cobra.CheckErr(err)
		slog.Info("query saved", "name", saveName, "workspace", workspaceName)
	}
},
```

That `slog.Info` matches how `setConnectionCmd` reports saves in
`cmd/connections.go`.

### 5.3 Add the subcommands: `run`, `list`, `show`, `rm`

Create `cmd/query_saved.go`. Attach subcommands to `queryCmd`, mirroring
how `cmd/set.go` composes `setCmd.AddCommand(setConnectionCmd)`:

```go
var queryRunCmd = &cobra.Command{
	Use:     "run [name]",
	Aliases: []string{"r"},
	Short:   "Rerun a saved query with its saved connections",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workspaceName := viper.GetString("workspace")

		repo, err := savedquery.NewFSRepository(workspaceName)
		cobra.CheckErr(err)

		saved, err := repo.Get(args[0])
		cobra.CheckErr(err)

		// Saved connections are the default; -c on the command line overrides.
		connectionNames := saved.Connections
		if override, _ := cmd.Flags().GetStringArray("connections"); len(override) > 0 {
			connectionNames = override
		}
		params, _ := cmd.Flags().GetStringArray("param")

		err = executeQuery(cmd.OutOrStdout(), workspaceName, saved.SQL, connectionNames, params)
		cobra.CheckErr(err)
	},
}

func init() {
	queryCmd.AddCommand(queryRunCmd, queryListCmd, queryShowCmd, queryRmCmd)
	queryRunCmd.Flags().StringArrayP("connections", "c", []string{}, "Override the saved connections")
	queryRunCmd.Flags().StringArrayP("param", "p", []string{}, "Parameters to pass to the query")
}
```

The remaining three are small:

- **`list`** (alias `ls`): `repo.List()`, then either print names line by
  line, or reuse `ToJsonString` from `cmd/format.go` to emit one JSON
  object per row (`{"name": ..., "connections": ..., "updated_at": ...}`) —
  JSON-lines output is DuckTape's house style and stays `jq`-friendly, per
  the pipeline examples in `cmd/root.go`'s long help.
- **`show`**: `repo.Get(name)`, print the SQL (and metadata to stderr, or
  behind a `--json` flag, so `dt q show x | pbcopy` gives you clean SQL).
- **`rm`** (alias `delete`): `repo.Delete(name)`, report via `slog.Info`.

### 5.4 Understand the dispatch subtlety you just created

`queryCmd` now has **both** a `Run` function taking one positional arg
(the raw SQL) **and** subcommands. How does Cobra decide what
`dt q run weekly_users` means?

Cobra matches the first positional argument against registered subcommand
names *before* falling back to the parent's own `Run`. So:

- `dt q "select 1"` → `"select 1"` matches no subcommand → `queryCmd.Run`
  executes it as SQL. Unchanged behavior. ✔
- `dt q run weekly_users` → `run` matches the subcommand → rerun path. ✔
- The edge case: `dt q run` — someone trying to execute the literal SQL
  string `run` (nobody), or more plausibly `dt q list` where the user
  wanted to query a table named `list`. Subcommands win. This is why the
  subcommand set is small and made of words that aren't plausible SQL
  statements, and why real SQL (`select …`, `create …`) always contains a
  space and thus can never collide with a subcommand name.

Also note `Args: cobra.ExactArgs(1)` on `queryCmd` still applies only when
no subcommand matched, so the validation story is unchanged.

There's one more integration detail: saved-query names can't shadow
subcommands *at the `dt q <name>` level* because rerunning always goes
through `dt q run <name>` — the name is an argument to `run`, never a
subcommand itself. That's the reason to prefer `dt q run foo` over the
tempting `dt q foo`: the latter turns every saved name into a potential
command and makes `dt q "select 1"` ambiguous forever. Namespacing via a
sub-verb is the standard Cobra answer.

---

## Part 6 — Testing the command layer

Follow the pattern in `cmd/database_test.go` (external `cmd_test` package,
`testify`). Two layers are worth covering:

1. **`executeQuery` directly** — point it at a temp workspace like
   `TestOpen` does (`WithDatabasePath("test.db")`), run
   `select 1+1 as answer`, and assert on the writer's contents. This is
   why it takes an `io.Writer` instead of printing to stdout.
2. **Flag/dispatch behavior** — Cobra commands can be executed in-process:

```go
func TestSaveThenRun(t *testing.T) {
	// Arrange: temp HOME so ~/.dt lands in the sandbox, viper workspace set.
	t.Setenv("HOME", t.TempDir())
	viper.Set("workspace", "test_workspace")
	cmd.InitConfig()

	out := new(bytes.Buffer)
	root := cmd.RootCmd() // you'll need a small accessor or use rootCmd via an exported Execute variant
	root.SetOut(out)
	root.SetArgs([]string{"q", "select 1+1 as answer", "--save", "answer"})
	require.NoError(t, root.Execute())

	out.Reset()
	root.SetArgs([]string{"q", "run", "answer"})
	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), `"answer":2`)
}
```

You'll hit a real-world snag here: `rootCmd` is unexported and flags
persist between `Execute` calls in one process. This is a known Cobra
testing friction — the usual fixes are a small exported test hook, or
constructing fresh command instances per test. Don't restructure the whole
`cmd` package for it in this tutorial; a `//nolint` shortcut or a minimal
exported accessor is fine, and a "command factory" refactor is a good
future exercise.

Also remember `Taskfile.yml` sets `CGO_ENABLED: '1'` — anything that
touches `go-duckdb` (i.e., any test through `executeQuery`) needs cgo, so
run via `task test`.

---

## Part 7 — Ship checklist

- [ ] `savedquery/savedquery.go` — struct, `Validate`, sentinel errors
- [ ] `savedquery/repository.go` — `Repository` interface, `FSRepository`,
      both constructors, atomic `Save`, error translation in `Get`/`Delete`
- [ ] `savedquery/repository_test.go` — round-trip, not-found, invalid
      names, overwrite semantics, empty list
- [ ] `cmd/query.go` — `--save` bool → string; `Run` slimmed to
      flags → `executeQuery` → conditional save
- [ ] `cmd/query_saved.go` — `run`/`list`/`show`/`rm` subcommands wired
      into `queryCmd` in `init()`
- [ ] `go.mod` — `gopkg.in/yaml.v3` promoted to a direct dependency
      (`go mod tidy`)
- [ ] `task lint && task test && task build` all green
- [ ] `Readme.md` — a "Saved queries" section with the Part 0 examples

## Part 8 — Exercises (future extensions)

1. **Saved parameters.** Persist `-p` values (`Params []string`) and let
   `dt q run` replay them, with CLI params overriding — same
   default/override dance as connections.
2. **IDs and aliases.** Add a short content hash (`ID string`) so renames
   don't orphan shell scripts: `dt q run 3f2a91`. Decide: is the hash over
   SQL only, or SQL + connections?
3. **`--from-file` and stdin.** `dt q --save big_report -f ./report.sql`
   and `cat report.sql | dt q --save big_report -` for queries too gnarly
   for a shell argument.
4. **Export/import.** `dt q export > queries.tar` for moving between
   workspaces — and think about what `Workspace` field semantics should be
   on import.
5. **The factory refactor.** Convert `cmd` package globals into
   `NewRootCmd() *cobra.Command` construction to make Part 6's tests
   first-class. This is the biggest structural improvement available to
   the codebase and deserves its own tutorial.
