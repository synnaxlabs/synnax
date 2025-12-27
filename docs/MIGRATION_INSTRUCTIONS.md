# Client Documentation Migration Instructions

This document provides detailed step-by-step instructions for migrating content from
the separate `python-client` and `typescript-client` documentation pages to the new
unified `client` documentation structure.

## Overview

The goal is to consolidate Python and TypeScript client documentation into unified
pages that use `<Client.Tabs>` components to switch between languages. During migration,
we use visual markers (colored Note blocks) to distinguish content sources while
preserving the tab structure.

## Prerequisites

Before starting any migration task:

1. **Read `CONTENT_MAPPING.md`** - Understand which sections need to be migrated and
   where they should go
2. **Identify the target page** - The new unified page in `/reference/client/`
3. **Identify source pages** - The corresponding pages in `/reference/python-client/`
   and `/reference/typescript-client/`

## Migration Process (Step-by-Step)

### Step 1: Read All Source Files

Read the following files to understand the existing content:

```
/docs/site/src/pages/reference/client/{section}/{target-page}.mdx    (destination)
/docs/site/src/pages/reference/python-client/{source-page}.mdx       (Python source)
/docs/site/src/pages/reference/typescript-client/{source-page}.mdx   (TypeScript source)
```

Also check if related content exists in other source files (e.g., range-related content
might be in `read-data.mdx` and `write-data.mdx`, not just `ranges.mdx`).

### Step 2: Content Transfer Pattern

For each section in the target page, replace placeholder content with actual content
from source files using this exact pattern:

#### 2.1 Tab Structure

Keep the existing `<Client.Tabs>` structure intact. The exclude array should typically
be `exclude={["cpp","console"]}` unless the page has Console content ready.

```jsx
<Client.Tabs client:load exclude={["cpp","console"]}>
  <Fragment slot="console">
  </Fragment>

  <Fragment slot="python">
    <Note.Note variant="warning">

{/* Python content goes here - MUST have blank line after opening Note */}

    </Note.Note>
  </Fragment>

  <Fragment slot="typescript">
    <Note.Note variant="info">

{/* TypeScript content goes here - MUST have blank line after opening Note */}

    </Note.Note>
  </Fragment>
</Client.Tabs>
```

#### 2.2 Color Coding with Note Variants

**CRITICAL: Use these exact variants for visual distinction:**

| Language   | Note Variant  | Rendered Color |
| ---------- | ------------- | -------------- |
| Python     | `"warning"`   | Yellow/Orange  |
| TypeScript | `"info"`      | Blue           |

**Example:**

```jsx
<Fragment slot="python">
  <Note.Note variant="warning">

Python content here...

  </Note.Note>
</Fragment>

<Fragment slot="typescript">
  <Note.Note variant="info">

TypeScript content here...

  </Note.Note>
</Fragment>
```

#### 2.3 New Section Titles (No Direct Match)

If a section doesn't have a corresponding section in BOTH source files, mark the title
with RED styling:

```jsx
## <span style={{ color: "red" }}>New Section Title</span>
```

This indicates the section is new to the unified structure or only exists in one
language.

### Step 3: Content Formatting Rules

#### 3.1 Whitespace Requirements

**CRITICAL**: MDX requires specific whitespace around content inside JSX components:

```jsx
<Note.Note variant="warning">

{/* BLANK LINE REQUIRED HERE */}
Your markdown content starts here...

Tables, code blocks, etc.

{/* BLANK LINE REQUIRED HERE */}
    </Note.Note>
```

Without blank lines, markdown won't render properly inside the Note components.

#### 3.2 Code Block Handling

Code blocks can be included directly inside Note components:

```jsx
<Note.Note variant="warning">

To create a range, use the `client.ranges.create` method:

```python
import synnax as sy

start = sy.TimeStamp("2023-02-12 12:30:00")
my_range = client.ranges.create(
    name="My Range",
    time_range=sy.TimeRange(start=start, end=end),
)
```

    </Note.Note>
```

#### 3.3 Tables

Tables work inside Note components but need blank lines:

```jsx
<Note.Note variant="warning">

| Parameter | Type   | Required | Description |
| --------- | ------ | -------- | ----------- |
| `name`    | string | Yes      | Range name  |

    </Note.Note>
```

#### 3.4 Missing Features

When a feature doesn't exist in one language, add a brief note explaining this:

```jsx
<Fragment slot="typescript">
  <Note.Note variant="info">

TypeScript doesn't support conditional range creation.

  </Note.Note>
</Fragment>
```

### Step 4: Handle Nested Subsections

If the source has multiple subsections under a main section, include ALL subsections
inside the same `<Client.Tabs>` block:

```jsx
## Main Section

<Client.Tabs client:load exclude={["cpp","console"]}>
  <Fragment slot="python">
    <Note.Note variant="warning">

### Subsection A

Python content for A...

### Subsection B

Python content for B...

### Subsection C

Python content for C...

    </Note.Note>
  </Fragment>

  <Fragment slot="typescript">
    <Note.Note variant="info">

{/* TypeScript might have different subsection organization */}

Content here...

    </Note.Note>
  </Fragment>
</Client.Tabs>
```

### Step 5: Replace Original Source Files with Redirects

After transferring content to the target page, replace the original source files with
redirect notices:

```mdx
---
layout: "@/layouts/Reference.astro"
title: "Original Title"
description: "Original description"
next: "Next Page"
nextURL: "/reference/python-client/next"
prev: "Previous Page"
prevURL: "/reference/python-client/prev"
---

import { Divider, Note } from "@synnaxlabs/pluto";
import { mdxOverrides } from "@/components/mdxOverrides";
export const components = mdxOverrides;

<Note.Note variant="info">
  This content has been migrated to the unified client documentation. See{" "}
  <a href="/reference/client/{section}/{page}">Page Title</a> for the updated
  documentation.
</Note.Note>
```

**Important:**

- Keep the frontmatter (title, description, navigation) unchanged
- Remove all actual content
- Remove unused imports (keep only `Divider, Note` from pluto)
- Remove `ExampleDetails`, `Icon`, `Text` imports if present

### Step 6: Remove Migrated Content from Related Files

If content was scattered across multiple source files (e.g., "Reading from Ranges" in
`read-data.mdx` and "Writing to Ranges" in `write-data.mdx`), remove those sections
from those files.

**Only remove the specific sections that were migrated, not the entire file.**

### Step 7: Update CONTENT_MAPPING.md

After migration, update the tracking document:

1. **Navigation Structure**: Change the page status from `📝 X sections` to
   `🔄 In Progress`

2. **Sections Remaining to Transfer**: Remove migrated sections from the tree listings
   under both Python and TypeScript

3. **Summary table**: Update the "Remaining" counts

4. **Implementation Checklist**: Mark the item with `[🔄]` and add a note like
   "(content transferred, needs cleanup)"

5. **Total Section Count table**: Update status to reflect progress

## Complete Example: Ranges Migration

Here's what a fully migrated section looks like in the target file:

```jsx
<Divider.Divider x />

## Creating Ranges

<Client.Tabs client:load exclude={["cpp","console"]}>
  <Fragment slot="console">
  </Fragment>

  <Fragment slot="python">
    <Note.Note variant="warning">

To create a range, we can use the `client.ranges.create` method:

```python
import synnax as sy

start = sy.TimeStamp("2023-02-12 12:30:00")
end = sy.TimeStamp("2023-02-12 14:30:00")

my_range = client.ranges.create(
    name="My Range",
    time_range=sy.TimeRange(start=start, end=end),
)
```

Synnax will automatically generate a unique identifier for the range.

    </Note.Note>
  </Fragment>

  <Fragment slot="typescript">
    <Note.Note variant="info">

To create a range, we can use the `client.ranges.create` method:

```typescript
import { TimeRange, TimeStamp } from "@synnaxlabs/client";

const start = new TimeStamp("2023-02-12 12:30:00");
const end = new TimeStamp("2023-02-12 14:30:00");
const range = await client.ranges.create({
  name: "My Range",
  timeRange: new TimeRange(start, end),
});
```

This creates a range inside Synnax that you can then view from the Console.

    </Note.Note>
  </Fragment>
</Client.Tabs>
```

## Audit Checklist

After completing migration for a page, verify:

- [ ] All sections from Python source are accounted for
- [ ] All sections from TypeScript source are accounted for
- [ ] Related content from other files (read-data, write-data) is migrated
- [ ] Original source files replaced with redirects
- [ ] Related files have migrated sections removed
- [ ] CONTENT_MAPPING.md is updated
- [ ] No content was lost (compare with `git show HEAD:path/to/file` if needed)

## Common Mistakes to Avoid

1. **Missing blank lines**: Content won't render without blank lines after `<Note.Note>`
   opening tag

2. **Wrong Note variant**: Python MUST use `variant="warning"`, TypeScript MUST use
   `variant="info"`

3. **Removing Client.Tabs**: Keep the tab structure - only replace content inside

4. **Forgetting redirects**: Always replace original files with redirect notices

5. **Missing content from related files**: Check read-data.mdx, write-data.mdx for
   content that belongs in the migrated page

6. **Using inline backgroundColor styles**: Use Note variants, not inline styles

7. **Copy instead of cut**: The goal is to MOVE content, not duplicate it. Source files
   should become redirects.

## Future Cleanup Phase

After all content is migrated, a cleanup phase will:

1. Remove the `<Note.Note>` wrapper blocks
2. Integrate Python/TypeScript content properly into the tab structure
3. Add Console tab content where applicable
4. Final navigation and link review
5. Delete or redirect remaining old pages

## Files Reference

| File Type          | Path Pattern                                               |
| ------------------ | ---------------------------------------------------------- |
| Target (unified)   | `/docs/site/src/pages/reference/client/{section}/{page}.mdx` |
| Python source      | `/docs/site/src/pages/reference/python-client/{page}.mdx`  |
| TypeScript source  | `/docs/site/src/pages/reference/typescript-client/{page}.mdx` |
| Progress tracking  | `/docs/CONTENT_MAPPING.md`                                  |
| These instructions | `/docs/MIGRATION_INSTRUCTIONS.md`                           |
