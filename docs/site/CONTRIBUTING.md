# Contributing to the Synnax documentation site

## Synnax product conventions

### Referring to products

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

### Console components

Console components are written in sentence case. Capitalize the first word of the
component name, and keep the component type lowercase:

- Channels toolbar
- Core selector
- Devices toolbar
- Ranges toolbar
- Search and command palette
- Tasks toolbar
- Users toolbar
- Project selector
- Projects toolbar

Make sure to use the correct form (plural or singular) when referring to these ("Ranges
toolbar" not "Range toolbar", and "Visualization toolbar" not "Visualizations toolbar").
After referring to these once, you should use the lower case of the component type when
referring to it ("toolbar", "selector", "palette").

### Task names

Task names are written in sentence case. Only proper nouns and acronyms keep their
capitals ("NI digital read task" not "NI Digital Read Task"). When referring to the
general category of a task, use lowercase ("write tasks let the Driver control
hardware").

## Documentation site conventions

### Sentence case

All headings, page titles, and navigation labels are in sentence case. Capitalize only
the first word, proper nouns, and acronyms ("Task configuration reference", not "Task
Configuration Reference").

### Dividers

Before every `<h2>` tag, you should add a `<Divider.Divider x />` tag.

```mdx
<Divider.Divider x />

## Ranges
```

## Writing conventions

### Linking to pages

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

### External product reference

When referring to external products, please use proper capitalization:

- macOS, not Mac OS
- PowerShell, not Power Shell
- NI-DAQmx, not DAQ Mx
- OPC UA driver, not OPC driver

Using shorthand ("DAQmx") is acceptable after the first reference.
