# Contributing to the Synnax Documentation Site

## Synnax Product Conventions

### Referring to Products

Synnax has several products, and each should be referred to by its proper name,
including capitalization. The following are the proper names:

- Synnax Core ("Core" for short)
- Synnax Console ("Console" for short)
- Synnax Driver ("Driver" for short)
- Synnax Python client ("Python client" for short)
- Synnax TypeScript client ("TypeScript client" for short)

When mentioning a product, use the full name the first time referencing it, and then use
the short name thereafter. As an analogy, imagine writing an essay on a topic, and
referring to "George Washington" the first time, and then "Washington" thereafter.

### Console Components

The following components of the Console should be referenced with capital letters:

- Channels Toolbar
- Cores Dropdown
- Devices Toolbar
- Ranges Toolbar
- Search and Command Palette
- Tasks Toolbar
- Users Toolbar
- Workspaces Dropdown
- Workspaces Toolbar

Make sure to use the plural when referring to these ("Ranges Toolbar" not "Range
Toolbar"). After referring to these once, you should use the lower case of the component
type when referring to it ("toolbar", "dropdown", "palette").

### Task Names

When referring to the full name of a task, capitalize everything ("NI Digital Read Task"
not "NI digital read task"). When referring to the general category of a task, use
lowercase, ("write tasks let the Driver control hardware").

## Documentation Site Conventions

### Title Case

In general, `<h1>`, `<h2>`, and `<h3>` tags should be in title case. `<h4>` tags and
lower level tags should be in sentence case.

### Dividers

Before every `<h2>` tag, you should add a `<Divider.Divider x />` tag.

```mdx
<Divider.Divider x />

## Ranges
```

## Writing Conventions

### Linking to Pages

When writing a link to a page, make sure the text the link is on has semantic meaning.
DO NOT write links on the word "page", "pages", or "here".

DO this:

```mdx
In Synnax, [ranges](/reference/concepts/ranges) are the primary means for organizing and
accessing the data stored in a Synnax Core.
```

DO NOT do this:

```mdx
In Synnax, ranges are the primary means for organizing and accessing the data stored in
a Synnax Core. You can find more information on ranges
[here](/reference/concepts/ranges).
```

or this:

```mdx
In Synnax, ranges are the primary means for organizing and accessing the data stored in
a Synnax Core. You can find more information on ranges on this
[page](/reference/concepts/ranges).
```

This helps with readability and SEO.

### External Product Reference

When referring to external products, please use proper capitalization:

- macOS, not Mac OS
- PowerShell, not Power Shell
- NI-DAQmx, not DAQ Mx
- OPC UA driver, not OPC driver

Using shorthand ("DAQmx") is acceptable after the first reference.
