# Theme: slop

Remove code and comments that break the `CLAUDE.md` rules. Start with the files that
changed on `main` in the last 7 days (`git log --since=7.days --name-only main`), since
new code is where slop lands.

Look for:

- **Long comments**: a comment longer than the code it explains, a doc comment over 3
  lines with no contract, filler ("note that", "in order to", "simply"), a second
  sentence that repeats the first, a comment that restates the next line, section
  labels, and references to old code. Cut them. Wrap comment prose at 88 columns and
  fill each line.
- **Repeated namespace**: an identifier that repeats its package or module name, such as
  `channel.ChannelService` or `Label.LabelProvider`. Rename it and every caller. Skip a
  rename that changes a client API or a wire name; put it in the report.
- **Defense in depth**: a second guard against a fault that one fix already closes, or
  an error that is caught, skipped, or treated as empty to hide a fault. Remove the
  guard only when you can show which code makes it dead, and a test covers that code.
  Otherwise put it in the report.
- **Private doc comments** on identifiers whose names already say what they do.

One PR per package and per kind of finding. A comment-only PR is `review/bot`.
