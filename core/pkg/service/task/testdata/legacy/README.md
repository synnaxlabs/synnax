# Legacy task config fixtures

These files are frozen copies of the task file shape that the released Console
exported: `JSON.stringify({ ...task.config, type: task.type })`, with the config in the
camelCase in-memory form that each task type's zod schema declared. They were derived by
hand from the schemas at Git ref `origin/main` commit
`0d092325a152250ce4c729ff08fdbcd92470161d`, and every task type in this directory
existed at that commit. Each fixture sets every field the schema accepts to a
distinctive non-default value so a lossy import path fails a diff. Do not regenerate
these files from newer schemas; they must stay byte-stable as proof of what released
Consoles wrote.
