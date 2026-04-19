# ADR-005: Exempt ProtoConjec units from H19Criterial duplicate-killer

**Date:** 2026-04-19
**Status:** Accepted
**Phase:** 7.5 follow-up
**Commits:** 55d405e

## Context

A 300-cycle shakeout after landing Phase 7.5 showed H19Criterial killing 840 of the 4402 ProtoConjec units H-Conjecture produced — false positives. Credit halvings against H-Conjecture piled up at 915, roughly matching, as the credit system punished it for making "duplicate" units.

Root cause: ProtoConjec units store their *distinguishing* information (conjectureAbout, statement, evidence) in non-criterial slots by design. Their *criterial* slots (isA, worth, conjecKind, creditors) are identical across the whole category. H19Criterial's "all criterial slots match some peer" check therefore matches every pair of ProtoConjec units trivially.

Three ways out:

1. Make `conjectureAbout` criterial globally.
2. Add some other criterial slot unique to each ProtoConjec.
3. Add ProtoConjec to H19Criterial's existing exemption list (which already skips H-Specialize / H18-Generalize output for analogous reasons).

## Decision

Option 3. Add `"ArgU" @ "ProtoConjec" isa?` to the skip list.

## Alternatives considered

- **Make `conjectureAbout` criterial** — broad effect on any future consumer that reads criterial slots. Also changes the meaning of "criterial" for conjectures specifically: criterial means "defines identity", but ConjectureAbout is more like "what the conjecture points at" — it belongs to the content, not the signature. Rejected as semantic drift.
- **Add a synthetic criterial key per ProtoConjec** — redundant with the naming scheme (ADR-001) which already dedupes by `(kind, sorted-about)`. Would double-bookkeep.
- **Change dedupe to happen at creation, not at H19Criterial** — already done via `make-protoconjec`'s `unit-exists?` check. H19Criterial kills were happening *after* successful dedupe, on genuinely distinct conjectures with the same criterial-slot signature.

## Consequences

- ProtoConjec kills dropped from 840 to 0 in the next 300-cycle run. Total kills dropped from 882 to 21. Credit halvings dropped from 915 to 21.
- Any future unit category that stores identity in non-criterial slots will need the same exemption. The pattern is: if the category's "what makes this unit distinct from a peer in the same isA bucket" lives outside the criterial slot set, H19Criterial will over-kill. Either rethink the slot assignment or add to the skip list. The skip list is stringly-typed (isA-match against specific categories); this is fragile but pragmatic.
- The observation also surfaced that our credit economy is load-bearing on H19Criterial's accuracy. A better design would have the credit system not punish creators whose output gets legitimately reclassified after-the-fact as a duplicate. Out of scope for this ADR.
