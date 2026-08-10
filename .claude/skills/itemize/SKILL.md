---
name: itemize
description: >
  Present findings one at a time instead of as one long list. Use when the user invokes
  /itemize, or when presenting multiple findings (bugs, review comments, cleanliness
  issues, optimizations) that the user must reason about or decide on individually.
---

You have a set of findings (bugs, code cleanliness issues, optimization improvements,
review comments). A long, jargon-filled list is impossible for the reader to reason
about all at once. Do not present it.

Break the findings into individual items and present them ONE BY ONE.

## What one by one means

Pay careful attention to the meaning of ONE BY ONE:

- Each item is its own response to the user. One item per message, nothing else.
- After presenting an item, STOP and end your turn. The user must EXPLICITLY approve,
  reject, or answer before you move to the next item.
- Never batch: no "here are items 1 through 3", no presenting the next item in the same
  message that closes out the previous one.
- Ask for the decision in plain prose. Do not use question-widget tools like
  AskUserQuestion.

Before the first item, you may send a single short orientation line: how many items
there are and the order you will take them in. Nothing more.

## For each item

Research the finding deeply before presenting it. Make sure you thoroughly understand
not only what the finding is, but how to explain it clearly. Follow the /restate rules
for clarity: short jargon-free sentences, necessary context only, restate the specific
field, variable, or comment inline, and give a tight example when it helps.

If the finding may require changes, give your recommendation: implement (and which
implementation), skip, reply to the comment, and so on. With the recommendation, briefly
explain WHY it is the robust, production-grade, long-term approach. If you deliberately
recommend an option that is NOT the robust, long-term approach, say so explicitly and
explain why you chose it anyway.

## Persistence

Stay in this mode for the whole set of findings. Do not fall back to list form because
the remaining items seem small or similar. The mode ends when every item has been
decided or the user says to stop.
