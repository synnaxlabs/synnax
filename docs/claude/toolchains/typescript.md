# TypeScript Development

## TypeScript/JavaScript Packages

The monorepo uses **pnpm workspaces** with **catalog dependencies** for centralized
version management:

- **Alamos** (`/alamos/ts/`) - Distributed instrumentation and observability with
  OpenTelemetry integration
- **Client** (`/client/ts/`) - TypeScript client library for Synnax server API
- **Console** (`/console/`) - Main Tauri desktop application
- **Drift** (`/drift/`) - Redux state synchronization for multi-window Tauri apps
- **Freighter** (`/freighter/ts/`) - Protocol-agnostic transport layer (HTTP, WebSocket,
  gRPC)
- **Media** (`/x/media/`) - Media utilities and helpers
- **Pluto** (`/pluto/`) - High-performance React visualization component library
- **X** (`/x/ts/`) - Shared TypeScript utilities and helpers

All packages use:

- **Vite** for bundling with dual CJS/ESM output
- **TypeScript 5.9+** with strict mode
- **Vitest** for testing
- **ESLint 9** with flat config format
- **Turbo** for build orchestration and caching

## Build Commands

- `pnpm build` - Build all packages using Turbo
- `pnpm build:alamos` - Build Alamos instrumentation library
- `pnpm build:client` - Build only the client libraries
- `pnpm build:console` - Build only the Console application
- `pnpm build:drift` - Build Drift state synchronization library
- `pnpm build:freighter` - Build Freighter transport layer
- `pnpm build:media` - Build Media utilities
- `pnpm build:pluto` - Build only the Pluto component library
- `pnpm build:x` - Build X utility library
- `pnpm check-types` - Type check all TypeScript packages
- `pnpm check-types:console` - Type check only Console

## Development & Testing

- `pnpm dev:console` - Start Console in development mode (Tauri)
- `pnpm dev:console-vite` - Start Console Vite dev server only
- `pnpm dev:pluto` - Start Pluto development server
- `pnpm test` - Run all tests across packages
- `pnpm test:console` - Run Console tests
- `pnpm test:pluto` - Run Pluto tests
- `pnpm watch` - Watch mode for all packages

### Console Dev Login

When running the Console in dev mode, connect to a local Synnax server with:

- **Host**: `localhost`
- **Port**: `9090`
- **Username**: `synnax`
- **Password**: `seldon`

Note: The login form shows placeholder text but does NOT pre-fill values. You must
actually type the username and password into the fields before clicking Log In.

## Code Quality

- `pnpm lint` - Lint all packages with ESLint
- `pnpm fix` - Auto-fix linting issues across packages
- `pnpm lint:console` - Lint only Console package
- `pnpm fix:console` - Fix linting issues in Console

## Code Style

- **Formatter**: Prettier (configured in `.prettierrc`)
  - 88 character line length
  - Configured plugins for XML, Astro, shell scripts, TOML
- **Linter**: ESLint 9 with flat config (`eslint.config.ts`)
  - React plugin with strict JSX rules
  - TypeScript ESLint with type-checked rules
  - Simple import sort plugin for automatic import ordering
  - Consistent type imports: `import { type Foo } from "bar"`
- **Import style**: Prefer absolute imports over relative (`@/components` not
  `../../../components`)
- **React patterns**: Function components, hooks, no prop-types (use TypeScript)
- **Exports**: Dual CJS/ESM via Vite build

### Key ESLint Rules

- `@typescript-eslint/consistent-type-imports` - Inline type imports
- `simple-import-sort/imports` - Automatic import sorting
- `simple-import-sort/exports` - Automatic export sorting
- `react/react-in-jsx-scope` - Off (not needed in React 17+)
- `react/jsx-curly-brace-presence` - Never use braces for strings
- `react/jsx-filename-extension` - Only `.jsx` and `.tsx` files
- `react/jsx-boolean-value` - Enforce consistent boolean attribute style
- `react/jsx-no-constructed-context-values` - Prevent re-renders
- `@typescript-eslint/no-floating-promises` - Require await/void for promises
- `@typescript-eslint/no-unused-vars` - Allow underscore-prefixed vars

## Testing with Vitest

### Structure

- Test files use `*.spec.ts` extension
- Tests co-located with source code
- BDD-style with `describe`/`it` blocks

### Example

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

describe("Feature Name", () => {
  describe("Sub-feature", () => {
    beforeEach(() => {
      // Setup
    });

    it("should do something specific", () => {
      expect(result).toEqual(expected);
    });
  });
});
```

### Key Features

- **Mocking:** Uses `vi.fn()` for function mocks
- **Assertions:** Standard expect API (`toEqual`, `toBe`, `toHaveBeenCalledTimes`)
- **Setup/Teardown:** `beforeEach`, `beforeAll`, `afterAll`
- **Async testing:** Native async/await support
- **React Testing:** Uses `@testing-library/react` for component tests
- **Hooks testing:** `@testing-library/react` for custom hooks

## Package Structure

Each package follows this structure:

```
package-name/
├── src/
│   ├── index.ts              # Main entry point
│   ├── feature/
│   │   ├── feature.ts
│   │   └── feature.spec.ts   # Tests co-located
│   └── ...
├── dist/                     # Vite build output
│   ├── index.js              # ESM
│   ├── index.cjs             # CommonJS
│   └── src/                  # Type declarations
├── package.json
├── tsconfig.json
├── tsconfig.vite.json
├── vite.config.ts
├── eslint.config.ts
└── README.md
```

### package.json Pattern

```json
{
  "name": "@synnaxlabs/package-name",
  "type": "module",
  "exports": {
    ".": {
      "types": "./dist/src/index.d.ts",
      "import": "./dist/index.js",
      "require": "./dist/index.cjs"
    }
  },
  "scripts": {
    "build": "tsc --noEmit && vite build",
    "check-types": "tsc --noEmit",
    "test": "vitest",
    "lint": "eslint",
    "fix": "eslint --fix",
    "madge": "madge --circular src"
  }
}
```

## Circular Dependency Detection

Use `madge` to detect circular dependencies:

```bash
pnpm madge         # Check all packages
pnpm madge:console # Check only Console
```

## Error Handling

The codebase has a typed-error system in `x/ts/src/errors/errors.ts` that supports
network-portable encoding (the registry carries `name` and `stack` opaquely across the
worker / Tauri / WebSocket boundary). `freighter/ts/src/errors.ts` and
`client/ts/src/errors.ts` are the canonical examples — domain errors are built with
`errors.createTyped("namespace")` and `.sub("subtype")`, then registered with
`errors.register({ encode, decode })`.

The rules below mirror the Go discipline in `docs/claude/toolchains/go.md` adapted for
TypeScript's looser type system.

### Rule 1: Preserve the cause when re-throwing

When a caught error is wrapped, augmented, or replaced with a different error, the
original **must** be attached as `Error.cause` (ES2022). This keeps the underlying stack
trace and message printable via V8's automatic `[cause]:` chain rendering, which is the
single biggest tool for debugging distributed failures (worker boundaries, Tauri IPC,
async middleware).

This applies both to wrapping a real `Error` and to constructing a synthetic `Error`
from a non-`Error` rejection (e.g. `new Error(String(err))`).

**Correct:**

```ts
try {
  await client.send(...);
} catch (err) {
  throw new Error(`failed to authenticate ${username}`, { cause: err });
}

return err instanceof Error ? err : new Error(String(err), { cause: err });
```

**Incorrect — never do this:**

```ts
try {
  await client.send(...);
} catch (err) {
  throw new Error("authentication failed"); // ❌ loses err
}

return err instanceof Error ? err : new Error(String(err)); // ❌ drops original
```

**Pass-through re-throws are fine.** When the catch block does not add information (a
filter that re-throws everything that isn't `EOF`; a cleanup that re-throws after
closing a stream; an instrumentation middleware that records the error and propagates
it), keep `throw err` as-is. Wrapping a pass-through only adds noise.

```ts
try {
  return await op();
} catch (err: unknown) {
  if (EOF.matches(err)) return DONE;
  throw errors.fromUnknown(err); // pass-through — no new context to add, make it Error
}
```

### Rule 2: Prefer typed errors for matchable conditions

When callers (including tests) may want to branch on an error's identity —
`if (NotFoundError.matches(err))` — define a typed error class via
`errors.createTyped()` or `.sub()` from `x/ts/errors`, register it with
`errors.register({ encode, decode })`, and throw an instance. Do not throw
`new Error("not found")` and expect callers to substring-match the message.

A typed error survives Tauri IPC, Worker `postMessage`, WebSocket round-trips, and HTTP
freighter calls with its type intact, because the registry has an encoder/decoder for
it. A generic `Error` becomes `errors.Unknown` on the other side.

**Correct:**

```ts
// x/ts errors infrastructure
export class TabNotFoundError extends TreeError.sub("tab_not_found") {}

if (tab == null) throw new TabNotFoundError(`tab ${key} not found in mosaic`);

// caller
if (TabNotFoundError.matches(err)) return null;
```

**Incorrect — caller has to substring-match a magic string:**

```ts
if (tab == null) throw new Error("tab not found");           // ❌
if (err instanceof Error && err.message.includes("tab")) ... // ❌
```

Generic `throw new Error("...")` is acceptable for one-off invariants that nothing
matches on (assertions, internal sanity checks). When in doubt, the threshold is: "is
any caller, now or plausibly soon, going to want to detect this specific failure?" If
yes, type it.

### Rule 3: Never silently drop `.catch()` errors

A `.catch()` that returns `undefined`, swallows the rejection, or routes to a callback
that does nothing is a debugging black hole — every flaky failure becomes invisible.
Every `.catch()` must do at least one of:

- **Propagate** (re-throw, reject, or hand to an error callback).
- **Log** — `console.error(err)` or `console.error("short description", err)`. Add a
  label only when the stack trace V8 prints with the error would not, on its own, name
  the failing operation — e.g. a single function with multiple distinct `.catch()`
  sites, or a wrapping function where the inner call's identity is the interesting bit.
  Generic utilities (a hook like `useAsyncEffect`) can use bare `.catch(console.error)`;
  the stack already names them. When you do label, keep it to the operation that failed
  (`"failed to write state"`) — do **not** prefix with the package or module name (no
  `"drift:"`, `"persist:"`, `"useAsyncEffect:"`), since that information is already in
  the stack frames.
- **Document why ignoring is correct** — a one-line comment explaining the invariant
  that makes the failure safe to drop.

**Correct:**

```ts
engine.persist(state).catch((err: unknown) => {
  console.error("failed to write state", err);
});

featuresPromise.then(setFeatures).catch((err: unknown) => {
  console.error("failed to resolve language features service", err);
});
```

**Incorrect — never do this:**

```ts
featuresPromise.then(setFeatures).catch(() => {}); // ❌ silently dropped
```

### Rule 4: Document error contracts on public APIs

For exported functions, methods, and React hooks that can throw or reject, name the
errors they raise in the JSDoc. The reader should be able to learn what to handle
without reading the implementation.

```ts
/**
 * Authenticates against the cluster and returns the active user.
 *
 * @throws {AuthError} if the credentials are rejected.
 * @throws {Unreachable} if the cluster cannot be reached.
 */
async retrieveUser(): Promise<user.User> { ... }
```

This is most valuable for the freighter / client / Aether public surfaces. Internal
helpers don't need it.

### Rule 5: Always check types via the pnpm scripts

After error-handling changes — especially ones that touch `try` / `catch` shapes — run
`pnpm check-types` and `pnpm lint`. Do not run `npx tsc` directly; it may not pick up
the right `tsconfig`. The error-encoding registry will silently round-trip an
incorrectly-typed error as `errors.Unknown`, so type errors here can be invisible at
runtime.

## Common Gotchas

- **Console**: Has both Tauri (`dev:console`) and Vite-only (`dev:console-vite`)
  development modes
- **pnpm catalog**: Shared dependency versions in `pnpm-workspace.yaml`; use `catalog:`
  prefix in package.json
- **Turbo**: Build cache can cause issues; clear with `npx turbo clean`
- **Absolute imports**: TypeScript paths configured in tsconfig.json, ensure bundler
  respects them
- **Vitest**: Always import from "vitest", not Jest
- **React 19**: Using latest React 19.1.1 - check for breaking changes from React 18

## Visual Verification with Playwright

After making console UI changes, verify them visually using the Playwright MCP tools
against the Vite dev server at `localhost:5173`.

### Workflow

1. Ensure `pnpm dev:console-vite` is running (or ask the user)
2. Use `browser_navigate` to go to `http://localhost:5173`
3. Navigate to the relevant page, modal, or component
4. Use `browser_snapshot` to inspect the rendered UI and verify changes
5. Use the command palette (`Cmd+Shift+P`) to open specific modals or views

### Tips

- Take a fresh `browser_snapshot` before each interaction — refs go stale after
  navigation
- The command palette is useful for reaching specific forms (e.g.,
  `> Connect an HTTP server`)
- Check both the visual layout and the default values/state of form fields

## Development Best Practices

- **Always prefer absolute imports** over relative imports in TypeScript projects
- **Vitest for testing** - always use Vitest APIs, not Jest
- **Dependency injection & composition** - prefer composition over singletons and
  inheritance
- **Type everything** - use TypeScript strict mode, avoid `any`
- **Test co-location** - keep tests next to source files for better discoverability
- **Visual verification** - use Playwright MCP to verify console UI changes at
  `localhost:5173`
