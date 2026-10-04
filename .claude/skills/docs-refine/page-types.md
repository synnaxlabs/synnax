# Page types

The charter gives each page one primary type. Each h2 section holds one type. On a page
that mixes types, the sections go in this order: a short concept, then tasks, then
reference, then troubleshooting. Tutorials and landing pages stay pure. The audit flags
a section that mixes types or is out of order. It never flags a page for holding more
than one type. Sources: GitLab topic types, Diátaxis, The Good Docs Project.

## Concept

What a thing is, and when and why to use it.

- The title is a noun. Never "Introduction" or "How it works".
- The definition comes first, in one or two sentences.
- A section of canonical use cases, each linking to the page that shows how.
- No numbered steps. No restating of what the UI already shows.
- Diagrams and comparison tables ("use when") are welcome.

## Task

How to do one thing.

- The title is a verb and a noun ("Create channels").
- One sentence on what the task does and why, then prerequisites (with links to the
  pages that satisfy them), then numbered steps, then the result.
- One action per step. The location comes before the action ("In the ranges toolbar,
  click **Add**"). Ten steps at most.
- One method. Other interfaces go in `Client.Tabs`. Rare alternatives go in their own h2
  after the main method.
- The reader can tell the task worked: the page says what they see at the end.

## Tutorial

One guided path for a new reader.

- It works with sample data or no hardware, so every reader can finish it.
- The first paragraph says what the reader will build and how long it takes.
- Visible results early and at every step. Point out what the reader should notice.
- Minimal explanation, no options, no alternatives. Link out for depth.
- Seven main steps at most, and it must work every time, step for step.
- Ends with next steps: two to six links.

## Reference

Facts to look up.

- The title is a noun.
- Tables and field lists, built to scan. The structure follows the structure of the
  product.
- Describe, and only describe. No steps, no persuasion.
- Short examples where they help.

## Troubleshooting

Help when something goes wrong.

- Each entry starts from what the reader sees: the symptom or the exact error text.
- Then the cause, then the fix as steps.
- Troubleshooting goes last on a page. At five or more entries, it gets its own page.

## Landing

The site front page and section front pages only. It routes the reader.

- One or two sentences on what the site or section is for, then the main action, then
  the routes (cards or a short list).
- No content that only lives there. Every fact on a landing page is also on the page it
  links to.
