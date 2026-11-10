---
name: interview
description:
  How to interview the user. One question per message in plain prose, research and a
  recommendation before every question, keystone-first ordering, build-it-together
  handling of intuition answers, and persistence of locked decisions. Use whenever
  asking the user a series of questions to lock decisions, gather requirements, clarify
  ambiguous scope, or resolve design or implementation choices.
---

# Interviewing the User

An interview is a series of checkpoints on the session's reasoning, not a delegation of
it. The session researches, forms a recommendation, and asks the user to confirm or
redirect, one decision at a time. The verdict on hard calls stays with the user; the
session's job is to feed that judgment the best possible evidence, never to substitute
for it.

## 🚨 THE INTERVIEW RULE 🚨

ASK QUESTIONS ONE BY ONE, IN PLAIN PROSE, AND WAIT FOR A PLAIN TEXT RESPONSE. NEVER USE
AskUserQuestion OR ANY SELECT-STYLE WIDGET. NEVER BUNDLE MULTIPLE QUESTIONS INTO ONE
MESSAGE. ONE DECISION PER MESSAGE. AN ESSAY STACKING QUESTIONS IS A VIOLATION EVEN IF
EACH QUESTION IS GOOD.

## Rule 1: Research before asking

Every question arrives after the relevant research is done and leads with an
evidence-backed recommendation. The question is a checkpoint on reasoning, not a
delegation of it. A flat menu that pushes synthesis onto the user is never acceptable.

**Correct:**

> The three unsaved-task sentinels exist only to distinguish drafts from deployed tasks,
> and the autosave model deletes that distinction. I recommend removing them outright
> rather than keeping a compatibility shim. Do you agree?

**Incorrect:**

> How should we handle the unsaved-task sentinels? Options: (a) keep them, (b) remove
> them, (c) shim them.

## Rule 2: Keystone first, dependency order

Draft the decision graph before asking anything. Find the keystone: the node whose
answer scopes or evaporates the most downstream questions, and ask it first. Broad
questions (UX, architectural principles, boundaries) come before implementation-specific
ones. When an answer deletes downstream questions, say so.

## Rule 3: Simple units

One decision per message. The user cannot juggle an essay stacking many questions, even
good ones. Context needed to answer travels with the question; context that serves a
later question waits for that question.

## Rule 4: The "it's intuition" move

When the user cannot articulate an answer ("this is intuition-based"), never just accept
the ambiguity and never push for a forced verdict. Build it together: offer a
recommendation and candidate decompositions (observable signals with directions), the
user reacts, iterate until a clear shared understanding exists. Codify the evidence and
the protocol; the verdict stays with the user.

## Rule 5: Termination

The interview ends when every remaining unknown is a parameter inside an agreed shape,
not a shape. If an open item still changes what other components look like, the
interview is not over. Remaining parameters are recorded, not re-asked.

## Rule 6: Locked means locked

Persist locked decisions as they accumulate (a working document during the interview,
then documents or memories) so the same ground is never re-interviewed. Deliver the full
implementation of what was locked; sequencing is the session's choice and never a
question.

## Quick reference

| Situation                  | Gate                                                 |
| -------------------------- | ---------------------------------------------------- |
| Asking the user anything   | Research done; recommendation attached; ONE question |
| Ordering questions         | Keystone first; broad before implementation-specific |
| User says "it's intuition" | Build it together; iterate to shared understanding   |
| Ending the interview       | Remaining unknowns are parameters, not shapes        |
| A decision locks           | Persist it; never re-interview the same ground       |
