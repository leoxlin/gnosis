---
type: Procedure
title: refining-procedure
description: Use when an existing Procedure record must be exhaustively traced, clarified with its author, and rewritten for reliable AI execution. Use only when the author asks to refine a procedure not just update.
tags: [gnosis, vault]
invocation: model
---

# refining-procedure

`refining-procedure` turns one existing Procedure into an evidence-backed, author-confirmed execution contract. Repository behavior comes from evidence; the author owns intent.

## STEP 1 - mapping-procedure

### Knowledge inputs

- The author's request, an exact or resolvable target Procedure identity, and any resumable refinement checkpoint.
- The effective Procedure record, URI, revision, origin, owning vault, ConceptType definition, repository rules, configuration, and history.
- Relevant invokers, downstream procedures, referenced knowledge, implementation, schemas, tests, call sites, and authoritative technical sources.

### Process

1. Resolve the target to exactly one existing effective record and read it with `gnosis read '<procedure URI>' --json`. Require type `Procedure`; bind its URI, revision, origin, and owning vault. If identity is ambiguous, ask one identity question with a recommended resolution. Stop when the record does not exist or has the wrong type.
2. Modify a local record only in its owning writable vault. For an imported or bundled record whose owner is unavailable, report its origin and stop; never shadow it into the current vault under a different URI.
3. Read the target, the effective Procedure ConceptType, governing rules, and configuration completely. On resume, recheck every bound revision and source before trusting the checkpoint.
4. Trace all evidence that constrains execution: inbound invokers, downstream handoffs, linked and pseudo-linked knowledge, implementation, tests, relevant call sites, recent path-scoped history, and external references when the workflow depends on them. Inspect unlinked paths only when evidence shows that they constrain the target.
5. Parallelize only independent, read-only fact discovery. One controller owns the evidence ledger, control-flow map, author dialogue, and every mutation decision. The controller verifies agent findings instead of treating reports as proof.
6. Bind every material fact to a knowledge URI and revision, repository path and current revision, or exact command or observation. Separate established facts, author-owned decisions, unresolved facts, contradictions, and assumptions. Research discoverable facts instead of asking the author to guess them.
7. Build a provisional control-flow and coverage map containing:
   - selection conditions, exclusions, actors, authority, and preconditions;
   - every input's source, format, freshness, validation, default, and use;
   - ordered actions, dependencies, permissions, state changes, and side effects;
   - every predicate, default or else outcome, nested branch, loop, retry bound, timeout, and cancellation path;
   - success, no-op, partial, unavailable-dependency, external, timing-dependent, failure, recovery, rollback, escalation, and blocked paths;
   - outputs, owners, handoffs, stop states, and observable completion evidence;
   - inbound and outbound contracts with every integration.
8. Derive materially possible implicit branches even when the target omits them. Mark a category inapplicable only with evidence. Visualize the provisional map for the author with unresolved, resolved, invalidated, and evidence-backed nodes. Prefer a flowchart when supported and otherwise use an indented decision tree. Keep the visualization and ledger transient.

### Completion

One existing Procedure is bound to an exact URI, revision, and origin. Its relevant evidence and integrations are source-bound, every explicit and materially implicit path appears in a provisional visual map, unresolved nodes are classified, and no repository record has changed.

## STEP 2 - resolving-contract

### Knowledge inputs

- The bound target and source revisions, provisional control-flow map, evidence ledger, and integration contracts.
- Resolved, unresolved, and invalidated author decisions and any confirmed scope expansion.
- The effective Procedure schema and repository-specific validation requirements.

### Process

1. Walk each branch end-to-end before moving to the next, resolving upstream conditions before dependent actions and outcomes.
2. Ask exactly one author-owned decision at a time. Identify the current branch and relevant evidence, give one recommended answer with rationale, and mention alternatives only when they materially change the outcome. Wait for an explicit answer or explicit delegation. Silence, ambiguity, or inferred preference does not resolve the node; clarify the same node before advancing.
3. When evidence conflicts with the target, its callers, tests, implementation, or governing knowledge, present the conflict and recommended resolution to the author. Correct purely mechanical schema violations from authoritative evidence without presenting them as intent decisions.
4. When a material fact remains unknown, retain the evidence gap and investigation history. Do not finalize unless the author supplies an authoritative source or approves behavior that remains safe regardless of the unknown fact.
5. After every answer, update the decision ledger and affected paths. If an answer changes an upstream assumption or topology, preserve the history, mark every dependent conclusion invalidated, reopen only affected paths, and redraw the map. Refresh the visualization after every completed branch and whenever topology changes.
6. Keep mutation scope limited to the selected Procedure by default. Report caller, dependency, implementation, and knowledge impacts without editing them. If the proposed change would make executable knowledge materially inconsistent, obtain explicit expanded scope and include every required repair in the confirmed change set or stop without writing.
7. Expand scope to a ConceptType only when removing procedural prose from that ConceptType would leave a target branch incomplete or non-executable. Propose moving operational sequencing into the Procedure while retaining declarative schema, lifecycle invariants, and record semantics in the ConceptType. Treat the coupled records as one confirmed change set. Ordinary schema or invariant references remain read-only.
8. Permit author-confirmed semantic improvements and conversion between valid single-step and multi-step shapes. Label every behavioral change against the bound revision and trace its effects. Preserve the target URI and title unless a separately approved rename or move includes every inbound-link repair.
9. When the target contains independently invocable workflows, propose a split only with explicit scope expansion, exact new records, and link repairs. When it duplicates or conflicts with another Procedure, propose one authoritative workflow but do not modify, replace, or remove either record without explicit multi-record approval.
10. Draft the exact proposed content for every in-scope record. Make selection conditions, inputs, ordered actions, predicates, loop bounds, permitted mutations, side effects, handoffs, stop states, and completion evidence explicit enough for reliable AI execution. Keep rules at their proper authority, include only execution-relevant rationale, preserve unknown metadata and unrelated valid content, and remove only text made obsolete or contradictory by the confirmed contract.
11. Dry-run the proposed instructions against at least one representative scenario for every terminal path, including failure and stop paths. Require each scenario to select its branch from stated evidence and reach one determinate outcome.
12. Audit that every input is validated and consumed, every predicate has exhaustive outcomes, every loop is bounded, every side effect has authority and ownership, every path reaches a handoff or stop state, and every completion claim has observable evidence. Reject unreachable, duplicate, circular, orphaned, contradictory, or evidence-free logic.
13. When another agent is available, obtain an independent read-only completeness review and verify every finding. A reviewer may identify missing decisions but cannot settle author intent. When no reviewer is available, perform the same adversarial self-review and disclose that fact.
14. Present the final visualization, complete proposed content for every record, semantic changes, integration impacts, evidence gaps, scenario results, and the validation and rollback plan. Require explicit approval of the exact change set before writing. Reopen affected nodes after any requested change.
15. If the interview pauses, return a resumable checkpoint containing bound revisions, the current map, source-bound evidence, resolved and invalidated decisions, remaining nodes, and the next question. Do not modify a repository record.

### Completion

Every material fact is established or rendered irrelevant by safe behavior; every author-owned choice is explicitly resolved; every path passes simulation and completeness review; no known integration contradiction remains outside the authorized change set; and the author has approved the exact proposed record contents. No repository mutation has occurred yet.

## STEP 3 - writing-and-verifying

### Knowledge inputs

- The exact approved record contents, semantic-change and integration-impact summaries, and validation plan.
- Bound target, dependency, and source revisions; local origins and paths; and pre-change content snapshots.
- Repository-supported write, indexing, validation, and directly governing check commands.

### Process

1. Immediately before mutation, reread every in-scope URI and recheck every evidence source. Preserve and report non-semantic metadata drift. If behavioral, lifecycle, integration, or topology drift affects the approved contract, return only the affected paths to `resolving-contract` and obtain fresh approval.
2. Preflight the complete change set against the effective schema, destination paths, link resolution, ownership, mutation permissions, and integration contracts. Snapshot every existing in-scope record. For a removal, bind the exact local origin and verify that all inbound references are repaired or intentionally removed.
3. Treat an approved multi-record refinement as one indivisible change set. Apply creates and updates through `gnosis write '<URI>' --filename <approved-file>` in the owning local vault. Perform a deletion only when its exact local path and removal were explicitly approved.
4. Read every written URI back with `gnosis read '<URI>' --json`; bind its new revision and compare the content with the approved contract plus any authorized non-semantic metadata preservation.
5. Run `gnosis index --vault <root>` when indexing is enabled, then run `gnosis validate --vault <root>` and every repository check that directly governs the changed records. Inspect complete output, warnings, exit status, and the final diff.
6. Repair a mechanical failure only when the correction preserves the approved semantics exactly, then repeat read-back and all validation. Before any semantic correction, safely restore the complete pre-change set and return affected paths to `resolving-contract`.
7. On a partial write or failed rollback, restore only content written by this invocation and only while its current revision still matches the invocation's write. Never overwrite external drift. Preserve the observed state, report the exact partial-state blocker, and stop.
8. Return the final URIs and revisions, changed and removed paths, semantic changes, integration impacts, scenario results, complete evidence and decision ledgers, validation commands and results, reviewer mode and findings, and any remaining nonblocking uncertainty.

### Completion

Every in-scope record matches the author-approved contract at a reported revision, the change set is internally and externally consistent, indexing is current when enabled, vault and governing checks pass, and no failure or uncertainty is represented as success.
