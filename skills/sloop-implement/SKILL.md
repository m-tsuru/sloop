---
name: sloop-implement
description: Implements a READY or FORCEREADY Sloop specification in the current repository. Use for requests such as "/sloop implement sloop-1", "implement sloop-1 with Sloop", or equivalent instructions. Reviews READY specifications rather than guessing externally observable behavior and maintains feature bindings for symbols it creates.
metadata:
  version: "0.1"
  compatibility: Requires the sloop CLI on PATH and a Sloop-initialized Git repository on macOS or Linux.
---

# Sloop Implement

Implement exactly one user-selected Sloop specification. Treat Sloop CLI JSON as the authority for specification content, revision identity, and feature bindings.

## Preconditions

Before editing:

1. Require an explicit specification selector. Never select a similar or merely likely ID.
2. Confirm `sloop` is on `PATH` and the current directory is the intended Git repository and Sloop project. Otherwise stop without repository changes.
3. Run `sloop context <specification> --json`. Do not parse human output or access Sloop's database, object store, or `.sloop` internals.
4. Require a recorded `READY` or `FORCEREADY` context. Never mark a specification READY/FORCEREADY.
5. Retain the full `revision_hash` as the Base Specification Revision.
6. Capture `git status --short` and, when available, `git rev-parse HEAD`. Preserve all pre-existing changes.

Use the current specification, explicit references, resolved feature bindings, and only repository context needed to implement it. Inspect explicit references and bindings first, then direct dependencies, related tests, and narrowly necessary surrounding code. Do not preload all specifications, revisions, documentation, or source files.

## Review

For READY, review before editing whether authoritative context determines every required externally observable behavior. Reject only when implementation needs an unspecified decision that affects visible behavior and neither explicit references nor established public behavior/tests determine it.

Do not reject open private implementation details. Repository convention alone is not a new product requirement.

Record a successful READY review:

```console
sloop accept <specification> --author.name codex --author.agent true
```

For ambiguity, make no changes and record:

```console
sloop reject <specification> \
  --reason "<missing decision, why it is needed, and affected behavior>" \
  --author.name codex --author.agent true
```

Then stop as REJECTED without changing status.

For FORCEREADY, never reject for ambiguity. Prefer specification consistency, preservation of public behavior, existing tests, then the smallest non-destructive change. Record material assumptions.

## Feature bindings

The context `features` array describes factual links between each `func:<feature-id>` and existing implementation/test symbols. Empty `impls` or `tests` only means no binding is currently recorded; it is not itself a requirement to create code or tests.

Use existing RESOLVED bindings as high-priority navigation context. Do not infer replacements for MISSING locators or automatically follow renames.

Whenever this run creates a new implementation or test symbol for a feature present in the context, register it after the symbol exists:

```console
sloop bind add <specification> <feature-id> --impl <path:function-or-method> \
  --author.name codex --author.agent true

sloop bind add <specification> <feature-id> --test <path:test-function-or-method> \
  --author.name codex --author.agent true
```

Use `path:function` for functions and `path:type:method` for methods. Register every newly created corresponding symbol before IMPLEMENTED or VERIFIED. Never register a future or unresolved symbol, and never treat a blank binding as permission to invent a requirement.

`bind add` is the only permitted specification metadata mutation in this workflow. Do not run `sloop edit`, change Markdown, parents, references, or user-controlled statuses.

Binding changes can create a recorded DRAFT revision. After the final binding update, reload `sloop context <specification> --json` and confirm:

- title, Markdown sections, parents, and prior bindings still match the Base Specification Revision;
- the only expected specification change is the factual addition of symbols created by this run;
- every non-empty binding is RESOLVED.

Treat that refreshed full hash as the Implementation Specification Revision. Any other change is stale and must stop status recording.

## Implement and verify

Make the smallest coherent change satisfying the Base Specification Revision. Preserve unrelated work and avoid repository-wide destructive cleanup or Git history rewriting.

Add or update tests when required by the specification. Use the existing test framework when none is prescribed. Check every acceptance criterion with relevant tests, builds, type checks, static analysis, or direct behavioral inspection; formatting or lint alone is not behavioral evidence.

Fix implementation bugs and rerun checks. For a newly discovered READY ambiguity, stop, record REJECTED, leave partial changes intact, and report them. For FORCEREADY, apply the conservative rule. Environmental failures are BLOCKED/ERROR, not specification rejection.

## Stale check and status

Immediately before IMPLEMENTED, reload JSON context. Its hash must equal:

- the Base Specification Revision if no bindings changed; or
- the verified Implementation Specification Revision produced only by this run's binding updates.

Otherwise leave the patch in place but record no status transition.

Record IMPLEMENTED only after the final patch and all required binding updates exist:

```console
sloop status <specification> implemented \
  --reason "<implemented behavior, changed areas, tests, bindings, and assumptions>" \
  --author.name codex --author.agent true
```

Refresh context after this transition and use its new full hash. Record VERIFIED only when every acceptance criterion was actually verified:

```console
sloop status <specification> verified \
  --reason "<checks run and acceptance-criteria evidence>" \
  --author.name codex --author.agent true
```

If implementation is complete but verification is blocked, leave status IMPLEMENTED. Only a human may set COMPLETED.

## Git and completion

Do not commit unless explicitly requested. If requested, do not include unrelated user changes and associate the commit with `sloop cr <specification> <commit-hash>`.

Report specification, result (`VERIFIED`, `IMPLEMENTED`, `REJECTED`, `BLOCKED`, or `ERROR`), final Sloop status, changed files/behavior, verification evidence, assumptions, and any partial changes. Never present partial work as complete.
