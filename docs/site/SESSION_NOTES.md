# Cleanup Session Notes

## Session 1: Toolchain & CI/CD

**Date:** 2026-01-28 **Completed:** [x] Phase 0, [x] Phase 1

### What was done:

#### Phase 0 - Switch to `astro check`

1. Added `@astrojs/check@^0.9.6` to devDependencies
2. Updated package.json scripts:
   - `"build": "astro check && astro build"`
   - `"check-types": "astro check"`
3. Fixed 7 type errors found by `astro check` that `tsc --noEmit` was missing:

| File                        | Error                                        | Fix                                                                           |
| --------------------------- | -------------------------------------------- | ----------------------------------------------------------------------------- |
| `CodeI.astro`               | Missing type for props                       | Added `type Props = ComponentProps<typeof Code>`                              |
| `ReleaseList.astro`         | Missing required `url` prop                  | Removed unused `url` from ReleaseList.tsx type definition                     |
| `ReleaseTile.astro`         | `shade` prop doesn't exist on Text.Text      | Changed to `color={7}`                                                        |
| `Root.astro`                | `anchor.style` TypeScript error              | Added generic type to querySelectorAll: `querySelectorAll<HTMLAnchorElement>` |
| `guides/Tile.astro`         | `sName` prop doesn't exist on Flex.Box       | Changed to `className`                                                        |
| `SynnaxDownloadURL.astro`   | `lang` string not assignable to CodeLanguage | Added explicit type: `let lang: "bash" \| "powershell"`                       |
| `SynnaxVersionOutput.astro` | Empty string not valid CodeLanguage          | Changed `lang=""` to `lang="text"`                                            |

4. Verified build still works: `astro check && astro build` succeeds

#### Phase 1 - Create test.docs.yaml

1. Created `.github/workflows/test.docs.yaml` following the pattern from
   `test.console.yaml`
2. Simplified to single ubuntu runner (docs don't need multi-OS testing)
3. Triggers on:
   - `pull_request` with path filters matching `deploy.docs.yaml`
   - `push` to `main` and `rc` branches with same path filters
   - `workflow_dispatch` for manual runs
4. Steps: checkout, pnpm setup, node setup, install, check-types, lint, build
5. Added `configs/eslint/**` to path triggers (missing from deploy.docs.yaml)

### Issues encountered:

1. **@astrojs/check version**: Initially specified `^0.10.0` which doesn't exist. Latest
   is `0.9.6`.

2. **Astro Code component Props export**: Tried to import `Props` as named export but
   it's only available via `ComponentProps<typeof Code>` from `astro/types`.

3. **Warnings about unsupported files**: The benchmark Python files in
   `src/pages/guides/comparison/performance/one-billion-rows/bench/` cause warnings.
   They should be prefixed with `_` to ignore, but this is documentation content and not
   part of this cleanup scope.

### Learnings for next session:

1. **`astro check` finds real issues** - The 7 type errors it found were all legitimate
   issues that `tsc --noEmit` couldn't catch because it ignores `.astro` files.

2. **Pluto component types** - Several errors were due to using non-existent props on
   Pluto components:
   - `shade` doesn't exist on `Text.Text` (use `color` instead)
   - `sName` doesn't exist on `Flex.Box` (use `className` instead)
   - This suggests the codebase may have other similar issues that went unnoticed.

3. **TypeScript in Astro files** - Proper typing is important:
   - Always define `type Props` for component props
   - Use `ComponentProps<typeof Component>` to inherit types from other Astro components
   - Use generic type parameters on `querySelectorAll<T>()` for DOM elements

4. **CI workflow patterns** - The repo uses:
   - `test.*.yaml` for validation (runs on PRs and push to main/rc)
   - `deploy.*.yaml` for deployment (runs on push to main only)
   - Path triggers should match between related workflows

---

## Session 2: Folder Structure & Styles

**Date:** 2026-01-28 **Completed:** [x] Phase 2, [x] Phase 3

### What was done:

#### Phase 2 - Folder Structure

1. Created `src/styles/` directory
2. Created `src/assets/` directory
3. No assets needed to move from `public/` (only `favicon.svg` which belongs there)

#### Phase 3 - Style Organization with CSS Layers

1. Created CSS layer system in `src/styles/main.css`:

   ```css
   @layer pluto, base, overrides;
   @import "@synnaxlabs/pluto/dist/pluto.css" layer(pluto);
   @import "@synnaxlabs/pluto/dist/theme.css" layer(pluto);
   @import "@synnaxlabs/media/dist/media.css" layer(pluto);
   /* CSS @import uses file-relative paths (standard CSS behavior) */
   @import "./base.css" layer(base);
   @import "./pluto.css" layer(overrides);
   @import "./layout.css" layer(overrides);
   ```

2. Extracted styles from Root.astro into organized CSS files:
   - `src/styles/main.css` - Layer declarations and imports
   - `src/styles/base.css` - Resets, typography, CSS variables
   - `src/styles/pluto.css` - Pluto component overrides
   - `src/styles/layout.css` - Page structure, navigation, article layout
3. Updated Root.astro to import main.css in frontmatter
4. Root.astro went from ~500 lines to ~90 lines
5. **Eliminated all `!important` declarations** from layered CSS files
6. **Eliminated all `!important` declarations** from Article.astro (11 removed)
7. Added Firefox scrollbar support (`scrollbar-width`, `scrollbar-color`)
8. Fixed bug in Article.astro: wrong selector `.pluto-text-link` → `a.pluto-text`

### Results:

| Metric                            | Before | After |
| --------------------------------- | ------ | ----- |
| Root.astro lines                  | ~500   | ~90   |
| `!important` in Root.astro        | 27     | 0     |
| `!important` in Article.astro     | 11     | 0     |
| **Total `!important` eliminated** | **38** | **0** |
| Firefox scrollbar support         | ❌     | ✅    |

### How CSS Layers Work:

The key insight is that **layer order determines cascade priority**, not specificity:

- `@layer pluto, base, overrides;` - Later layers win
- Pluto CSS is in `pluto` layer (lowest priority)
- Our overrides are in `overrides` layer (highest priority)
- **Unlayered CSS always beats layered CSS** - This is why Article.astro's
  `<style is:global>` (unlayered) beats Pluto (layered) without needing `!important`

### Styles organization (in src/styles/):

| File         | Contents                                                                    |
| ------------ | --------------------------------------------------------------------------- |
| `main.css`   | Layer declarations, imports Pluto and our CSS into layers                   |
| `base.css`   | HTML/body resets, CSS variables, typography, tables, links, code, scrollbar |
| `pluto.css`  | .pluto-pack, .pluto-tabs-selector, .pluto-tree, .pluto-dialog overrides     |
| `layout.css` | main, nav, .page-nav, article structure                                     |

### Bug fix:

Article.astro breadcrumb styles were targeting wrong class:

- **Before:** `.pluto-text-link` (doesn't exist)
- **After:** `a.pluto-text` (actual Pluto class)

This was a pre-existing bug, not something we broke during refactoring.

### What was NOT done (pending for future session):

- Convert 19 files with `<style is:global>` to scoped `<style>`
- Use `:global()` selector pattern for Pluto children

### Learnings for next session:

1. **CSS Layers DO work with external CSS** - The `@import "..." layer(name)` syntax
   successfully wraps Pluto CSS in a layer, giving our overrides higher priority.

2. **Layer order is key** - Declare layers first (`@layer pluto, base, overrides;`),
   then import. Later layers beat earlier layers regardless of specificity.

3. **Unlayered beats layered** - Component `<style is:global>` blocks are unlayered, so
   they automatically beat layered Pluto styles without `!important`.

4. **Always verify class names** - The breadcrumb bug was caused by using a non-existent
   class name. DevTools CSS panel shows actual applied classes.

5. **Firefox needs different scrollbar properties** - `::-webkit-scrollbar` is
   Chrome/Safari only. Firefox uses `scrollbar-width` and `scrollbar-color`.

---

## Session 3: Component Organization

**Date:** 2026-01-28 **Completed:** [x] Phase 4

### What was done:

#### Phase 4 - Namespace Pattern Applied

1. **Created namespace pattern** (`index.ts` + `external.ts`) for folders with TypeScript exports:
   - `article/` - Breadcrumb component
   - `client/` - Already had pattern, verified
   - `code/` - codeUtil exports
   - `console/` - Fixed to use external.ts pattern
   - `driver/` - Tabs component
   - `feedback/` - FeedbackButton component
   - `media/` - Image, Video components (moved from flat file)
   - `nav/` - **NEW** consolidated folder (from PageNav + OnThisPage)
   - `platform/` - Already had pattern, verified
   - `pluto/` - Plot components
   - `releases/` - ReleaseList component
   - `search/` - Search component
   - `table/` - Table component (moved from flat file)
   - `tabs/` - Tabs component (moved from flat file)
   - `text/` - textFactory (moved from flat file)

2. **Moved flat files into folders:**
   - `Table.tsx`, `Table.astro` → `table/`
   - `Tabs.tsx` → `tabs/`
   - `Text.tsx` → `text/`
   - `Media.tsx` → `media/`
   - `Header.astro`, `Footer.astro` → `layout/`
   - `NextPrev.astro` → `article/`

3. **Consolidated nav components:**
   - Merged `PageNav/` and `OnThisPage/` into single `nav/` folder
   - Renamed `PageNav` → `Page`, `PageNavMobile` → `PageMobile`
   - Kept `OnThisPage` name (descriptive within nav context)

4. **Created `components/index.ts`** as single entry point for all namespaces

5. **Updated 85+ import statements** across MDX files and components

### Folders WITHOUT namespace files (Astro-only, imported directly):

These folders only contain Astro components and don't benefit from namespace pattern:
- `core/` - WindowsDownloadButton.astro
- `deploy/` - Synnax download/version Astro components
- `details/` - Details, ExampleDetails, ChannelTypeDetails Astro components
- `layout/` - Header.astro, Footer.astro

### Final component structure:

```
components/
├── index.ts                    # Single entry point for all namespaces
├── mdxOverrides.ts             # MDX component overrides (special)
├── article/
│   ├── index.ts + external.ts  # Article namespace
│   ├── Article.astro
│   ├── Breadcrumb.tsx
│   └── NextPrev.astro
├── client/
│   ├── index.ts + external.ts  # Client namespace
│   ├── client.tsx
│   ├── Tabs.tsx
│   ├── SelectButton.tsx
│   └── Var.tsx
├── code/
│   ├── index.ts + external.ts  # Code namespace
│   ├── Code.astro
│   ├── CodeI.astro
│   └── codeUtil.ts
├── console/
│   ├── index.ts + external.ts  # Console namespace
│   └── DownloadButton.tsx
├── core/                       # No namespace (Astro only)
│   └── WindowsDownloadButton.astro
├── deploy/                     # No namespace (Astro only)
│   ├── SynnaxDownloadURL.astro
│   ├── SynnaxMoveCommand.astro
│   └── SynnaxVersionOutput.astro
├── details/                    # No namespace (Astro only)
│   ├── Details.astro
│   ├── ExampleDetails.astro
│   └── ChannelTypeDetails.astro
├── driver/
│   ├── index.ts + external.ts  # Driver namespace
│   ├── DownloadURL.astro
│   └── Tabs.tsx
├── feedback/
│   ├── index.ts + external.ts  # Feedback namespace
│   ├── Feedback.astro
│   └── Feedback.tsx
├── layout/                     # No namespace (Astro only)
│   ├── Header.astro
│   └── Footer.astro
├── media/
│   ├── index.ts + external.ts  # Media namespace
│   └── Media.tsx
├── nav/
│   ├── index.ts + external.ts  # Nav namespace
│   ├── Page.tsx
│   ├── Page.astro
│   ├── OnThisPage.tsx
│   └── OnThisPage.astro
├── platform/
│   ├── index.ts + external.ts  # Platform namespace
│   ├── platform.tsx
│   ├── Tabs.tsx
│   └── SelectButton.tsx
├── pluto/
│   ├── index.ts + external.ts  # Pluto namespace
│   ├── Plot.tsx
│   ├── ComponentFrame.astro
│   └── worker.ts
├── releases/
│   ├── index.ts + external.ts  # Releases namespace
│   ├── Release.astro
│   ├── ReleaseList.astro
│   ├── ReleaseList.tsx
│   ├── ReleaseTile.astro
│   └── PolkaDot.astro
├── search/
│   ├── index.ts + external.ts  # Search namespace
│   ├── Search.tsx
│   └── Search.astro
├── table/
│   ├── index.ts + external.ts  # Table namespace
│   ├── Table.tsx
│   └── Table.astro
├── tabs/
│   ├── index.ts + external.ts  # Tabs namespace
│   └── Tabs.tsx
└── text/
    ├── index.ts + external.ts  # Text namespace
    └── Text.tsx
```

### Naming conventions applied:

- **Folders:** lowercase (`nav/`, `table/`, not `Nav/`, `Table/`)
- **Namespace.Component:** Avoid redundancy (`Nav.Page` not `Nav.PageNav`)
- **Namespace.Namespace:** OK for main component (`Tabs.Tabs`, `Table.Table`)

### Issues encountered:

1. **Empty external.ts files** - Initially created empty external.ts files for Astro-only
   folders with `export {}`. This was useless - removed them entirely.

2. **Astro components can't be re-exported from TS** - Even with `astro check`, you
   can't `export * from "Component.astro"`. Astro components must be imported directly.

### Learnings for next session:

1. **Namespace pattern only for TS exports** - Don't create index.ts/external.ts for
   folders that only contain Astro components. They provide no value.

2. **Consolidate related components** - Merging PageNav + OnThisPage into `nav/` was
   the right call. Both are navigation-related.

3. **Rename during move** - When moving files to new folders, good time to rename
   (PageNav → Page, PageNavMobile → PageMobile).

4. **MDX imports need updating** - Many MDX files had direct component imports that
   needed updating. A Task agent was helpful for bulk updates.

---

## Session 4: Naming & MDX

**Date:** 2026-01-28 **Completed:** [x] Phase 5, [x] Phase 6

### What was done:

#### Phase 5 - Naming Conventions

1. **Renamed component files to PascalCase:**
   - `platform/platform.tsx` → `platform/Platform.tsx`
   - `client/client.tsx` → `client/Client.tsx`

2. **Renamed code components to avoid namespace redundancy:**
   - `code/Code.astro` → `code/Block.astro` (usage: `Code.Block`)
   - `code/CodeI.astro` → `code/Inline.astro` (usage: `Code.Inline`)

3. **Renamed releases components:**
   - `releases/ReleaseList.astro` → `releases/List.astro`
   - `releases/ReleaseTile.astro` → `releases/Tile.astro`
   - `releases/ReleaseList.tsx` → `releases/List.tsx`

4. **Renamed details components:**
   - `details/ExampleDetails.astro` → `details/Example.astro`
   - `details/ChannelTypeDetails.astro` → `details/ChannelType.astro`

5. **Moved standalone components to namespace folders:**
   - `StepText.astro` → `text/Step.astro` (usage: `Text.Step`)
   - `Diagram.astro` → `diagram/Diagram.astro` (usage: `Diagram.Diagram`)
   - `Rule.astro` → `rule/Rule.astro` (usage: `Rule.Rule`)
   - `PostHog.astro` → `analytics/PostHog.astro` (usage: `Analytics.PostHog`)

6. **Created new namespaces with Astro component exports:**
   - `Analytics` - `Analytics.PostHog`
   - `Core` - `Core.WindowsDownloadButton`
   - `Deploy` - `Deploy.DownloadURL`, `Deploy.MoveCommand`, `Deploy.VersionOutput`
   - `Details` - `Details.Details`, `Details.Example`, `Details.ChannelType`
   - `Diagram` - `Diagram.Diagram`
   - `Rule` - `Rule.Rule`

7. **Updated existing namespaces with Astro exports:**
   - `Code` - `Code.Block`, `Code.Inline`
   - `Table` - `Table.Table` (Astro component)
   - `Text` - `Text.Step`
   - `Driver` - `Driver.DownloadURL`
   - `Releases` - `Releases.Release`, `Releases.List`, `Releases.Tile`, `Releases.PolkaDot`
   - `Pluto` - `Pluto.ComponentFrame`

#### Phase 6 - MDX Standardization

1. **Converted ALL direct Astro imports to namespace pattern** across 70+ MDX files

2. **Import patterns converted:**
   | Before | After |
   |--------|-------|
   | `import Table from "@/components/table/Table.astro"` | `import { Table } from "@/components"` |
   | `import Diagram from "@/components/Diagram.astro"` | `import { Diagram } from "@/components"` |
   | `import Rule from "@/components/Rule.astro"` | `import { Rule } from "@/components"` |
   | `import StepText from "@/components/StepText.astro"` | `import { Text } from "@/components"` |
   | `import Release from "@/components/releases/Release.astro"` | `import { Releases } from "@/components"` |
   | `import SynnaxDownloadURL from "@/components/deploy/..."` | `import { Deploy } from "@/components"` |
   | `import WindowsDownloadButton from "@/components/core/..."` | `import { Core } from "@/components"` |
   | `import Details from "@/components/details/Details.astro"` | `import { Details } from "@/components"` |
   | `import ExampleDetails from "@/components/details/..."` | `import { Details } from "@/components"` |
   | `import ComponentFrame from "@/components/pluto/..."` | `import { Pluto } from "@/components"` |

3. **Usage patterns converted:**
   | Before | After |
   |--------|-------|
   | `<Table>` | `<Table.Table>` |
   | `<Diagram>` | `<Diagram.Diagram>` |
   | `<Rule>` | `<Rule.Rule>` |
   | `<StepText>` | `<Text.Step>` |
   | `<Release>` | `<Releases.Release>` |
   | `<SynnaxDownloadURL>` | `<Deploy.DownloadURL>` |
   | `<ExampleDetails>` | `<Details.Example>` |
   | `<ChannelTypeDetails>` | `<Details.ChannelType>` |

### Final component structure:

```
components/
├── index.ts                    # Exports ALL namespaces
├── analytics/
│   ├── index.ts + external.ts
│   └── PostHog.astro
├── code/
│   ├── index.ts + external.ts
│   ├── Block.astro             # Code.Block
│   ├── Inline.astro            # Code.Inline
│   └── codeUtil.ts
├── core/
│   ├── index.ts + external.ts
│   └── WindowsDownloadButton.astro
├── deploy/
│   ├── index.ts + external.ts
│   ├── SynnaxDownloadURL.astro # Deploy.DownloadURL
│   ├── SynnaxMoveCommand.astro # Deploy.MoveCommand
│   └── SynnaxVersionOutput.astro # Deploy.VersionOutput
├── details/
│   ├── index.ts + external.ts
│   ├── Details.astro           # Details.Details
│   ├── Example.astro           # Details.Example
│   └── ChannelType.astro       # Details.ChannelType
├── diagram/
│   ├── index.ts + external.ts
│   └── Diagram.astro           # Diagram.Diagram
├── releases/
│   ├── index.ts + external.ts
│   ├── Release.astro           # Releases.Release
│   ├── List.astro              # Releases.List
│   ├── List.tsx
│   ├── Tile.astro              # Releases.Tile
│   └── PolkaDot.astro
├── rule/
│   ├── index.ts + external.ts
│   └── Rule.astro              # Rule.Rule
├── text/
│   ├── index.ts + external.ts
│   ├── Text.tsx
│   └── Step.astro              # Text.Step
└── ... (other folders unchanged)
```

### Key achievement:

**Zero direct `.astro` imports in MDX files** - All components now use the unified namespace pattern:
```typescript
import { Table, Diagram, Rule, Text, Deploy, Details } from "@/components";
```

### Learnings:

1. **Astro components CAN be re-exported from TS** using `export { default as Name } from "...astro"`

2. **Namespace.Namespace is acceptable** for main components (e.g., `Diagram.Diagram`, `Rule.Rule`)

3. **Avoid redundant naming** - `Details.Example` not `Details.ExampleDetails`

4. **Single import source** - `@/components` is now the single source for all component imports

---

## Session 5: Astro-React Patterns & Performance

(Fill in after Session 4)

---

## Session 6: Code Quality & Final Review

(Fill in after Session 5)
