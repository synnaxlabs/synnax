# Adding Arc to Synnax Docs - Technical Reference

---

## 1. Documentation Location and Framework

**Path**: `/docs/site/`

**Framework**:

- Astro 5.16.5 with MDX support
- React 19 for interactive components
- Shiki syntax highlighting with Arc grammar
- Vercel deployment (automatic on merge)

**Build Commands**:

```bash
cd docs/site
pnpm dev         # Development server
pnpm build       # Production build
pnpm check-types # Type checking
```

**Live Site**: docs.synnaxlabs.com

---

## 2. Directory Structure

```
/docs/site/src/
├── pages/
│   ├── reference/              # Main documentation
│   │   ├── _nav.ts            # Navigation config
│   │   ├── concepts/          # Core concepts
│   │   ├── core/              # Server docs
│   │   ├── client/            # Client libraries
│   │   ├── control/           # Control sequences ← ARC GOES HERE
│   │   ├── console/           # Console app
│   │   ├── driver/            # Hardware driver
│   │   └── pluto/             # Visualization
│   ├── guides/                # User guides
│   └── releases/              # Release notes
├── layouts/
│   └── Reference.astro        # Reference page layout
├── components/
│   ├── Media/                 # Image, Video components
│   ├── PageNav/               # Navigation components
│   ├── Table.astro            # Table component
│   └── mdxOverrides.ts        # MDX component overrides
└── util/
```

---

## 3. Existing Reference Sections

| Section  | Path                   |
| -------- | ---------------------- |
| Concepts | `/reference/concepts/` |
| Core     | `/reference/core/`     |
| Client   | `/reference/client/`   |
| Control  | `/reference/control/`  |
| Console  | `/reference/console/`  |
| Driver   | `/reference/driver/`   |
| Pluto    | `/reference/pluto/`    |

---

## 4. Control Section Structure

Arc lives in `/reference/control/` alongside Python and Embedded sequences:

```
/reference/control/
├── _nav.ts                # Main control navigation
├── get-started.mdx
├── control-authority.mdx
├── python/                # Python sequences
│   ├── _nav.ts
│   └── *.mdx
├── embedded/              # Lua sequences
│   ├── _nav.ts
│   └── *.mdx
└── arc/                   # Arc documentation ← NEW
    ├── _nav.ts
    └── *.mdx
```

---

## 5. Navigation Configuration

### Navigation File Format

```typescript
// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav/PageNav";

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
      key: "/reference/control/arc/concepts",
      href: "/reference/control/arc/concepts",
      name: "Concepts",
    },
  ],
};
```

### Nested Navigation

```typescript
{
  key: "concepts",
  name: "Concepts",
  children: [
    {
      key: "/reference/control/arc/concepts/reactive-execution",
      href: "/reference/control/arc/concepts/reactive-execution",
      name: "Reactive Execution",
    },
  ],
}
```

### Importing Child Navigation

```typescript
import { ARC_NAV } from "@/pages/reference/control/arc/_nav";

export const CONTROL_NAV: PageNavNode = {
  key: "control",
  name: "Control",
  children: [
    // ... existing children
    ARC_NAV,
  ],
};
```

---

## 6. MDX Format Reference

### Frontmatter

```yaml
---
layout: "@/layouts/Reference.astro"
title: "Page Title"
description: "Brief description for SEO"
next: "Next Page Title"
nextURL: "/reference/control/arc/next-page"
prev: "Previous Page Title"
prevURL: "/reference/control/arc/prev-page"
---
```

| Field         | Required | Purpose                                       |
| ------------- | -------- | --------------------------------------------- |
| `layout`      | Yes      | Always `@/layouts/Reference.astro`            |
| `title`       | Yes      | Browser tab and nav display                   |
| `description` | Yes      | SEO meta description                          |
| `heading`     | No       | Larger page heading (if different from title) |
| `next`        | No       | Next page title for navigation                |
| `nextURL`     | No       | Next page URL                                 |
| `prev`        | No       | Previous page title for navigation            |
| `prevURL`     | No       | Previous page URL                             |

### Standard Imports

```typescript
import { Divider, Note } from "@synnaxlabs/pluto";
import { Image, Video } from "@/components/Media";
import { mdxOverrides } from "@/components/mdxOverrides";

export const components = mdxOverrides;
```

The `mdxOverrides` export is required - it provides styled headings with anchor links,
code blocks with copy buttons, and styled tables.

### Heading Hierarchy

- **Never use H1** - Generated from frontmatter `title`
- **H2 (`##`)** - Major sections
- **H3 (`###`)** - Subsections
- **H4 (`####`)** - Sub-subsections (use sparingly)

All headings get anchor links automatically (e.g., `#section-name`).

### Dividers

```jsx
<Divider.Divider x />
```

Use between major sections.

### Notes (Callouts)

```jsx
<Note.Note variant="info">
  Helpful information or tips.
</Note.Note>

<Note.Note variant="warning">
  Important caution or gotcha.
</Note.Note>

<Note.Note variant="error">
  Critical warning or breaking change.
</Note.Note>
```

### Code Blocks

````markdown
```arc
func add(x f64, y f64) f64 {
    return x + y
}
```
````

Supported: `arc`, `python`, `typescript`, `go`, `lua`, `bash`, `json`, `yaml`

### Images

```jsx
<Image client:only="react" id="control/arc/image-name" />
```

CDN path format: `control/arc/<page-name>/<image-description>`

### Videos

```jsx
<Video client:only="react" id="control/arc/video-name" />
```

CDN path format: `control/arc/<page-name>/<action-description>`

Videos auto-play, loop, and are muted. Both images and videos support light/dark
variants automatically.

### Media Placeholders

When writing docs, add placeholder stubs for media that needs to be recorded:

```jsx
{
  /* TODO: Video - control/arc/get-started/create-new-automation */
}
<Video client:only="react" id="control/arc/get-started/create-new-automation" />;

{
  /* TODO: Image - control/arc/concepts/dataflow-diagram */
}
<Image client:only="react" id="control/arc/concepts/dataflow-diagram" />;
```

### Tables

Standard markdown:

```markdown
| Column 1 | Column 2 |
| -------- | -------- |
| Value 1  | Value 2  |
```

### Collapsible Sections

```jsx
<details>
  <summary>Click to expand</summary>
  Hidden content here.
</details>
```

### Links

Internal:

```markdown
See [Reactive Execution](/reference/control/arc/concepts/reactive-execution).
```

External:

```markdown
See the [Synnax website](https://synnaxlabs.com).
```

### Complete Page Example

````mdx
---
layout: "@/layouts/Reference.astro"
title: "Get Started"
description: "Create your first Arc automation"
next: "Reactive Execution"
nextURL: "/reference/control/arc/concepts/reactive-execution"
---

import { Divider, Note } from "@synnaxlabs/pluto";
import { Video } from "@/components/Media";
import { mdxOverrides } from "@/components/mdxOverrides";

export const components = mdxOverrides;

Arc is a domain-specific language for reactive automation and control systems.

<Note.Note variant="info">
  Arc is currently in beta. Language features may change before the stable release.
</Note.Note>

<Divider.Divider x />

## Creating Your First Automation

Open Console and navigate to the Arc section.

{/* TODO: Video - control/arc/get-started/create-automation */}

<Video client:only="react" id="control/arc/get-started/create-automation" />

## Writing Arc Code

```arc
func double(value f64) f64 {
    return value * 2
}

sensor -> double{} -> output
```
````

<Divider.Divider x />

## Next Steps

See [Reactive Execution](/reference/control/arc/concepts/reactive-execution) to
understand how Arc programs execute.

````

---

## 7. Arc Syntax Highlighting

Arc syntax highlighting is configured in `/docs/site/astro.config.ts`:

```typescript
import { grammar as arcGrammar } from "@synnaxlabs/arc";

export default defineConfig({
  markdown: {
    shikiConfig: {
      theme: "css-variables",
      langs: [arcGrammar],
    },
  },
});
````

### Keeping Syntax Highlighting in Sync

The Arc grammar is defined in `/arc/ts/src/arc.tmLanguage.json`. This grammar is used by
both:

- **Docs site** - Shiki highlighting (via astro.config.ts)
- **Console** - Monaco editor highlighting

When Arc syntax changes, update the grammar file. Both environments pick up changes
after rebuild. Verify highlighting matches between docs and Console.

---

## 8. Implementation Checklist

### Create Arc Directory

- [ ] Create `/docs/site/src/pages/reference/control/arc/`
- [ ] Create `_nav.ts` with navigation structure
- [ ] Create `index.astro` (redirect to first page)

### Create MDX Pages

- [ ] Create pages per structure in `02-documentation-best-practices.md`
- [ ] Add frontmatter (layout, title, description)
- [ ] Add next/prev navigation links
- [ ] Include standard imports (`mdxOverrides`, `Divider`, `Note`)
- [ ] Add media placeholders with CDN paths for videos/images

### Update Parent Navigation

- [ ] Edit `/docs/site/src/pages/reference/control/_nav.ts`
- [ ] Import `ARC_NAV` from `./arc/_nav`
- [ ] Add `ARC_NAV` to children array

### Verify Build

- [ ] Run `pnpm build` - no errors
- [ ] Run `pnpm dev` - navigate to Arc pages
- [ ] Verify Arc syntax highlighting works
- [ ] Verify navigation links work
- [ ] Verify next/prev navigation works

### Final Checks

- [ ] All pages have descriptions
- [ ] Code examples are accurate
- [ ] No broken internal links
- [ ] Media placeholders have correct CDN paths
