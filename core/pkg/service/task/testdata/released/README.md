# Released Core export fixtures

Each file is a task file that a stable Core exported. To make one, the Core at that
release tag imported the Console fixture of the same name from `../legacy` and then
exported the task. Only files that differ between releases are kept: the full set from
v0.59.2, plus the HTTP files that v0.58.2 and v0.59.0 wrote differently. There are no
Arc files, because released Cores refuse to export Arc tasks.

Do not regenerate these files. They are proof of what stable Cores wrote.
