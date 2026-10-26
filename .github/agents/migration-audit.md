# Audit migrations before a release

A release of Synnax is about to ship. Find each stored shape that changed between the
last stable release and the candidate, and prove that data from the stable release still
opens on the candidate. Nobody watches this session, so do not ask questions. Do not
change code or push commits: this run only reports.

Synnax owes backward compatibility only to what a stable release shipped. A format that
only a release candidate (`-rc.N`) or an unreleased commit wrote needs no migration.

## 1. Read

Read `CLAUDE.md`, `oracle/CLAUDE.md`, and `docs/claude/architecture.md`. The Oracle
rules on version files and on when to mint a version are the main contract here.

## 2. Find each stored-shape change

The Core, the Console, and the Driver release apart, so each has its own stable tag.
Diff each product's code from its stable tag to the candidate
(`git diff <stable>..<candidate>`). The Core tag covers the Core, Oracle, Gorp, Cesium,
and Aspen. Look at:

- **Oracle schemas** in `schemas/`. A version file that exists at the stable tag is
  frozen: any change to its stored bytes is a problem. A stored-shape change in a live
  file needs a new version file and a migration.
- **Gorp tables**: each `versions/` tree, `migrate.go`, `migrate.gen.go`, the entry
  types, their key types, and table names. Also look at migrations that read data that
  another service writes, and at migrations that run in the composition pass.
- **Cesium**: `cesium/internal/meta`, `cesium/internal/migrate`, and the on-disk layout
  of the domain and index files.
- **Aspen and KV**: key prefixes and the encoded values.
- **Core and Driver config**: CLI flags, environment variables, config files, and the
  files the Driver keeps on disk.
- **Console**: persisted state (`types/vN.ts` and `migrations.ts` files) and the import
  and export formats.

Wire shapes that nothing stores are out of scope.

## 3. Check each change

For each change, confirm all of these:

1. A migration exists and its chain registers it.
2. A spec writes the bytes the stable release wrote and checks the upgraded value.
3. Data from older stable releases also upgrades. A user can skip releases, so a
   migration must not depend on an intermediate release having run.
4. The migration does not depend on the order in which services open, and it does not
   decode old rows with the current type.

## 4. Upgrade real data

1. Download the stable Core:
   `gh release download <stable Core tag> --pattern 'synnax-v<version>-<linux|macos>'`.
2. Start it on an empty directory: `<binary> start -i -d <dir>`. Pass the license from
   `SYNNAX_LICENSE_KEY` when the environment has it.
3. In a new virtual environment, install the client of the stable minor
   (`uv pip install 'synnax==<major>.<minor>.*'`). Create at least one record of each
   shape from step 2.
4. Stop the stable Core. Start the candidate on the same directory:
   `go run main.go start -i -d <dir>` in `core/`. Leave out the `driver` build tag: it
   needs a built Driver in `core/pkg/driver/assets/`.
5. Read each record back with the candidate client in `client/py`, and read the Core log
   for migration errors.

If you cannot run this step, say why. That is a problem unless step 2 found no change.

## Report

Your last message is the report. List each problem with the file, the shape that the
stable release wrote, what is missing, and the evidence. Then list what you checked and
found correct, in one line each. Keep it short.

The last line of the report is `Verdict: pass` when you found no problem, otherwise
`Verdict: fail`. The release stops on any other last line.
