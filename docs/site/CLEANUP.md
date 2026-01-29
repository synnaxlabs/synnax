# Documentation Site Cleanup Plan

**THIS CLEANUP FOCUSES ON CODE QUALITY ONLY. DO NOT MODIFY DOCUMENTATION CONTENT (MDX FILES, PAGE TEXT, ETC.)**

This document tracks code quality improvements for the docs site. Focus is on code
organization, style conventions, and Astro-React patterns.

## Current State

**Stack:** Astro 5 + React 19 + TypeScript + Pluto UI library

---

## 0. Toolchain: Switch to `astro check`

### Current Issue

The docs site uses `tsc --noEmit` for type checking:
```json
"build": "tsc --noEmit && astro build",
"check-types": "tsc --noEmit"
```

**Problems:**
- `tsc` ignores `.astro` files entirely - no type checking on Astro components
- Cannot re-export `.astro` components from `.ts` barrel files (breaks `tsc`)
- Not following Astro's official recommendation

### Solution: Use `astro check`

Astro's official recommendation is to use `astro check` for type checking. It covers both
`.astro` AND `.ts`/`.tsx` files.

**Updated package.json:**
```json
{
  "scripts": {
    "build": "astro check && astro build",
    "check-types": "astro check"
  }
}
```

### Why This Matters

- **Complete coverage** - type checks `.astro` files (currently not checked at all)
- **Enables namespacing** - can re-export `.astro` components from `.ts` barrel files
- **Official recommendation** - aligns with Astro's intended toolchain
- **CI-friendly** - exits with code 1 on errors, designed for CI workflows

### Action Items

- [ ] Verify `@astrojs/check` is installed (should be with Astro 5)
- [ ] Update `build` script to `astro check && astro build`
- [ ] Update `check-types` script to `astro check`
- [ ] Test that all existing code passes `astro check`
- [ ] Update any CI/CD pipelines if needed

## 1. Style Organization

### Current Issues

- **Root.astro has ~420 lines of global CSS** - mix of:
  - Base resets and typography
  - Component-specific overrides (`.pluto-tree`, `.pluto-tabs-selector`, etc.)
  - Layout styles (`.page-nav`, `article`, `nav`)
  - Responsive breakpoints scattered throughout

- **Heavy `!important` usage** (~25 occurrences) - indicates specificity conflicts

- **No CSS organization pattern** - styles live in:
  - `Root.astro` (global)
  - `Header.astro` (component-specific but global scope)
  - `Article.astro` (component-specific but global scope)
  - Other component `.astro` files

### Solution: CSS Cascade Layers

Use CSS `@layer` to control cascade order and eliminate specificity battles. This is the
same approach used by Astro Starlight (Astro's official docs framework).

**Browser support:** Excellent - Chrome 99+, Firefox 97+, Safari 15.4+ (all since 2022,
~96% global coverage). See [Can I Use](https://caniuse.com/css-cascade-layers).

**Key rule:** Define layers in CSS files, import in Astro frontmatter (not in `<style>`
tags). There's a known Astro dev mode bug when using `@import` with layers inside
component `<style>` blocks.

### Proposed Structure: Hybrid Approach

**Centralize truly global styles in `src/styles/`:**
```
src/
├── styles/
│   ├── main.css          # Layer declarations and imports
│   ├── base.css          # Resets, typography, CSS variable overrides
│   ├── layout.css        # Header, footer, nav, article structure
│   └── pluto.css         # Pluto component overrides (global)
```

**Keep component-specific styles co-located:**
```astro
<!-- Header.astro -->
<header>...</header>

<style>
  /* Scoped styles for this component only */
  header { ... }
</style>
```

| Style Type | Location | Why |
|------------|----------|-----|
| Base/resets/typography | `src/styles/base.css` | Global by nature |
| Pluto overrides | `src/styles/pluto.css` | Needs layers to control cascade |
| Layout structure | `src/styles/layout.css` | Shared across pages |
| Component-specific | In `.astro` component | Scoped, co-located |

**main.css:**
```css
/* Declare layer order - later layers win */
@layer base, pluto, layout;

/* Import Pluto CSS into its own layer */
@import "@synnaxlabs/pluto/dist/pluto.css" layer(pluto);
@import "@synnaxlabs/pluto/dist/theme.css" layer(pluto);
@import "@synnaxlabs/media/dist/media.css" layer(pluto);

/* Import global docs site styles (CSS @import uses relative paths for same-folder files) */
@import "./base.css" layer(base);
@import "./layout.css" layer(layout);
@import "./pluto.css" layer(pluto);  /* Pluto overrides in same layer, after */
```

**Note:** CSS `@import` statements use file-relative paths (standard CSS behavior).
TypeScript/JavaScript imports should always use absolute `@/` paths.

**Root.astro:**
```astro
---
// Import in frontmatter, NOT in <style> tag
import "@/styles/main.css";
import "@fontsource/geist-mono";
---
```

**Component with scoped styles:**
```astro
<!-- Header.astro -->
<style>
  /* These are automatically scoped to Header */
  .header-content { ... }

  /* Use :global() only when targeting Pluto children */
  :global(.pluto-btn) { ... }
</style>
```

### Why Layers Work

- **No more `!important`** - a simple selector in a later layer beats a complex selector
  in an earlier layer
- **Pluto stays untouched** - import Pluto CSS into its own layer, override cleanly in
  higher layers
- **Clear mental model** - you know exactly why a style wins
- **Gradual migration** - unlayered CSS wins by default, so existing code keeps working

### Action Items

- [ ] Create `src/styles/` directory with `main.css`, `base.css`, `layout.css`, `pluto.css`
- [ ] Set up CSS layers in `main.css`
- [ ] Extract base/reset styles from Root.astro into `base.css`
- [ ] Extract layout styles into `layout.css`
- [ ] Extract Pluto overrides into `pluto.css`
- [ ] Update Root.astro to import `main.css` in frontmatter
- [ ] Convert `<style is:global>` to scoped `<style>` where possible
- [ ] Use `:global()` selector only for targeting Pluto children
- [ ] Audit and eliminate `!important` usage (layers should make this unnecessary)
- [ ] Test in dev mode to verify no Astro layer bugs

---

## 2. Component Organization

### Current Issues

**Inconsistent folder structure:**
```
components/
├── Table.tsx              # Top-level
├── Table.astro            # Top-level (same name, different type)
├── Tabs.tsx               # Top-level
├── PageNav/               # Nested folder
│   ├── PageNav.astro      # Wrapper that just imports .tsx
│   └── PageNav.tsx        # Actual component
├── platform/              # Nested folder (lowercase)
│   ├── index.ts           # Re-exports external.ts
│   ├── external.ts        # Actual exports
│   ├── platform.tsx       # Core logic (lowercase)
│   └── Tabs.tsx           # Component (PascalCase)
```

**Problems:**
- Mix of flat and nested organization
- Inconsistent naming (PascalCase vs lowercase)
- Some redundant Astro wrappers

### Namespace Module Pattern

Use the namespace pattern consistently across all component groupings. This enables clean
imports like `import { Nav } from "@/components/nav"` and usage like `Nav.Page`.

**Pattern structure (matches Pluto codebase):**
```typescript
// nav/Page.tsx - component file with named exports
export interface PageProps { ... }
export const Page: FC<PageProps> = () => { ... }

// nav/external.ts - re-exports everything from each file
export * from "@/components/nav/Page";
export * from "@/components/nav/OnThisPage";
export * from "@/components/nav/Breadcrumb";

// nav/index.ts - creates the namespace
export * as Nav from "@/components/nav/external";

// Usage
import { Nav } from "@/components/nav";
<Nav.Page />
<Nav.Breadcrumb />
```

**Why `external.ts`?** This pattern enables `import { Nav }` (named import) rather than
`import * as Nav` (namespace import). The latter is less explicit about what's imported.

### Import/Export Strategy (CRITICAL)

This matches the Pluto codebase pattern exactly. **Follow these rules strictly:**

#### Rule 1: Always Use Absolute Imports
```typescript
// ✅ Correct
import { Nav } from "@/components/nav";
export * from "@/components/nav/Page";

// ❌ Wrong - NEVER use relative imports
import { Nav } from "../nav";
export * from "./Page";
```

**Only exception:** CSS `@import` uses relative paths (CSS has no alias support).

#### Rule 2: Never Use Default Exports
```typescript
// ✅ Correct - named exports only
export const Page: FC<PageProps> = () => { ... };
export interface PageProps { ... }

// ❌ Wrong - no default exports
export default function Page() { ... }
```

#### Rule 3: Use `export *` in Barrel Files
```typescript
// ✅ Correct - external.ts re-exports everything
export * from "@/components/nav/Page";
export * from "@/components/nav/Breadcrumb";

// ❌ Wrong - don't cherry-pick exports
export { Page } from "@/components/nav/Page";
export { Breadcrumb } from "@/components/nav/Breadcrumb";
```

#### Rule 4: Use `export * as` for Namespaces
```typescript
// ✅ Correct - index.ts creates namespace
export * as Nav from "@/components/nav/external";

// ❌ Wrong
export { Nav } from "@/components/nav/external";
```

#### Rule 5: Root Barrel Imports Namespaces
```typescript
// ✅ Correct - components/index.ts imports namespaces
export { Nav } from "@/components/nav";
export { Article } from "@/components/article";

// This works because nav/index.ts does: export * as Nav from ...
```

#### Complete Example

```
components/nav/
├── Page.tsx          # export const Page = ...; export interface PageProps = ...
├── Breadcrumb.tsx    # export const Breadcrumb = ...
├── external.ts       # export * from "@/components/nav/Page";
│                     # export * from "@/components/nav/Breadcrumb";
└── index.ts          # export * as Nav from "@/components/nav/external";

components/
└── index.ts          # export { Nav } from "@/components/nav";
                      # export { Article } from "@/components/article";
```

**Usage:**
```typescript
import { Nav, Article } from "@/components";
<Nav.Page />
<Article.Article />
```

### Proposed Structure

```
components/
├── article/
│   ├── index.ts           # export * as Article from "@/components/article/external"
│   ├── external.ts        # export * from "@/components/article/Article"
│   ├── Article.astro
│   └── NextPrev.astro
├── code/
│   ├── index.ts
│   ├── external.ts
│   ├── Block.astro        # Rename from Code.astro
│   ├── Inline.astro       # Rename from CodeI.astro
│   └── utils.ts
├── nav/
│   ├── index.ts
│   ├── external.ts
│   ├── Page.tsx           # Rename from PageNav.tsx
│   ├── OnThisPage.tsx
│   └── Breadcrumb.tsx
├── platform/
│   ├── index.ts
│   ├── external.ts
│   ├── Platform.tsx       # Rename from platform.tsx
│   ├── Tabs.tsx
│   └── SelectButton.tsx
├── client/
│   ├── index.ts
│   ├── external.ts
│   ├── Client.tsx         # Rename from client.tsx
│   ├── Tabs.tsx
│   └── SelectButton.tsx
├── search/
│   ├── index.ts
│   ├── external.ts
│   ├── Search.tsx
│   └── Search.astro       # If needed for SSR wrapper
├── feedback/
│   ├── index.ts
│   ├── external.ts
│   ├── Button.astro
│   └── Form.tsx
├── release/
│   ├── index.ts
│   ├── external.ts
│   ├── List.tsx
│   └── Tile.astro
├── layout/
│   ├── index.ts
│   ├── external.ts
│   ├── Header.astro
│   ├── Footer.astro
│   └── Root.astro
├── tabs/
│   ├── index.ts
│   ├── external.ts
│   └── Tabs.tsx           # Tabs.Tabs
├── table/
│   ├── index.ts
│   ├── external.ts
│   └── Table.tsx          # Table.Table
└── text/
    ├── index.ts
    ├── external.ts
    └── Text.tsx           # Text.Text
```

### Naming Within Namespaces (Go-style)

Avoid redundancy - the namespace provides context:
- ✅ `Nav.Page` not `Nav.PageNav`
- ✅ `Code.Block` not `Code.CodeBlock`
- ✅ `Tabs.Tabs` is OK (main component matches namespace, like Go's `tabs.Tabs`)
- ✅ `Table.Table`, `Text.Text`, `Article.Article` - all fine
- ❌ `Tabs.TabSelector` - redundant, should be `Tabs.Selector`

### Action Items

- [ ] Apply namespace pattern to all component folders (including shared components)
- [ ] Add `index.ts` + `external.ts` to each folder
- [ ] Rename components to avoid namespace redundancy (e.g., `Nav.Page` not `Nav.PageNav`)
- [ ] `Namespace.Namespace` is OK for main components (e.g., `Tabs.Tabs`, `Table.Table`)
- [ ] Update all imports throughout codebase

---

## 3. Astro-React Integration

### Current Patterns

**Good:**
- `client:only="react"` for interactive components
- `transition:persist` for header across navigations
- Fragment slots for tabbed content in MDX

**Issues:**

1. **Inconsistent client directive usage:**
   `client:only="react"` was added reactively to fix issues, not based on clear principles.

2. **Polling for URL state sync (Tabs.tsx:41-46):**
   ```typescript
   const i = setInterval(() => {
     const url = new URL(window.location.href);
     setSelected(url.searchParams.get(queryParamKey) ?? tabs[0].tabKey);
   }, 200);
   ```
   Should use `popstate` event listener instead.

3. **Inline script in Root.astro:**
   Large script block with multiple initialization functions could be a module.

### Client Directive Guidelines

| Scenario | Pattern | Why |
|----------|---------|-----|
| Needs React hooks/state | `client:only="react"` | React lifecycle required |
| Needs browser APIs on load (localStorage, window) | `client:only="react"` | Not available during SSR |
| Uses Pluto interactive components (Tabs, Dialog, Form) | `client:only="react"` | Pluto assumes React context |
| Pure display from props | No directive (SSR) | Static HTML is faster |
| Simple DOM manipulation (copy button, scroll) | `<script>` tag | No React overhead needed |
| Needs to work without JS | Astro SSR | Progressive enhancement |

**Rule of thumb:**
- Default to **no directive** (SSR) unless you have a reason
- Use **`client:only="react"`** when: React hooks, browser APIs, or Pluto interactive components
- Use **`<script>`** for: Simple DOM interactions that don't need React

**Avoid:**
- `client:load` / `client:visible` with Pluto components (hydration issues)
- Mixing React state with vanilla JS scripts for the same feature

### React Context Isolation

**Critical:** Each `client:only="react"` component is its own isolated React island. They
do NOT share React context with each other.

**Implications:**
- Cannot use shared React context providers across islands
- Each island has its own state - no automatic sync between components
- If components need shared state, options are:
  - Combine them into a single island
  - Use URL query params (current approach for Platform/Client tabs)
  - Use browser storage (localStorage/sessionStorage)
  - Use a global event bus pattern

**Current workaround in codebase:** Platform and Client tabs sync via URL query params,
which works across islands but uses polling (should switch to `popstate` events).

### When to Use Astro vs React

**Use Astro (.astro) for:**
- Static content and layouts
- Server-rendered markup
- Components that don't need client interactivity
- Wrapping React components with server-side props

**Use React (.tsx) for:**
- Interactive UI with state
- Components using React hooks
- Pluto component compositions
- Complex event handling

### Action Items

- [ ] Audit all components for correct client directive usage
- [ ] Replace URL polling with `popstate` event listener
- [ ] Extract Root.astro inline script to module
- [ ] Document these guidelines in codebase

---

## 4. Naming Conventions

### Current State

| Pattern | Examples | Issue |
|---------|----------|-------|
| Folders | `PageNav/`, `platform/` | Mixed case |
| Files | `Tabs.tsx`, `platform.tsx` | Mixed case |
| Components | `PageNav`, `textFactory` | Mixed patterns |

### Proposed Conventions

**Folders:**
- Lowercase, no hyphens for single words: `nav/`, `platform/`, `article/`
- Represents the namespace name (lowercase version)

**Component files (.tsx, .astro):**
- PascalCase: `Page.tsx`, `Tabs.tsx`, `Header.astro`
- Name should be semantic within namespace context (avoid redundancy)

**Barrel files:**
- `index.ts` - creates namespace: `export * as Nav from "@/components/nav/external"`
- `external.ts` - re-exports from files: `export * from "@/components/nav/Page"`

**Utility files (.ts):**
- camelCase: `utils.ts`, `codeUtils.ts`
- Or descriptive: `platform.ts` for platform detection logic

**Single entry point** - `components/index.ts` re-exports all namespaces:
```typescript
// components/index.ts
export { Article } from "@/components/article";
export { Client } from "@/components/client";
export { Code } from "@/components/code";
export { Nav } from "@/components/nav";
export { Platform } from "@/components/platform";
// etc.
```

**Usage in MDX/Astro:**
```typescript
// Preferred - single import
import { Platform, Client, Nav } from "@/components";

// Also OK - individual namespace imports
import { Platform } from "@/components/platform";

// NEVER - relative imports
import { Platform } from "../components/platform";  // ❌

// NEVER - direct component imports
import Tabs from "@/components/platform/Tabs";  // ❌
```

**Namespace usage - no redundancy:**
```
Nav.Page          // not Nav.PageNav
Code.Block        // not Code.CodeBlock
Platform.Tabs     // clear and semantic
Article.Article   // OK when component IS the concept
```

### Import Rules

See **Import/Export Strategy** section above for complete rules. Summary:

- **Always absolute imports** (`@/`) - never relative imports in TypeScript/JavaScript
- **CSS exception** - `@import` uses relative paths (CSS has no alias support)
- **Named exports only** - never use default exports
- **`export *` in barrel files** - don't cherry-pick

```typescript
// ✅ Correct
import { Platform } from "@/components/platform";
export * from "@/components/nav/Page";

// ❌ Wrong - relative imports
import { Platform } from "../platform";
import { someUtil } from "../../util/helpers";
export * from "./Page";  // ❌ even in barrel files
```

### Action Items

- [ ] Rename all folders to lowercase
- [ ] Rename all component files to PascalCase
- [ ] Ensure consistent `index.ts` + `external.ts` pattern
- [ ] Create `components/index.ts` as single entry point
- [ ] Update all imports to use absolute paths
- [ ] Document conventions in a CONTRIBUTING.md or similar

---

## 5. MDX File Conventions

### Structure

MDX files are strictly for **content** - no component logic.

```mdx
---
layout: "@/layouts/Reference.astro"
title: "Page Title"
description: "Page description"
---
import { Platform, Client } from "@/components";
import { Divider, Note } from "@synnaxlabs/pluto";
import { mdxOverrides } from "@/components/mdxOverrides";
export const components = mdxOverrides;

Content goes here...

<Platform.Tabs client:only="react">
  <Fragment slot="Linux">Linux content</Fragment>
  <Fragment slot="macOS">macOS content</Fragment>
</Platform.Tabs>
```

### Rules

1. **No component logic** - MDX files only import and use components
2. **Absolute imports only** - use `@/components`, never relative paths
3. **Namespace imports** - use `import { Platform } from "@/components"`, not direct imports
4. **Client directives** - use `client:only="react"` for interactive Pluto/React components
5. **Fragment slots** - use `<Fragment slot="name">` for tabbed content

### Client Directive Consistency

Currently inconsistent (`client:load` vs `client:only="react"`). Standardize to:
- **`client:only="react"`** for all React/Pluto interactive components

### Action Items

- [ ] Audit MDX files for consistent client directive usage
- [ ] Update any relative imports to absolute
- [ ] Ensure all interactive components use `client:only="react"`

---

## 6. Code Quality

### Issues

1. **Repeated patterns** - Platform and Client tabs are nearly identical
2. **Magic numbers** - Hard-coded breakpoints (800px, 1100px, 600px)
3. **Type safety** - Some `any` types in component props
4. **Polling instead of events** - URL state sync uses `setInterval` instead of `popstate`

### Action Items

- [ ] Extract shared tab logic into generic hook/component
- [ ] Define breakpoint constants as CSS custom properties or JS constants
- [ ] Improve type definitions (remove `any` types)
- [ ] Replace URL polling with `popstate` event listener

---

## Implementation Order

1. **Phase 0: Toolchain** - Switch to `astro check` (enables later phases)
2. **Phase 1: CI/CD** - Add validation workflow (catches regressions early)
3. **Phase 2: Folder structure** - Create `styles/`, `assets/` directories
4. **Phase 3: Styles** - CSS layers, extract from components
5. **Phase 4: Component organization** - Namespace pattern, reorganize folders
6. **Phase 5: Naming** - Rename files/folders, update imports
7. **Phase 6: MDX standardization** - Consistent imports and client directives
8. **Phase 7: Astro-React patterns** - Audit client directives, fix polling
9. **Phase 8: Performance** - Optimize client directives, lazy loading audit
10. **Phase 9: Code quality** - Ongoing refinement

---

## 7. Folder Structure

### Current Structure

```
src/
├── components/     # All components (flat + nested mix)
├── layouts/        # Astro layouts
├── pages/          # MDX content pages
└── util/           # Utilities
```

### Recommended Astro Structure

Based on [Astro's official docs](https://docs.astro.build/en/basics/project-structure/):

```
src/
├── assets/         # Images, fonts (processed by Astro)
├── components/     # Reusable components (with namespace pattern)
├── layouts/        # Page layouts
├── pages/          # Routes and content
├── styles/         # Global CSS (new - for CSS layers)
└── util/           # Utilities and helpers
```

### Changes Needed

- [x] `components/` - exists, needs reorganization (see Section 2)
- [x] `layouts/` - exists, OK
- [x] `pages/` - exists, OK
- [ ] `styles/` - **create** for CSS layer organization
- [ ] `assets/` - **create** for processed images/fonts (currently in `public/`)
- [x] `util/` - exists, OK

### Action Items

- [ ] Create `src/styles/` directory for CSS layers
- [ ] Create `src/assets/` for images that need processing
- [ ] Move appropriate assets from `public/` to `src/assets/`

---

## 8. CI/CD Validation

### Current State

- `deploy.docs.yaml` - only updates Algolia search index on push to main
- ❌ No `test.docs.yaml` workflow (unlike console, pluto, etc.)
- ❌ No type checking, lint, or build validation on PRs

### Repo Pattern

Following the established pattern (`test.console.yaml`, `test.pluto.yaml`):
- `test.*.yaml` - runs on PRs AND push to main/rc (validation)
- `deploy.*.yaml` - runs on push to main only (deployment tasks)

### Recommended: Create `test.docs.yaml`

```yaml
# .github/workflows/test.docs.yaml
name: Test - Docs

on:
  pull_request:
    paths:
      - .github/workflows/test.docs.yaml
      - alamos/ts/**
      - client/ts/**
      - configs/ts/**
      - configs/vite/**
      - docs/site/**
      - freighter/ts/**
      - package.json
      - pluto/**
      - pnpm-lock.yaml
      - pnpm-workspace.yaml
      - turbo.json
      - x/media/**
      - x/ts/**
  push:
    branches:
      - main
      - rc
    paths:
      # Same paths as above
  workflow_dispatch:

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    env:
      TURBO_TOKEN: ${{ secrets.TURBO_TOKEN }}
      TURBO_TEAM: ${{ vars.TURBO_TEAM }}
    steps:
      - name: Checkout Repository
        uses: actions/checkout@v6

      - name: Set up pnpm
        uses: pnpm/action-setup@v4

      - name: Set up Node.js
        uses: actions/setup-node@v6
        with:
          node-version-file: package.json
          cache: pnpm

      - name: Install Dependencies
        run: pnpm install

      - name: Check Types
        working-directory: docs/site
        run: pnpm check-types

      - name: Lint
        working-directory: docs/site
        run: pnpm lint

      - name: Build
        working-directory: docs/site
        run: pnpm build
```

### Action Items

- [ ] Create `test.docs.yaml` workflow following repo pattern
- [ ] Match path triggers with `deploy.docs.yaml`
- [ ] Add type checking (`astro check`)
- [ ] Add lint checking
- [ ] Add build validation
- [ ] Consider adding Lighthouse CI for performance monitoring

---

## 9. Performance Optimization

### Astro Performance Features

Astro provides excellent performance by default through:
- **Zero JS by default** - only ships JS for interactive islands
- **Islands Architecture** - partial hydration for interactivity
- **Built-in image optimization** - via `<Image />` component

### Current Performance Considerations

1. **Client directives** - need audit for optimal loading strategy:
   - `client:load` - loads immediately (current usage)
   - `client:idle` - loads when main thread is free
   - `client:visible` - loads when entering viewport (lazy)

2. **Image optimization** - verify using Astro's `<Image />` component

3. **Bundle size** - Pluto is a large library; ensure tree-shaking works

### Lazy Loading Strategy

| Component Type | Directive | Rationale |
|----------------|-----------|-----------|
| Above-the-fold interactive | `client:load` | Needs immediate interactivity |
| Below-the-fold interactive | `client:visible` | Lazy load when scrolled into view |
| Non-critical interactive | `client:idle` | Load when browser is idle |
| Search modal | `client:idle` | Not needed immediately |
| Feedback form | `client:visible` | Only when user scrolls to it |

### Media/Image Handling

**Current approach:** Custom `<Image>` and `<Video>` components in `Media.tsx` that:
- Load from DigitalOcean CDN (`synnax.nyc3.cdn.digitaloceanspaces.com`)
- Support themed variants (dark/light based on `prefers-color-scheme`)
- Use `client:only="react"` for live theme switching
- Use plain `<img>` tag (no Astro optimization)

**This is correct for CDN images** - Astro's `<Image />` would double-process CDN images,
which is not recommended. The CDN already handles optimization.

**Potential improvements:**
- Add `loading="lazy"` attribute for below-fold images
- Add explicit `width`/`height` props to prevent CLS (Cumulative Layout Shift)
- Add `decoding="async"` for non-blocking decode
- Consider `IntersectionObserver` for lazy loading (Video already has this)

### Action Items

- [ ] Audit all `client:*` directives for optimal loading
- [ ] Consider `client:visible` for below-fold components
- [ ] Consider `client:idle` for non-critical components (search, feedback)
- [ ] Add `loading="lazy"` to Image component for below-fold images
- [ ] Add `width`/`height` props to Image component to prevent CLS
- [ ] Run Lighthouse audit and document baseline
- [ ] Consider Lighthouse CI integration

---

## Notes

- All changes should maintain existing functionality
- Run `astro check && pnpm build` after each phase to verify no regressions
- Documentation content (MDX files) should NOT be modified
- MDX imports may need updating when component paths change

### Pluto Component Quirks

Some Pluto components require extra care:
- **Tabs** - Generally quirky, heavy CSS overrides needed
- **Tree** - Complex state management, careful with styling

These components may need more extensive CSS layer overrides and should be tested
thoroughly after style changes.

---

## Implementation Sessions

Six focused sessions to implement this cleanup. Each session builds on the previous.
At the end of each session, document learnings in a `SESSION_NOTES.md` file to inform
the next session.

### Session 1: Toolchain & CI/CD (Phases 0-1)

```
Read docs/site/CLEANUP.md for full context.

Implement Phase 0 (Toolchain) and Phase 1 (CI/CD):

PHASE 0 - Switch to astro check:
1. Verify @astrojs/check is installed
2. Update package.json scripts:
   - "build": "astro check && astro build"
   - "check-types": "astro check"
3. Run `astro check` and fix any type errors
4. Verify build still works

PHASE 1 - Create test.docs.yaml:
1. Create .github/workflows/test.docs.yaml following the pattern in test.console.yaml
2. Trigger on pull_request AND push to main/rc
3. Match path triggers with deploy.docs.yaml
4. Include steps: checkout, pnpm setup, node setup, install, check-types, lint, build
5. Test workflow locally if possible

After completing:
- Document any issues encountered
- Note any type errors found by astro check that tsc missed
- Record in docs/site/SESSION_NOTES.md for next session
```

### Session 2: Folder Structure & Styles (Phases 2-3)

```
Read docs/site/CLEANUP.md and docs/site/SESSION_NOTES.md for context.

Implement Phase 2 (Folder Structure) and Phase 3 (Styles):

PHASE 2 - Create folder structure:
1. Create src/styles/ directory
2. Create src/assets/ directory
3. Move any processable assets from public/ to src/assets/ if appropriate

PHASE 3 - CSS Layers (Hybrid Approach):
1. Create src/styles/main.css with layer declarations:
   @layer base, pluto, layout;
2. Import Pluto CSS into pluto layer
3. Extract base/reset styles from Root.astro → src/styles/base.css
4. Extract layout styles → src/styles/layout.css
5. Extract Pluto overrides → src/styles/pluto.css
6. Update Root.astro to import main.css in frontmatter
7. Convert <style is:global> to scoped <style> where possible
8. Use :global() only for Pluto children
9. Audit and eliminate !important usage
10. Test thoroughly - layers can break in subtle ways

Key rule: Import CSS in Astro frontmatter, NOT in <style> tags (Astro dev mode bug).

After completing:
- Document which styles stayed global vs became scoped
- Note any !important that couldn't be removed and why
- Record Pluto components that needed special handling
- Update SESSION_NOTES.md
```

### Session 3: Component Organization (Phase 4)

```
Read docs/site/CLEANUP.md and docs/site/SESSION_NOTES.md for context.

Implement Phase 4 (Component Organization):

Apply namespace pattern to ALL component folders:

Pattern for each folder (matches Pluto pattern):
  folder/
  ├── index.ts      # export * as Namespace from "@/components/folder/external"
  ├── external.ts   # export * from "@/components/folder/ComponentA"
  └── ComponentA.tsx  # export const ComponentA = ...

Folders to create/reorganize:
- article/ (Article.astro, NextPrev.astro)
- code/ (Block.astro, Inline.astro, utils.ts)
- nav/ (Page.tsx, OnThisPage.tsx, Breadcrumb.tsx)
- platform/ (already exists, verify pattern)
- client/ (already exists, verify pattern)
- search/ (Search.tsx, Search.astro if needed)
- feedback/ (Button.astro, Form.tsx)
- release/ (List.tsx, Tile.astro)
- layout/ (Header.astro, Footer.astro, Root.astro)
- tabs/ (Tabs.tsx)
- table/ (Table.tsx, Table.astro)
- text/ (Text.tsx)
- media/ (Image, Video from Media.tsx)

Create components/index.ts as single entry point:
  export { Article } from "@/components/article";
  export { Code } from "@/components/code";
  // etc.

Naming rules:
- Namespace.Namespace is OK (Tabs.Tabs, Table.Table)
- Avoid redundancy (Nav.Page not Nav.PageNav)

After completing:
- Document any circular dependency issues
- Note components that were tricky to namespace
- List any Astro components that couldn't be re-exported (if any)
- Update SESSION_NOTES.md
```

### Session 4: Naming & MDX (Phases 5-6)

```
Read docs/site/CLEANUP.md and docs/site/SESSION_NOTES.md for context.

Implement Phase 5 (Naming) and Phase 6 (MDX Standardization):

PHASE 5 - Naming conventions:
1. Rename folders to lowercase (PageNav/ → nav/, OnThisPage/ → part of nav/)
2. Rename component files to PascalCase
3. Rename to avoid namespace redundancy:
   - PageNav.tsx → Page.tsx (in nav/)
   - CodeI.astro → Inline.astro (in code/)
4. Update ALL imports to absolute paths (@/components/...)
5. Run astro check after each major rename to catch broken imports

PHASE 6 - MDX Standardization:
1. Audit all MDX files for client directive usage
2. Replace client:load with client:only="react" for Pluto/React components
3. Convert imports to namespace pattern:
   - Before: import Tabs from "@/components/platform/Tabs"
   - After: import { Platform } from "@/components"
4. Remove any relative imports
5. Verify Fragment slot pattern is consistent

Use grep/search to find:
- All client:load usages
- All relative imports in MDX
- All direct component imports (not namespace)

After completing:
- Document any MDX files that needed special handling
- Note any client:load that couldn't be converted and why
- Record import patterns that were problematic
- Update SESSION_NOTES.md
```

### Session 5: Astro-React Patterns & Performance (Phases 7-8)

```
Read docs/site/CLEANUP.md and docs/site/SESSION_NOTES.md for context.

Implement Phase 7 (Astro-React Patterns) and Phase 8 (Performance):

PHASE 7 - Astro-React patterns:
1. Audit all components for correct client directive usage:
   - client:only="react" for React hooks, browser APIs, Pluto interactive
   - No directive for pure SSR
   - <script> for simple DOM manipulation
2. Replace URL polling in Tabs.tsx with popstate event:
   - Remove setInterval polling
   - Add window.addEventListener("popstate", ...)
   - Also listen for custom events if tabs change URL
3. Extract Root.astro inline script to module:
   - Create src/util/init.ts or similar
   - Move updateVersion, bindClickHandlers, etc.
   - Import and call from Root.astro

PHASE 8 - Performance:
1. Audit client:* directives for optimal loading:
   - client:visible for below-fold components
   - client:idle for non-critical (search, feedback)
2. Update Media.tsx Image component:
   - Add loading="lazy" prop support
   - Add width/height props to prevent CLS
   - Add decoding="async"
3. Run Lighthouse audit and document baseline scores
4. Identify any major performance issues

After completing:
- Document performance baseline (Lighthouse scores)
- Note any components that couldn't use optimal directives
- Record any React context issues discovered
- Update SESSION_NOTES.md
```

### Session 6: Code Quality & Final Review (Phase 9)

```
Read docs/site/CLEANUP.md and docs/site/SESSION_NOTES.md for context.

Implement Phase 9 (Code Quality) and final review:

PHASE 9 - Code quality:
1. Extract shared Platform/Client tab logic:
   - Identify duplicated code between platform/Tabs.tsx and client/Tabs.tsx
   - Create shared hook or base component
   - Refactor both to use shared code
2. Define breakpoint constants:
   - Create src/styles/breakpoints.css or add to base.css
   - Define CSS custom properties: --breakpoint-sm, --breakpoint-md, etc.
   - Or create src/util/breakpoints.ts for JS usage
   - Replace magic numbers (800px, 1100px, 600px)
3. Improve type definitions:
   - Find and fix any `any` types
   - Add proper types to component props
   - Ensure strict TypeScript compliance

FINAL REVIEW:
1. Run full build: astro check && pnpm build
2. Test site locally - navigate through pages
3. Verify all imports work
4. Check for console errors
5. Run Lighthouse again - compare to baseline
6. Update CLEANUP.md with completion status
7. Create summary of all changes made

Deliverables:
- All phases complete
- SESSION_NOTES.md with learnings from all sessions
- Updated CLEANUP.md marking completed items
- Performance comparison (before/after Lighthouse)
```

---

## Session Notes Template

Create `docs/site/SESSION_NOTES.md` at the start of Session 1:

```markdown
# Cleanup Session Notes

## Session 1: Toolchain & CI/CD
**Date:**
**Completed:** [ ] Phase 0, [ ] Phase 1

### What was done:

### Issues encountered:

### Learnings for next session:

---

## Session 2: Folder Structure & Styles
(Fill in after Session 1)

---

(Continue for each session)
```
