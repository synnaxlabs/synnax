# TypeScript Development

## Packages

pnpm workspaces with catalog dependencies: `/alamos/ts/` (instrumentation),
`/client/ts/` (Synnax client), `/console/` (Tauri app), `/drift/` (multi-window Redux
sync), `/freighter/ts/` (transport), `/x/media/`, `/pluto/` (viz components), `/x/ts/`
(shared utilities). All use Vite (dual CJS/ESM output), TypeScript 5.9+ strict, Vitest,
ESLint 9 flat config, Turbo for orchestration.

## Commands

- `pnpm build` / `pnpm build:<pkg>` (console, pluto, client, x, drift, freighter,
  alamos, media)
- `pnpm check-types` / `pnpm check-types:console`
- `pnpm dev:console` (Tauri) / `pnpm dev:console-vite` (Vite only, faster, no Tauri
  APIs) / `pnpm dev:pluto`
- `pnpm test` / `pnpm test:<pkg>`
- `pnpm lint` / `pnpm fix` (also per-pkg variants)
- `pnpm madge` / `pnpm madge:console` — circular dependency check

### Console dev login

Local server: host `localhost`, port `9090`, user `synnax`, password `seldon`. The login
form shows placeholders but does NOT pre-fill — actually type the values.

## Style

- Prettier, 88-char lines. ESLint 9 flat config.
- Function components + hooks; TypeScript instead of prop-types.
- Absolute imports: `@/components`, never `../../../components`.
- Inline type imports: `import { type Foo } from "bar"`; simple-import-sort for
  import/export ordering.
- JSX: no braces for string props, consistent boolean attributes, no constructed context
  values.
- `no-floating-promises`: await or `void` every promise. Unused vars must be
  underscore-prefixed.

## Namespaces & Imports

Modules form namespaces via barrels; consumers use member access.

### Rule 1: Namespaces are formed with `export * as`

A module's `index.ts` re-exports its public surface:
`export * as Channel from "@/channel/external"`. Never import-side
`import * as X from "..."` — the barrel is the only namespace mechanism. (Exception: a
dependency subpath that ships only named exports but is conceptually one namespace, e.g.
`import * as Drift from "@synnaxlabs/drift/react"`.)

### Rule 2: Cross-module imports go through the namespace

`import { Range } from "@/range"` then `Range.FavoriteButton` — never
`import { FavoriteButton } from "@/range/FavoriteButton"` from another module.
Same-module sibling files import named members directly and use them unqualified (the
barrel would be circular). Spec files follow the same rule — tests are not exempt.

### Rule 3: Package subpaths mirror `src/` layout

Where a package exposes subpath entries (`@synnaxlabs/pluto/testutil`), exports-map keys
and vite `lib.entry` keys use real slash paths
(`"telem/aether": "src/telem/aether/index.ts"`) so `dist/` mirrors the subpath.

### Rule 4: Alias only to resolve a name collision

`import { X as Y }` is banned except when the bare name `X` is already bound in the
file. Never alias to shorten a name, dodge the namespace-carries-context rule, or by
preference — an alias exists to name a real collision, nothing else. When a collision
forces one, which side gets the bare name and which gets aliased follows the file's own
subject:

- **Wrapping the same name**: a file whose own primary export shares the exact name of
  the lower-layer/pluto component it wraps aliases that import to `Base` (`Tree as Base`
  in a Tree wrapper, `Toolbar as Base` in a Toolbar wrapper, `Schematic as Base` in a
  Schematic tree adapter). `Base.<Member>` reads as "the underlying implementation."
- **Secondary collision**: when the colliding import isn't the file's own primary
  subject (a type built on top of it, a companion `Props` type, an unrelated same-named
  import), alias it to `Base` + the identifier (`Store as BaseStore`,
  `ChannelListProps as BaseProps`, `Diagram as BaseDiagram`).
- **Cross-package/layer collision**: when a local or feature-layer identifier collides
  with an imported one, alias the import to the identifier prefixed (or, for lowercase
  namespaces, suffixed) with a short tag for its origin, matching the identifier's own
  casing: `P` for `@synnaxlabs/pluto` (`Form as PForm`, `Menu as PMenu`, `CSS as PCSS`),
  `Platform` for console's `platform/` layer (`Device as PlatformDevice`,
  `Nav as PlatformNav`), `Client`/`client` for `@synnaxlabs/client` (`Synnax as Client`,
  `table as clientTable`), `X` for `@synnaxlabs/x` (`TimeSpan as XTimeSpan`), and the
  short name of any other third-party package (`Position as RFPosition` for
  `@xyflow/react`). The file's own identifier keeps the bare name; only the import is
  aliased.

## Comments

The universal body-comment and doc-comment rules in the root CLAUDE.md apply. TypeScript
form: JSDoc `/** */` directly above the declaration. Tags: `@param name - Description.`,
`@returns`, `@throws {ErrType} if ...` (see Error Handling Rule 4), `{@link Symbol}` for
cross-references. Never put types in JSDoc; the signature carries them.

```ts
/**
 * Parses the value into a TimeStamp.
 * @param value - The timestamp value to parse.
 * @returns The parsed TimeStamp.
 * @throws {ValidationError} if the value cannot be parsed.
 */
```

## Testing (Vitest — never Jest)

`*.spec.ts` co-located with source. `describe`/`it` blocks, `vi.fn()` mocks,
`@testing-library/react` for components and hooks, native async/await support. Avoid
testing implementation details.

## Error Handling

Typed-error system lives in `x/ts/src/errors/errors.ts`; canonical usage in
`freighter/ts/src/errors.ts` and `client/ts/src/errors.ts`. Domain errors are built with
`errors.createTyped("namespace")` / `.sub("subtype")` and registered with
`errors.register({ encode, decode })` so they survive worker / Tauri IPC / WebSocket
boundaries.

### Rule 1: Preserve the cause when re-throwing

Any wrapped, augmented, or replaced error must carry the original as `Error.cause` —
including synthetic errors built from non-Error rejections:

```ts
throw new Error(`failed to authenticate ${username}`, { cause: err });
return err instanceof Error ? err : new Error(String(err), { cause: err });
```

Pass-through re-throws that add no context stay `throw err` — wrapping them only adds
noise.

### Rule 2: Typed errors for matchable conditions

If any caller (now or plausibly soon) will branch on an error's identity, define a typed
error (`TreeError.sub("tab_not_found")`), register it, throw it, and match with
`TabNotFoundError.matches(err)`. Never throw `new Error("not found")` and expect
substring matching — a generic `Error` decodes as `errors.Unknown` on the other side of
any boundary. Bare `new Error("...")` is fine only for one-off invariants nothing
matches on.

### Rule 3: Never silently drop `.catch()` errors

Every `.catch()` must propagate, log, or document why ignoring is correct. Logging:
`console.error(err)`; add a short label only when the stack wouldn't name the failing
operation (`"failed to write state"`) — never a module/package prefix (no `"drift:"`).
Generic utilities can use bare `.catch(console.error)`. `.catch(() => {})` is a
debugging black hole.

### Rule 4: Document error contracts on public APIs

`@throws {AuthError} if ...` JSDoc on exported functions, methods, and hooks that can
throw/reject — most valuable on freighter/client/Aether surfaces; internal helpers
exempt.

### Rule 5: Type-check via pnpm scripts only

After error-handling changes run `pnpm check-types` and `pnpm lint`. Never raw `npx tsc`
(wrong tsconfig). Mistyped errors silently round-trip as `errors.Unknown`, so type
errors here are invisible at runtime.

## Package Layout

`src/index.ts` entry, tests co-located, `dist/` output (ESM `index.js`, CJS `index.cjs`,
types at `dist/src/index.d.ts`). Standard scripts: `build` =
`tsc --noEmit && vite build`, plus `check-types`, `test`, `lint`, `fix`, `madge`.

## Gotchas

- pnpm catalog: shared versions in `pnpm-workspace.yaml`; use `catalog:` prefix in
  package.json.
- Turbo build cache issues: `npx turbo clean`.
- React 19 (19.1.1) — check for React 18 breaking changes.

## Visual Verification (Playwright MCP)

After console UI changes, verify against the Vite dev server at `localhost:5173` (ensure
`pnpm dev:console-vite` is running or ask): `browser_navigate` → navigate to the
page/modal → `browser_snapshot` to inspect layout and form defaults. The command palette
(`Cmd+Shift+P`) reaches specific modals (e.g. `> Connect an HTTP server`). Take a fresh
snapshot before each interaction — refs go stale after navigation.
