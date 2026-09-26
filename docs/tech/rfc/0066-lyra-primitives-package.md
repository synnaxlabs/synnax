# 66 Lyra - Primitives package and TypeScript build toolchain

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-23
- **Related**: [RFC 0051 - Pluto visual language](0051-pluto-visual-language.md)

## 0 Summary

Every documentation page ships 3.2 MB of JavaScript and 1 MB of CSS because the five
islands on each page import `@synnaxlabs/pluto`, and Pluto builds as one monolith. A
Pluto build spends 26 of its 27 seconds emitting type declarations through a plugin.
This RFC introduces `@synnaxlabs/lyra`, a subpath-only package holding the 19 UI
namespaces the docs site uses, built so a consumer pays only for the modules it imports,
and replaces the declaration plugin with native `tsc`. The docs site drops its Pluto
dependency. Pluto extends lyra by adding Synnax-aware members under the same namespace
names. A first attempt in May 2026 (`sy-4152-split-pluto-with-new-charon-package`) moved
forty modules including Aether and died in merge conflicts; this design moves the narrow
set and cuts the seams inside Pluto first.

## 1 Motivation

Measured on `main` at `e61d9bcdd8`, 2026-09-23:

- **Docs pages load all of Pluto**: `Search.tsx`, `Feedback.tsx`, `nav/Page.tsx`,
  `PageMobile` and `OnThisPage` hydrate on every page, and each one's static import
  closure is `pluto.*.js` (2.13 MB, 638 kB gzip) plus `tabs-*.js` (1.05 MB, 302 kB
  gzip). Pluto's own subpath entries share the same chunks because
  `pluto/vite.config.ts:81` sets `preserveModules: false`.
- **Docs CSS is Pluto CSS**: `Root.css` is 1.0 MB (467 kB gzip); `pluto.css` is 1.1 MB
  of it, and 394 kB of that is nineteen woff2 fonts inlined as data URIs, because Vite
  lib mode inlines every asset. The browser cannot cache them.
- **Declaration emit dominates the build**: a cold `turbo build` of the chain up to
  Pluto takes 48.7 s; Pluto alone 26.5 s, of which `unplugin-dts` reports
  `Declaration files built in 25749ms`. The JavaScript bundle takes 1.5 s. Native TS 7
  `tsc --noEmit` takes 0.6 s and `tsc --emitDeclarationOnly` 6.7 s on the same code.
- **The same code is type-checked three times**: `tsc --noEmit &&` in every build
  script, again inside the declaration plugin, and again in `check-types`, which
  `turbo.json` makes wait on `^build` because cross-package types resolve through
  `dist/src/*.d.ts`.
- **Hand-written export maps drift**: `pluto/package.json` exports `./dropdown`, which
  is not a build entry, and builds `dialog`, which is not exported.

## 2 Vocabulary

- **Primitive**: A main-thread React component or hook that knows nothing about Synnax
  data, the Core, or the Aether worker.
- **Aether**: Pluto's worker-thread component tree. Anything that calls `Aether.use`
  stays in Pluto.
- **Extension by addition**: A downstream package adds members to a concept an upstream
  package owns, under the same namespace name, without re-exporting the upstream.
- **Island**: An Astro component hydrated in the browser (`client:*` directive). Only
  islands ship JavaScript.

## 3 Principles

1. **Membership is the docs set**: Lyra holds exactly the namespaces the docs site uses
   on `main`, so the docs site needs no Pluto. Nothing else moves in this round.
2. **A consumer pays for what it imports**: Lyra is built one module per file, with CSS
   carried by the module's own import. No aggregate bundle, no aggregate stylesheet.
3. **Extension by addition, never re-export**: Pluto adds Synnax-aware members under a
   lyra namespace's name. It never re-exports lyra, and a consumer that needs both
   halves aliases the lyra one `as Base` (`docs/claude/toolchains/typescript.md` Rule
   4).
4. **Cut the seam before moving the module**: Every edge from a primitive into Aether,
   `Status`, or `Telem` is cut inside Pluto, in its own pull request, before the module
   leaves.
5. **One type pass**: Declaration emit is the type check. No build script runs `tsc`
   twice.
6. **Generated, not hand-written, package surfaces**: The exports map and the build
   entries derive from the `src/` layout, and a check fails when they drift.
7. **No new conventions for the packages that stay**: Pluto and x keep their root
   barrels and their build shape this round.

## 4 Design

### 4.0 Membership

Lyra holds the 19 namespaces the docs site imports on `main`: Divider, Note, Icon, Text,
Flex, Button, Breadcrumb, Dialog, Select, Triggers, Input, List, Component, Form, Nav,
Tree, Tabs, Video and CSS, plus the modules they depend on and Theming.

| Tier             | Modules                                                                                                         | Why                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| Leaf helpers     | context, state, generic, util, memo, hooks, position, store, key, portal, haul, cursor, resize                  | Imported by the tier above; no Pluto imports of their own |
| Aether-free UI   | css, flex, divider, note, text, icon, video, breadcrumb, caret, tooltip, triggers, nav, tag, progress, header   | Reach no Aether module on `main`                          |
| After a seam cut | theming, status (base half), telem/text, button, menu, dialog, list, input, select, form, tree, tabs, component | Each has one of the edges in §4.2                         |

Pluto keeps everything else: Aether, Synnax, Flux, Channel and the other resource
namespaces, the visualizations, Code, Arc, and `Pluto.Provider`.

### 4.1 Package shape

- `lyra/` at the repository root, `@synnaxlabs/lyra`, ESM only, `"files": ["dist"]`.
- **Subpath-only**: no `.` export. Every module is `@synnaxlabs/lyra/<module>` and
  exports one namespace (`export * as Button from "@/button/external"`). Nested modules
  use slash keys (`"status/aether"`), never dashes. Pluto and x keep their root barrels
  this round; the 782 Pluto-root and 1041 x-root import sites are a later program.
- **Generated exports**: `@synnaxlabs/vite-plugin` derives `lib.entry` from every
  `src/**/index.ts`, and `scripts/` gains a check, run with the other codegen checks,
  that fails when `package.json` exports differ from that layout.
- **Dependencies**: `react`, `react-dom`, `@synnaxlabs/x`, `react-icons`, `zod`, `clsx`,
  `@fontsource/*`. All are externals of the lyra build. `mathjs` (expressions in
  `Input.Numeric`) and `compromise` (phrases in the time inputs) are bundled under
  `dist/vendor/` and loaded with `import()` on first use; a plain number commits
  synchronously and the time grammar reads without the phrase parser. Lyra never imports
  `@synnaxlabs/client`.
- **Tests** move with their modules. Lyra gets its own `vitest` jsdom setup and
  `testutil/`; Pluto keeps its own copy of the setup file.

### 4.2 Seams to cut inside Pluto

Each edge below is cut in Pluto before its module moves, so the move is a pure rename.

- **`Button` → `Theming` → Aether**: `pluto/src/button/Button.tsx:211` calls
  `Theming.use()`, and `pluto/src/theming/Provider.tsx` imports `Aether` to push the
  theme to the worker. Theming splits: the theme spec, `toCSSVars`, fonts, the context,
  `use` and a main-thread `Provider` are the whole `Theming` namespace and move to lyra.
  The push to the worker becomes an internal bridge component in `pluto/src/pluto/` that
  reads `Theming.use()` and writes it through `Aether.use`, mounted by `Pluto.Provider`
  (`pluto/src/pluto/Pluto.tsx:64`). The worker-side `theming` stays under
  `@synnaxlabs/pluto/ether`.
- **`Status` aggregator is an Aether component**: `pluto/src/status/base/Aggregator.tsx`
  collects statuses through `Aether.use`. Lyra's `Status` holds the presentation
  (Indicator, Summary, Loading, Orbital, Notification, colors, icons, variant data) and
  a main-thread `Aggregator` with `useAdder`, `useErrorHandler`, `useAsyncErrorHandler`
  and `useNotifications`, over a lyra-owned status shape: `key`, `variant`, `name`,
  `message`, `description`, `time`, `details`. A client `status.Status` is assignable to
  it. Pluto's `Status` keeps `queries`, `Select`, `useCreateModal` and the worker
  aggregator; a bridge inside `Pluto.Provider` forwards worker statuses into lyra's
  `useAdder`. Four Console files use both halves today and alias the lyra one.
- **`Component` → `Select`**: `pluto/src/component/SelectSize.tsx` pulls `select` into
  every importer of `@/component`. It moves to `select/SelectSize.tsx`.
- **`Input.time` → `Telem.Text`**: `pluto/src/input/time/DateTime.tsx:35` and
  `grammar.ts:12` import `@/telem/text`. The `telem/text` components (TimeStamp,
  TimeSpan, TimeRange text) become lyra's `Telem`; Pluto's `Telem` keeps `Provider`,
  `SelectDataType` and the Aether side, by the same extension rule as `Status`.
- **`Select.Dialog`, `Form.use`, `Button.Copy` → `Status`**: satisfied by lyra's
  `Status` once it exists.

### 4.3 Extension by addition

A namespace name may exist in both packages. Lyra owns the generic half; Pluto adds the
Synnax half. Pluto never re-exports lyra: the May attempt's re-export broke the Vite
optimizer and hid which package a call site depended on. A file that needs both halves
writes:

```ts
import { Status as Base } from "@synnaxlabs/lyra/status";
import { Status } from "@synnaxlabs/pluto/status";
```

On `main`, 103 Console files use the lyra half of `Status`, 9 use the Pluto half, and 4
use both.

### 4.4 Build

- **Declarations** (all packages): the shared `lib()` plugin in
  `configs/vite/src/index.ts:27` drops `unplugin-dts` and runs native
  `tsc --emitDeclarationOnly --declaration --declarationMap` scoped to `src`, followed
  by a rewrite of `@/` specifiers to relative paths in the emitted `.d.ts` files (2,229
  imports across Pluto in 0.07 s). Build scripts drop the `tsc --noEmit &&` prefix;
  `check-types` stays `tsc --noEmit`; `watch` runs `tsc --watch --emitDeclarationOnly`
  beside `vite build --watch`. Verified on `main`: Console, docs and Pluto type-check
  with zero errors against the native output; declaration emit for the chain takes 9.9 s
  instead of ~35 s.
- **Lyra bundle**: `preserveModules: true` with `preserveModulesRoot: "src"`, ES only,
  `cssCodeSplit: true`. The 19-module set contains none of the Vite import queries
  (`?url`, `?raw`) that stop Pluto from using this mode.
- **CSS**: each module keeps its `import "./X.css"` in the emitted JavaScript. A
  `generateBundle` hook in `lib()` prepends the imports listed in
  `chunk.viteMetadata.importedCss` (the metadata is empty in `renderChunk` and populated
  in `generateBundle` on Vite 8). `"sideEffects": ["**/*.css"]` keeps them alive under
  tree-shaking. There is no `lyra.css`. CSS for the whole page ships as files an app
  imports once: `base.css` holds the root font size, element typography, shared utility
  classes and the font imports, and the generated static `theme.css` holds the theme
  variables for an app without `Theming.Provider`. No JavaScript module imports them, so
  tree-shaking cannot drop them.
- **Fonts**: `@fontsource/*` is external and imported by `base.css`, so the consumer's
  bundler emits the woff2 files as cacheable assets. The woff2 URL imports for the
  worker's canvas text (`pluto/src/theming/Provider.tsx:12-20`) stay in Pluto's bridge.
- **Turbo**: unchanged. `check-types` keeps `dependsOn: ["^build"]`; the build it waits
  on is now short.

### 4.5 Docs site

- `docs/site/package.json` drops `@synnaxlabs/pluto`; every import becomes a lyra
  subpath. The `reference/pluto` section and `Plot.tsx` are already gone on `main`
  (SY-4908).
- `astro.config.ts` adds `vite.ssr.noExternal: ["@synnaxlabs/lyra"]` so server-rendered
  components collect lyra CSS.
- Lyra CSS stays unlayered, like Pluto's. The docs site's 32 override rules depend on
  `@layer pluto, base, overrides` (`docs/site/src/styles/main.css:20-25`), so its
  PostCSS config wraps lyra's stylesheets into `layer(pluto)`. The alternative, a
  `@layer lyra` inside lyra, would let every unlayered Console rule beat every lyra rule
  regardless of specificity, a change nobody can audit cheaply.

### 4.6 Console and Pluto call sites

Every `import { Button } from "@synnaxlabs/pluto"` for a moved namespace becomes
`import { Button } from "@synnaxlabs/lyra/button"`, and every `@/button` inside Pluto
becomes `@synnaxlabs/lyra/button`. Sizes on `main`: 332 Console files import a moved
namespace from the Pluto root, 738 Pluto files import a moving module, 174 docs files
import Pluto. These rewrites are mechanical and ship as `review/bot` pull requests.

Pluto's root barrel stops exporting the moved names. `@synnaxlabs/pluto` is published,
so this is a breaking change for any external consumer; it lands with a minor bump and a
changelog entry that names the lyra subpath for each moved namespace.

## 5 Targets

Measured on the branch with the scripts used for the `main` baseline. A docs page is
`reference/concepts/overview`, JavaScript is the static import closure of its islands.

| Measure                            | `main` | Target                         | Branch |
| ---------------------------------- | ------ | ------------------------------ | ------ |
| JavaScript a docs page ships, gzip | 940 kB | The islands' own closure       | 252 kB |
| Docs CSS, gzip                     | 467 kB | Lyra CSS of the page's modules | 44 kB  |
| Docs client output on disk         | 4.8 MB |                                | 2.6 MB |
| Cold chain build to Pluto          | 48.7 s | About 25 s                     | 10.6 s |
| Pluto `vite build`                 | 27.3 s | About 8 s                      | 4.6 s  |
| Lyra `vite build`                  |        |                                | 4 s    |

The page still loads `mathjs` (174 kB gzip) and `compromise` (36 kB gzip) as separate
chunks, but only when a number input receives an expression or a time input receives a
phrase.

## 6 Implementation phases

The phases were built in this order and landed as one pull request into `main`: a module
lives in one package at any commit, so a phase boundary between the moves left Pluto,
the Console and docs importing from two packages without a reviewable seam.

- **Phase 0: Native declarations.** Replace `unplugin-dts` in `lib()`, add the alias
  rewrite, drop `tsc --noEmit &&` from build scripts, adjust `watch`. Standalone win.
- **Phase 1: Cut the seams in Pluto.** Split theming and status as in §4.2, move
  `SelectSize` into `select`, add the lyra-owned status shape (still inside Pluto). No
  new package, no import rewrites outside Pluto.
- **Phase 2: Scaffold lyra.** Package, build (§4.4), generated exports and its check,
  test setup, workspace and Turbo wiring, docs `noExternal` and the PostCSS layer wrap.
  Ships with the leaf helpers so the build is exercised.
- **Phase 3: Move the Aether-free UI tier** and rewrite Pluto, Console and docs imports
  for it. Mechanical rewrites in a separate `review/bot` pull request.
- **Phase 4: Move the seam-cut tier** (theming, status, telem text, button, menu,
  dialog, list, input, select, form, tree, tabs, component) the same way, and wire the
  two bridges into `Pluto.Provider`.
- **Phase 5: Docs drops Pluto.** Remove the dependency, measure, record the numbers
  against §5.

No flags: a module lives in one package at any commit.

Compatibility: no persisted or wire format changes. Published packages:
`@synnaxlabs/lyra` is new; `@synnaxlabs/pluto` loses the moved namespaces from its root.

## 7 What this RFC does not cover

- Removing the root barrels of Pluto and x, or building Pluto with `preserveModules`
  (blocked by its `?url`/`?raw` imports and bundled deps).
- Moving Aether, Flux, Synnax, or any visualization out of Pluto.
- Oracle emitting a base status shape into x (offered, declined for now).
- Docs site hydration strategy (`client:only` on every page).

## 8 Resolved decisions

- **Narrow membership, the docs set** (§4.0): The May attempt moved forty modules and
  1,943 files and never merged. The trade is real: Tabs, Select, Dialog, List, Tree,
  Input and Form need seam cuts before they move, and they are in the set because the
  docs islands use them.
- **Subpaths for lyra only** (§4.1): Every lyra import is a new line, so subpaths cost
  nothing; rewriting 1,800 Pluto and x root imports would cost weeks for a yield the
  docs measurements do not show (`sideEffects: false` on x left every docs chunk
  byte-identical).
- **Same names in both packages, with `Base` aliases** (§4.3): A different name for
  Pluto's half would claim it is a different concept. Rejected: renaming Pluto's
  status-resource pieces; re-exporting lyra from Pluto.
- **Lyra-owned status shape** (§4.2): Duplicates seven fields Oracle already defines.
  The trade is real; the Oracle route (`schemas/x/status.oracle` base plus client
  `extends`) stays open.
- **Native `tsc` declarations** (§4.4): The plugin exists to rewrite `@/` aliases; a
  20-line pass does that. Rejected: keeping the plugin with `SYNNAX_NO_DTS` escape
  hatches, which the May branch tried.
- **CSS carried by JS imports** (§4.4): Element Plus and Fluent UI v9 shape. Rejected:
  one aggregate stylesheet (Mantine, Radix Themes), which is what makes every docs page
  load 1 MB of CSS today, and what the May branch kept.
- **Docs wraps lyra CSS into its layer** (§4.5): Rejected `@layer lyra` inside lyra
  (Console cascade change) and rewriting docs overrides by specificity.

## 9 Open questions

None. The docs PostCSS mechanism is a plugin (`docs/site/src/util/layer.ts`) that wraps
every stylesheet under `lyra/` in `@layer pluto`. `Input.Numeric` needs no loading
state: a plain number commits synchronously, only an expression waits for `mathjs`. The
root-barrel change ships with the next Pluto minor.
