---
name: feature-tutorial
description: Produce a codebase-grounded tutorial that teaches the developer how to implement a feature themselves, instead of implementing it for them. Use whenever the user asks for a tutorial, walkthrough, or step-by-step guide for building a feature or subsystem; says things like "don't implement this — teach me", "walk me through how I would build X", "write a tutorial for adding X", or "explain what modules I'd need to create"; or wants to keep engineering context with the team by writing the code themselves. Trigger even when the word "tutorial" isn't used but the user clearly wants to learn-by-building rather than receive finished code.
---

# Feature Tutorial

Write a tutorial document that teaches a developer how to implement a specific
feature in *their* codebase — and write **no implementation code into the
repository yourself**.

## Why this mode exists

When you implement a feature, the design decisions live in your context window
and evaporate when the session ends. When the developer implements it from a
good tutorial, the decisions live in their head — and the tutorial stays in the
repo as a record of *why* the code is shaped the way it is. The user has
explicitly chosen that trade: slightly slower delivery for durable
understanding. Respect it. The tutorial file is the entire deliverable.

Success looks like: the developer can implement the feature without asking you
anything, understands why each decision was made, and could defend those
decisions in code review.

## Step 1 — Research before writing a word

A generic tutorial ("first, define your struct...") is worthless; the reader
could get that from any blog post. The entire value of this document is that it
is grounded in *this* repository. Before drafting, read enough of the codebase
to know:

- **The closest existing feature.** Something similar almost always exists
  (another CLI command, another storage layer, another endpoint). Its
  structure is the template the new code should mirror, and the tutorial
  should say so by file path.
- **House conventions.** Package layout, naming, error handling style, logging
  library, how configuration is read, comment/copyright headers.
- **Test style.** Which framework, internal vs. external test packages, how
  fixtures are managed, how tests are run (Makefile/Taskfile/CI config —
  check for environment requirements like cgo or docker).
- **Existing hooks for this feature.** Search for TODOs, dead flags, stub
  functions, or commented-out code related to the request. Finding a dormant
  placeholder ("there's already a `--save` flag that does nothing") makes the
  tutorial dramatically more grounded — the reader starts from something real.
- **Dependency reality.** If the design needs a library, check the manifest
  and lockfile first. "This is already an indirect dependency; `go mod tidy`
  promotes it" is a much better instruction than "add this dependency".
- **Structural hazards.** Check the things you'd check if implementing:
  import cycles, unexported symbols the tests will need, dispatch or routing
  ambiguities the new surface introduces.

Verify everything you plan to reference. Every file path, function name, flag,
and line-number claim in the tutorial must come from actually reading the code
this session — a single hallucinated reference destroys the reader's trust in
the whole document.

## Step 2 — Make the design decisions for real

Do the same design thinking you would do before implementing: pick the storage
format, the package layout, the interface shape, the CLI/API surface. Then
*teach* the decision instead of just announcing it:

- Present the 2–3 realistic options (a small comparison table works well),
  with honest pros and cons.
- State which one the tutorial builds and the reason it wins *in this
  codebase* — usually "it extends an existing convention" beats abstract
  superiority.
- A tutorial that hedges ("you could do A or B, up to you!") on load-bearing
  decisions teaches nothing and leaves the reader stuck. Decide. Genuinely
  optional choices can be flagged as such.

## Step 3 — Write the tutorial

### Structure

Adapt this skeleton rather than following it mechanically — it's the shape
that has worked, not a form to fill in:

```markdown
# Tutorial: <feature>
Brief: what the reader will build and what they'll understand afterward.

## Part 0 — The contract
The user-visible surface first: CLI invocations, API calls, or UI flow,
written as if the feature exists. Then the 1–3 requirements that drive
the whole design. Everything else in the tutorial should visibly fall
out of this section.

## Part 1 — Design decisions
The options tables and choices from Step 2, including where new code
lives (package/module layout) and where data lives, referencing the
repo's existing layout.

## Parts 2..N — One part per module, inside-out
Domain types first, then the core mechanism (storage/service/logic),
then integration. Each part: the decision being made, the code that
decision produces, and pointers to the existing code it mirrors.

## The wiring part
The integration changes to *existing* files — exact file paths, the
current code being changed (quoted), and what it becomes. Call out
dispatch/routing subtleties the new surface creates. This is usually
the part the reader can't figure out alone; be most precise here.

## Testing
Concrete test cases in the repo's own test style, plus the command that
runs them. List the edge cases worth a table test rather than writing
every test out.

## Ship checklist
Checkbox list of every file to create/modify, docs to update, and the
lint/test/build commands that must pass.

## Exercises / future extensions
2–5 follow-ups that deepen understanding or extend the feature, with a
sentence on the design question each one raises.
```

### Show code — but don't ship the feature in prose

Include real code for the parts that carry decisions or subtlety: the domain
type, the tricky method, the integration diff, the security-relevant line.
Elide the mechanical parts with a sentence describing what they do ("`List` is
a glob + unmarshal loop; skip corrupt files with a warning rather than failing
the listing"). Two reasons: a tutorial the reader completes purely by
copy-paste transfers nothing, and elided sections are where the reader's own
understanding gets built. State explicitly near the top that the tutorial is
not a paste-ready implementation.

### Teach idioms at the moment they appear

When the code demonstrates a language or framework idiom — sentinel errors,
accept-interfaces-return-structs, atomic file writes, constructor patterns —
add a short labeled callout (e.g. **"Go idiom checkpoint"**) explaining the
idiom and why it applies. These callouts carry the durable learning; the
feature is partly a vehicle for them.

### Call out the snags they will actually hit

From your Step 1 research, you know the real obstacles: the unexported
variable that blocks testing, the cgo requirement, the flag that shadows a
subcommand, the config file that gets rewritten on save. Name them, explain
them, and either give the fix or explicitly scope it out as a future
refactor. Predicting a snag the reader then hits is the moment the tutorial
earns its keep. Where a decision has security consequences (path traversal,
injection, secrets in files), show which specific line is the defense and
what it defends against.

## Hard rules

- **Write only the tutorial file.** No source files created or modified, no
  scaffolding, no dependency changes, no "small starter commit". If the user
  asks you to also implement, that's a separate request outside this skill —
  confirm before switching modes.
- **File placement:** use the user's requested name/location verbatim. If
  they didn't specify, follow an existing convention in the repo (existing
  tutorials, a `docs/` dir); otherwise default to `docs/tutorials/<feature-slug>.md`.
- **Depth follows the feature.** A small feature gets a short tutorial. Don't
  pad with generic language explanations the reader didn't sign up for —
  calibrate to what the codebase suggests about the team's experience.

## After writing

Summarize for the user: the parts of the tutorial, the key design decisions it
commits to, and anything surprising you found in the codebase while
researching (dead code, existing placeholders, hazards). This lets them judge
whether to read it now and whether the design direction matches their intent —
the summary is the review hook for the document.
