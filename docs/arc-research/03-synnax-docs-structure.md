# Synnax Documentation Structure

Research on where Synnax documentation is written, how it's structured, and where Arc
documentation should fit.

---

## 1. Documentation Location and Framework

### Location

**Path**: `/docs/site/`

### Framework

- **Astro 5.16.5** with MDX support
- **@astrojs/mdx** for Markdown + React components
- **React 19** for interactive components
- **Vercel** deployment via `@astrojs/vercel` adapter
- **Shiki** syntax highlighting with custom Arc grammar already integrated

### Build Commands

```bash
cd docs/site
pnpm dev         # Development server
pnpm build       # Production build
pnpm check-types # Type checking
```

### Hosting

- **URL**: docs.synnaxlabs.com
- **Search**: Algolia integration

---

## 2. Directory Structure

```
/docs/site/src/
├── pages/
│   ├── reference/              # Main documentation
│   │   ├── _nav.ts            # Navigation config
│   │   ├── index.mdx          # Landing page
│   │   ├── concepts/          # Core concepts
│   │   ├── core/              # Server docs
│   │   ├── client/            # Client libraries
│   │   ├── control/           # Control sequences ← ARC GOES HERE
│   │   │   ├── _nav.ts
│   │   │   ├── index.astro
│   │   │   ├── python/        # Python sequences
│   │   │   └── embedded/      # Lua sequences
│   │   ├── console/           # Console app
│   │   ├── driver/            # Hardware driver
│   │   └── pluto/             # Visualization
│   ├── guides/                # User guides by role
│   ├── releases/              # Release notes
│   └── _nav.ts               # Root navigation
├── layouts/
│   └── Reference.astro        # Reference page layout
├── components/
│   └── ...                    # Reusable UI components
└── util/
```

---

## 3. Existing Reference Sections

Defined in `/docs/site/src/pages/_nav.ts`:

| Section     | Path                   | Description                      |
| ----------- | ---------------------- | -------------------------------- |
| Get Started | `/reference/`          | Introduction and setup           |
| Concepts    | `/reference/concepts/` | Overview, Channels, Ranges, etc. |
| Core        | `/reference/core/`     | Server installation              |
| Client      | `/reference/client/`   | Python & TypeScript clients      |
| **Control** | `/reference/control/`  | **Control sequences**            |
| Console     | `/reference/console/`  | Desktop application              |
| Driver      | `/reference/driver/`   | Hardware integration             |
| Pluto       | `/reference/pluto/`    | Visualization components         |

---

## 4. Control Section (Where Arc Fits)

The Control section currently has:

```
/reference/control/
├── _nav.ts                # Navigation
├── index.astro           # Redirect to Python
├── python/               # Python Sequences
│   ├── _nav.ts
│   ├── index.astro
│   ├── get-started.mdx
│   ├── set-and-read-channels.mdx
│   ├── add-commands-and-control-logic.mdx
│   └── ...
└── embedded/             # Embedded Sequences (Lua)
    ├── _nav.ts
    ├── index.astro
    ├── get-started.mdx
    ├── language-basics.mdx
    └── ...
```

### Why Arc Fits in Control

1. **Same purpose** - Control sequences for hardware automation
2. **Parallel structure** - Python and Lua sequences already here
3. **User expectation** - Control engineers look here for automation tools
4. **Natural hierarchy** - Arc is another control language option

---

## 5. Page Format

### Directory Pattern

```
reference/[section]/
├── _nav.ts              # Navigation definition
├── index.astro          # Index page (often redirects)
├── get-started.mdx      # First page
├── [topic].mdx          # Topic pages
└── [subsection]/
    ├── _nav.ts
    └── [topic].mdx
```

### Navigation Definition (`_nav.ts`)

```typescript
import type { PageNavNode } from "@/components/PageNav";

export const SECTION_NAV: PageNavNode = {
  key: "section-key",
  name: "Display Name",
  children: [
    {
      key: "/reference/section/page",
      href: "/reference/section/page",
      name: "Page Title",
    },
    {
      key: "subsection",
      name: "Subsection Name",
      children: [
        // nested pages
      ],
    },
  ],
};
```

### MDX Page Format

```mdx
---
layout: "@/layouts/Reference.astro"
title: "Page Title"
description: "Brief description"
next: "Next Page Title"
nextURL: "/reference/section/next-page"
prev: "Previous Page Title"
prevURL: "/reference/section/prev-page"
---

import { Divider, Note } from "@synnaxlabs/pluto";
import { Image, Video } from "@/components/Media";
import { mdxOverrides } from "@/components/mdxOverrides";
export const components = mdxOverrides;

# Page Title

Content here...

<Divider.Divider x />

## Section Heading

More content...

<Note.Note variant="info">Helpful tip or callout.</Note.Note>
```

### Available Components

- **Pluto**: `Divider`, `Note`, `Button`, etc.
- **Custom**: `Image`, `Video`, `Diagram`, `Table`, `Tabs`, `PackageManagerTabs`
- **Standard**: Markdown, code blocks, lists, tables

---

## 6. Arc Syntax Highlighting (Already Configured)

Arc syntax highlighting is already set up in `/docs/site/astro.config.ts`:

```typescript
import { grammar as arcGrammar } from "@synnaxlabs/arc";

export default defineConfig({
  markdown: {
    shikiConfig: {
      langs: [arcGrammar], // Arc syntax highlighting ready
    },
  },
});
```

Code blocks use triple backticks with `arc`:

````markdown
```arc
func add(x f64, y f64) f64 {
    return x + y
}
```
````

---

## 7. Proposed Arc Documentation Structure

### Location

**Path**: `/docs/site/src/pages/reference/control/arc/`

### Directory Structure

```
reference/control/arc/
├── _nav.ts                    # Navigation
├── index.astro               # Redirect to get-started
├── get-started.mdx           # Introduction & first program
├── tutorials/
│   ├── _nav.ts
│   ├── basic-calculations.mdx
│   ├── stateful-programs.mdx
│   ├── control-sequences.mdx
│   └── multi-stage-automation.mdx
├── concepts/
│   ├── _nav.ts
│   ├── reactive-execution.mdx
│   ├── channels-and-series.mdx
│   ├── sequences-and-stages.mdx
│   └── types-and-units.mdx
├── language-reference/
│   ├── _nav.ts
│   ├── syntax.mdx
│   ├── types.mdx
│   ├── operators.mdx
│   ├── functions.mdx
│   ├── sequences.mdx
│   └── built-ins.mdx
├── how-to/
│   ├── _nav.ts
│   ├── create-alarms.mdx
│   ├── conditional-routing.mdx
│   ├── error-handling.mdx
│   └── debugging.mdx
└── examples/
    ├── _nav.ts
    ├── calculations.mdx
    ├── data-filtering.mdx
    ├── test-sequences.mdx
    └── full-automations.mdx
```

### Navigation (`_nav.ts`)

```typescript
import type { PageNavNode } from "@/components/PageNav";

export const ARC_NAV: PageNavNode = {
  key: "arc",
  name: "Arc",
  children: [
    {
      key: "/reference/control/arc/get-started",
      href: "/reference/control/arc/get-started",
      name: "Get Started",
    },
    {
      key: "tutorials",
      name: "Tutorials",
      children: [
        {
          key: "/reference/control/arc/tutorials/basic-calculations",
          href: "/reference/control/arc/tutorials/basic-calculations",
          name: "Basic Calculations",
        },
        // ... more tutorials
      ],
    },
    {
      key: "concepts",
      name: "Concepts",
      children: [
        // ... concept pages
      ],
    },
    {
      key: "language-reference",
      name: "Language Reference",
      children: [
        // ... reference pages
      ],
    },
    {
      key: "how-to",
      name: "How-To Guides",
      children: [
        // ... how-to pages
      ],
    },
    {
      key: "examples",
      name: "Examples",
      children: [
        // ... example pages
      ],
    },
  ],
};
```

---

## 8. Files to Update When Adding Arc Docs

### 1. Control Navigation

**File**: `/docs/site/src/pages/reference/control/_nav.ts`

Add Arc to control children:

```typescript
import { ARC_NAV } from "./arc/_nav";

export const CONTROL_NAV: PageNavNode = {
  key: "control",
  name: "Control",
  children: [
    // existing Python nav
    // existing Embedded nav
    ARC_NAV, // Add Arc
  ],
};
```

### 2. Reference Index (Optional)

**File**: `/docs/site/src/pages/reference/index.mdx`

Add Arc to the components table if there's a feature matrix.

### 3. No Root Nav Changes Needed

The root `_nav.ts` already includes Control, so Arc will appear automatically as a
child.

---

## 9. Content Migration Notes

### From Spec to Docs

The existing `/arc/docs/spec.md` (753 lines) is a technical specification. It should be:

- **Adapted** for user-facing documentation (less formal)
- **Split** across multiple pages (concepts, reference, etc.)
- **Enhanced** with examples, tutorials, and how-to guides

### Console Integration

The Console UI documentation for Arc should cross-reference:

- Graph editor usage → link to Arc concepts
- Text editor features → link to language reference
- Deployment process → link to how-to guides

---

## 10. Documentation Conventions

### Existing Patterns to Follow

1. **Start with "Get Started"** - Basic concepts, minimal setup
2. **Use `<Divider.Divider x />`** - Separate major sections
3. **Include `next`/`prev` links** - Navigation flow in frontmatter
4. **Use `Note.Note variant="info"`** - Tips and callouts
5. **Prefer examples over prose** - Show, don't just tell
6. **Consistent heading hierarchy** - h1 for title, h2+ for sections

### Code Examples

Arc code blocks are automatically syntax-highlighted:

````markdown
```arc
func threshold(value f64) (above f64, below f64) {
    if value > 50.0 {
        above = value
    } else {
        below = value
    }
}
```
````

### Cross-References

Use relative markdown links:

```markdown
See [Channels and Series](/reference/control/arc/concepts/channels-and-series) for more
details.
```

---

## 11. Comparison with Existing Control Docs

### Python Sequences Structure

```
get-started.mdx           - First steps
set-and-read-channels.mdx - Channel operations
add-commands-and-control-logic.mdx - Logic building
```

### Embedded (Lua) Sequences Structure

```
get-started.mdx           - First steps
language-basics.mdx       - Syntax and types
```

### Arc Should Mirror and Extend

Arc is more complex than Embedded Lua, so it needs:

- More tutorial depth (like Python)
- Full language reference (beyond Lua basics)
- Concepts section (unique reactive model)
- Examples section (real-world patterns)

---

## 12. Summary

| Aspect         | Details                                              |
| -------------- | ---------------------------------------------------- |
| **Location**   | `/docs/site/src/pages/reference/control/arc/`        |
| **Framework**  | Astro 5.16 + MDX + React 19                          |
| **Syntax**     | Arc grammar already configured in Shiki              |
| **Layout**     | Use `@/layouts/Reference.astro`                      |
| **Components** | Pluto (`Divider`, `Note`), custom (`Image`, `Video`) |
| **Navigation** | Add to `/reference/control/_nav.ts`                  |
| **Deployment** | Auto-deploys to docs.synnaxlabs.com via Vercel       |
