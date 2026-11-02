# Theme: duplication

Remove duplicated code at its cause. The goal is thousands of removed lines each night,
across many small PRs. At the end of this prompt is a ranked list of clone groups that
jscpd found on `main`, with the config in `.github/agents/nightly/jscpd.json`. Work down
the list for your language. Skip a group that an open PR already covers.

For each clone group:

1. Read every copy and the code around it. Decide what the copies share and where that
   code belongs: an existing module that owns the concept, or a `common/` package next
   to the callers (never `shared/`).
2. Skip the group, and say why in the report, when the copies are:
   - intentional table data in a test, such as a table of cases;
   - copies that only look alike and change for different reasons;
   - generated, frozen (`versions/v*`), or vendored code that the config missed.
3. Extract the shared code once and make every copy call it. Follow `CLAUDE.md`:
   composition over inheritance, a deep module with a small interface, no pass-through
   functions, and a name that does not repeat its namespace. Prefer a test helper or a
   table-driven test for duplicated test code.
4. Keep behavior the same. The tests that cover the copies must pass unchanged, unless
   the change is to the tests themselves.
5. Count the lines before and after with `git diff --shortstat main`.

One clone group per PR. Put the line count in the description, as in
`Removes 240 lines (312 deleted, 72 added)`. This theme replaces the `Label` rule above:
add no review tier label and let the classifier pick one, except `review/human` when the
change alters the public API of a client library (`client/ts`, `client/py`,
`client/cpp`).

Keep each PR under about 200 changed lines. Split a large group by package. When a group
needs a design decision, such as where a shared type lives across layers, open no PR and
put the decision in the report.
