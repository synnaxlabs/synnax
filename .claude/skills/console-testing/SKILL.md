---
name: console-testing
description: Rules for writing and editing unit tests in the Console (`console/src/**/*.spec.ts[x]`). Use whenever authoring, refactoring, or reviewing a console Vitest spec, building or reaching for a test wrapper/store/fixture, deciding whether to mock or use real infrastructure, or wiring shared test scaffolding. Enforces blackbox testing through the public namespace, real-over-mock with dependency injection, and a single home per helper.
---

# Console Unit Testing

These rules govern every Vitest spec under `console/src`. They exist because a handful of
failure modes keep recurring: importing internals instead of the public API, reaching for
mocks when the real thing would do, stubbing browser primitives, and copy-pasting
scaffolding across files. The rules below are hard rules in the same register as
`docs/claude/toolchains/typescript.md` — follow them exactly.

The guiding principle behind all five: **a console test should exercise production code as
close to production as possible, through the same public surface a real caller uses, with
dependencies injected rather than mocked.** Reach for a mock only when there is genuinely
no way to use the real thing.

## Rule 1: Test through the public namespace, never a domain's internal files

Console domains are blackboxes. A spec imports the thing it tests — and everything else it
touches — through the domain's public namespace barrel, exactly like a Go `package
foo_test` tests through the exported surface. Reaching into a domain's internal file with a
deep named import is prohibited, **including for the thing under test itself.** The public
namespace export *is* the contract; if you test against the file, you are testing an
internal that no real caller depends on.

Every domain surfaces itself as a namespace (`export * as Log from "@/session/log/external"`,
`export * as Bar from "@/app/nav/bar/external"`). Use it.

**Correct:**

```ts
import { Session } from "@/session";
import { Nav } from "@/app/nav";

// the thing under test, reached through its namespace
render(<Nav.Bar.Top />);
store.dispatch(Session.Log.create({ ... }));
```

**Incorrect — never do this:**

```ts
import { Top } from "@/app/nav/bar/Top";          // ❌ deep import of the SUT's file
import { create } from "@/session/log/slice";     // ❌ deep import into another domain
```

**The one carve-out: co-located test scaffolding.** A spec's own `testutil` file
(`@/app/nav/bar/testutil`, `@/platform/modals/testutil`) is test infrastructure, not the
domain's public API, so named imports from it are fine. This is the *only* exemption.

```ts
import { renderBar, TIMEOUT } from "@/app/nav/bar/testutil";  // ✅ test scaffolding
```

**If the thing you want to test is not in a barrel, that is a signal, not a license.** Do
**not** add a barrel export just so a test can reach it — that is "export only for tests,"
which is banned (see `feedback_no_export_for_tests`). Either the symbol is genuinely public
and belongs in the barrel on its own merits, or it is internal and you test it through the
public API that exercises it.

## Rule 2: Prefer real infrastructure over mocks — the dependency hierarchy

When a test needs a collaborator, work down this list and stop at the first tier that works.
Do not skip to a lower tier because it is faster to type.

1. **Real client + real flux stores against a live cluster.** This is the default reach for
   anything that touches data, queries, or the flux store. Use `createTestClient()` from
   `@synnaxlabs/client` (connects to a real local cluster) plus `createConsoleWrapper` /
   `renderHookWithConsole` from `@/testutil/testutil`. This exercises the production query
   infrastructure end to end.

2. **Real store + real flux with a `null` client and preloaded state.** For logic that
   needs no network — pure reducers, selectors, layout placement. Build a minimal store with
   the real reducers. A `session/*/slice.spec` or `selectors.spec` configuring a
   single-slice store inline is correct and preferred — do **not** force it through the full
   provider stack.

3. **`vi.fn()` injected through a prop or DI parameter.** A spy handed to a component as a
   leaf callback is **not a mock of production behavior** — it is the dependency-injection
   pattern this codebase wants. It is encouraged, not discouraged. What is discouraged is
   reaching for `vi.mock` to dodge wiring up the real collaborator.

4. **`vi.mock` a module / `vi.stubGlobal` (last resort).** Only for genuinely unmockable
   runtime seams that have no injection point — e.g. `isMainWindow`, a version check. Today
   only three console specs legitimately need this. If you are reaching for `vi.mock` on a
   fourth kind of thing, stop and find the injection point instead.

**Correct — real cluster is the default for data-touching tests:**

```ts
import { createTestClient } from "@synnaxlabs/client";
import { createConsoleWrapper } from "@/testutil/testutil";

const client = createTestClient();
const { wrapper } = await createConsoleWrapper({ client });
const { result } = renderHook(() => Table.useCreate(), { wrapper });
```

**Correct — a spy injected via DI is fine:**

```ts
const onChange = vi.fn();
render(<Table.CellForm onChange={onChange} />);
// ... assert onChange was called with the right args
```

**Incorrect — mocking the module to avoid the real collaborator:**

```ts
vi.mock("@/session/log/slice");  // ❌ use the real reducer with a preloaded store
```

**Authoring vs. running.** Writing live-core tests against `createTestClient` is correct and
encouraged. But do **not** *execute* those specs against the user's dev cluster without
approval — hand the actual run to the user (see `feedback_no_tests_against_running_server`).
Verify your work with `check-types` / `lint` / `prettier`.

## Rule 3: Never stub DOM primitives or substitute components

Two specific mock abuses are always wrong:

- **Stubbing a browser primitive** (`getBoundingClientRect`, a canvas 2D context, layout
  APIs) to make a component render. A `getBoundingClientRect` stub is a smell, not a
  technique — it means the component is being tested outside the environment it runs in.
  Render it in the real provider stack instead.
- **Substituting a child component with a placeholder** and asserting the placeholder
  appears (see `feedback_no_mock_substitution`). Either mount the real child through the
  real wrapper and assert on real DOM, or delete the assertion. Function spies are fine;
  component substitutes are not.

**Incorrect — never do this:**

```ts
vi.spyOn(Element.prototype, "getBoundingClientRect").mockReturnValue({ ... }); // ❌
vi.mock("@/feature/table/Table", () => ({ Table: () => <div data-testid="table" /> })); // ❌
```

## Rule 4: Cross-package shared scaffolding must be vitest-free

A test helper may cross a package boundary (e.g. be published from `@synnaxlabs/pluto` and
imported by console) **only if it imports nothing from `vitest` and exposes no vitest types
(`Mock`, `MockInstance`) in its signatures.** `vi` is a runtime import; publishing a helper
that uses it drags vitest into the shipped bundle graph, and its types do not resolve
portably in a consumer that configures TypeScript differently. This is a real
build-toolchain trap, not a style preference.

- **Vitest-free scaffolding is publishable.** The provider-stack wrapper (`createSynnaxWrapper`
  — Aether/Status/Synnax/Flux) is vitest-free and belongs in the lowest package that owns
  what it wraps. `@synnaxlabs/client` already publishes `createTestClient` this way.
- **Vitest-coupled helpers stay in-package and are never exported.** Anything that uses `vi`
  internally or returns a `Mock` (e.g. pluto's `mockRenderContext`, `mockBoundingClientRect`)
  stays where it is defined.
- **If a consumer needs a spy, it injects its own `vi.fn()`.** A shared helper types that
  parameter as a plain function, never a vitest `Mock`. (This is Rule 2 tier 3 again — the
  package boundary makes DI mandatory rather than merely preferred.)
- A published testutil entry (e.g. `@synnaxlabs/pluto/testutil`) is a **guarded surface**:
  adding a `vi`-touching helper to it is the specific mistake that breaks the toolchain.

## Rule 5: One home per helper — hoist on the second use, never copy

Every shared test helper lives in exactly one place: the lowest tier below that covers all
its users. Copying a helper between two spec files *is* the violation.

1. **Cross-package, vitest-free** → the owning package's published testutil (Rule 4).
2. **Console-wide** (provider stack, store factory, `renderWithConsole`,
   `renderHookWithConsole`, `createConsoleWrapper`, `renderLinkHook`, genuinely global
   fixtures) → `console/src/testutil`. Single home.
3. **Domain-specific** (`renderBar`, a modal `Wrapper`, a domain's fixture builders) →
   co-located `@/<layer>/<domain>/testutil`, **composing on top of `@/testutil`** — never
   re-deriving the provider stack or re-implementing store construction.
4. **Inline in one spec** → only when exactly one spec uses it and it is trivial.

**The threshold: the second use is the trigger to hoist.** One spec → inline is fine. The
moment a second spec needs the same helper, move it up to the nearest shared testutil. Never
paste it into the second file.

**Two guardrails so this does not recreate a bloated god-module:**

- `console/src/testutil` owns *global* scaffolding only — providers, store, render entry
  points. Domain-specific fixtures do **not** accumulate there; they live in the domain
  testutil.
- A pure slice/selector spec building a minimal single-slice store inline is allowed and
  preferred (Rule 2 tier 2). It must not be forced through `renderWithConsole`.

**Incorrect — re-deriving `@/testutil` inline (real example of the anti-pattern):**

```ts
// platform/table/useCreate.spec.tsx, before
const buildHarness = async () => {
  const fluxClient = new Flux.Client({ client, storeConfig: { ...Pluto.FLUX_STORE_CONFIG }, ... });
  const rootReducer = combineReducers({ [Drift.SLICE_NAME]: Drift.reducer, ... });
  // ...30 lines rebuilding what @/testutil already exports
};
```

**Correct — use the shared home:**

```ts
import { createConsoleWrapper } from "@/testutil/testutil";

const { wrapper, store } = await createConsoleWrapper({ client });
```

## Writing a console test — quick reference

Reach for these, in order of how close they are to production:

| Need | Use | From |
| --- | --- | --- |
| Component/hook that hits data, against a live cluster | `createConsoleWrapper({ client })` + `renderHook`/`render` | `@/testutil/testutil` + `createTestClient()` |
| Component/hook needing the provider stack, no network | `renderWithConsole` / `renderHookWithConsole` | `@/testutil/testutil` |
| A deep-link resource hook | `renderLinkHook` | `@/testutil/testutil` |
| Pure reducer/selector | minimal single-slice `configureStore` inline | `@reduxjs/toolkit` |
| Flux queries without console slices | `createSynnaxWrapper` / `createAsyncSynnaxWrapper` | `@/testutil/Synnax` |

And always: import the thing under test and its dependencies through their **namespace**
(Rule 1); prefer **real** over mock (Rule 2); never stub DOM primitives or substitute
components (Rule 3); put shared helpers in **one home** (Rule 5).
