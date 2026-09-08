---
name: sloop-implement
description: Implements a READY or FORCEREADY Sloop specification in the current repository. Use when the user asks Codex to implement a Sloop specification, including requests such as "/sloop implement sloop-1", "implement sloop-1 with Sloop", or equivalent instructions. Reviews READY specifications for missing externally observable behavior and rejects ambiguity instead of guessing.
metadata:
  version: "0.1"
  compatibility: Requires the sloop CLI on PATH and a Sloop-initialized Git repository on macOS or Linux.
---

# Sloop Implement

Implement exactly one Sloop specification against the current repository, verify its acceptance criteria, and record the outcome through the Sloop CLI. Sloop is the authority for specification content and history.

## Required input

Identify the specification selector explicitly supplied by the user, such as `1`, `sloop-1`, or `project-12`. Never guess a missing selector or silently substitute a similar ID. If it cannot be determined uniquely, ask for it and do not begin implementation.

Treat `/sloop implement <specification>` and equivalent natural-language requests as the same workflow.

## Preconditions and context

Before changing the repository:

1. Confirm `sloop` is available on `PATH`. If it is absent, stop without modifying the repository and report `BLOCKED`.
2. Confirm the current directory belongs to the intended Git repository. Use the current working directory and repository only; do not search unrelated directories for another Sloop project.
3. Run `sloop context <specification> --json`. Require its specification ID, persistent `specification_uuid`, full `revision_hash`, status, reserved sections, references, Git relations, and review policy. Use JSON only; do not parse human-readable tables, ANSI output, the SQLite database, the object store, or `.sloop` internals.
4. Require a recorded context whose status is `READY` or `FORCEREADY`. Do not implement `DRAFT`, `IMPLEMENTED`, `VERIFIED`, `CANCELED`, or `COMPLETED`. Never mark a specification READY or FORCEREADY yourself.
5. Retain the returned full `revision_hash` as the Base Specification Revision. Base review and implementation decisions on that immutable revision.
6. Capture `git status --short` and, when available, `git rev-parse HEAD` before editing. A dirty worktree is allowed. Treat every pre-existing change as user-owned.

The Sloop context is the primary input. It includes the specification ID and revision, status, goal, implementation specification, acceptance criteria, explicit references, Git relations, sections, and review policy.

Use only the current specification, its explicit references, and repository context that becomes necessary. Inspect in this order:

1. Explicit references.
2. Their direct dependencies.
3. Related tests.
4. Other narrowly necessary repository context.

Do not preload all specifications, revisions, documentation, or source files. Repository instructions already supplied by the coding environment still apply; do not copy `AGENTS.md` into the Sloop context. If an applicable repository instruction conflicts with the Sloop specification, report the conflict and stop instead of inventing precedence.

## Review before editing

For a `READY` revision, decide whether its authoritative context determines every externally observable behavior needed for implementation. Review at least CLI/API behavior, visible output, errors and exit codes, persistence and data-loss semantics, compatibility, defaults, authorization, timeouts, retries, public schemas, ordering, and destructive behavior when they are relevant.

Reject only when all of these are true:

1. Implementation requires a decision.
2. That decision changes externally observable behavior.
3. The specification gives no basis for it.
4. Explicit references and established public behavior or tests do not determine it uniquely.

Do not reject merely because the specification is short or because an internal implementation choice is open. Private names, helper boundaries, equivalent internal data structures and algorithms, private package layout, and test-helper structure are ordinary implementation choices.

Existing code and tests may establish authoritative compatibility behavior. Repository convention alone must not be promoted into a new product requirement.

If the revision is reviewable, record acceptance before editing:

```console
sloop accept <specification> --author.name codex --author.agent true
```

If a `READY` revision is ambiguous, make no repository changes. Record the rejection:

```console
sloop reject <specification> \
  --reason "<missing decision, why it is required, and affected observable behavior>" \
  --author.name codex \
  --author.agent true
```

Then stop with `REJECTED`, state the required user decision, and do not change the specification status.

For `FORCEREADY`, do not reject for specification ambiguity. Continue using this conservative order:

1. Behavior most consistent with the specification.
2. Preservation of existing public behavior.
3. Behavior consistent with existing tests.
4. The smallest, least destructive change.

Track material assumptions so they can be included in the implementation transition reason and final response.

## Implement

Make the smallest coherent repository change that satisfies the Base Specification Revision. Preserve unrelated changes and integrate carefully with pre-existing edits in touched files. Never use repository-wide destructive cleanup, discard changes that you did not create, or rewrite Git history.

Do not change the Sloop specification or its metadata. In particular, do not run `sloop edit`, alter specification Markdown, change parents, add or remove references, or set READY/FORCEREADY. Do not access Sloop's database or object store directly.

Prioritize tests required by the acceptance criteria. Reuse the repository's existing test framework when no framework is specified. A requirement to add tests is mandatory.

If a new externally observable ambiguity appears during implementation:

- For `READY`, stop. Record a REJECTED review against the current revision when Sloop still permits it. Do not record IMPLEMENTED or VERIFIED. Do not revert partial changes; report `Partial changes: present` and list the modified files.
- For `FORCEREADY`, continue under the conservative decision rule and record the assumption.

## Verify

Check every acceptance criterion individually. Use relevant automated tests, builds, type checking, linting, static analysis, or direct behavioral inspection. Formatting or lint success alone is not evidence that behavioral criteria pass.

When a check fails:

- Fix an implementation bug and rerun the relevant checks.
- Treat newly exposed ambiguity according to the READY/FORCEREADY rules above.
- Classify unavailable dependencies, missing tools, permission failures, and unrelated pre-existing test failures as environmental failures, not specification ambiguity.

If the implementation is complete but verification cannot finish for an environmental reason, it may be recorded as IMPLEMENTED but must not be recorded as VERIFIED.

## Detect a stale specification

Immediately before recording IMPLEMENTED, run `sloop context <specification> --json` again. Continue only if its full `revision_hash` still equals the Base Specification Revision and the recorded content/status has not changed. A context failure caused by new working changes is also stale.

If it is stale, do not record a status transition and do not assume the patch matches the new revision. Leave the implementation changes in place and report the base hash and the need to review the new revision.

## Record status

After the final patch exists and the stale check passes, record IMPLEMENTED with a concrete summary:

```console
sloop status <specification> implemented \
  --reason "<implemented behavior, main changed areas, tests, and material FORCEREADY assumptions>" \
  --author.name codex \
  --author.agent true
```

Do not use a reason such as `done`, `implemented`, or `finished`.

Status transitions create revisions. After IMPLEMENTED, refresh with `sloop context <specification> --json` and use the returned new full hash. Never reuse the Base Specification Revision for the VERIFIED transition.

Only after every acceptance criterion has been verified, record VERIFIED:

```console
sloop status <specification> verified \
  --reason "<checks run and evidence for the acceptance criteria>" \
  --author.name codex \
  --author.agent true
```

Leave the specification at IMPLEMENTED when verification is incomplete. Only a human may make the final COMPLETED decision.

## Git rules

Do not create a commit unless the user explicitly asked for one. If asked, create it without rewriting existing history, then associate it through:

```console
sloop cr <specification> <commit-hash>
```

Never use a commit to hide or absorb unrelated user changes.

## Failure handling

Classify the result as:

- `VERIFIED`: implementation and all acceptance-criteria checks completed.
- `IMPLEMENTED`: implementation completed, but verification could not be completed.
- `REJECTED`: a READY revision lacks a required externally observable decision.
- `BLOCKED`: the environment or repository state prevents safe progress.
- `ERROR`: the Sloop CLI or this workflow fails unexpectedly.

On CLI, environment, or repository failure, do not mislabel the result as a specification rejection. Preserve existing and partial work.

## Final response

Always report the specification, result, final Sloop status, implementation summary, verification evidence, and any partial changes or environmental limitations.

For REJECTED, state the missing decision, why it affects implementation, the observable choices the user must resolve, and whether partial changes exist. Never describe partial work as completed implementation.
