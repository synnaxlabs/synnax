---
name: brand
description: >-
  The Synnax visual language for anything that shows Synnax outside the product UI: the
  website, docs hero art, release note stills, films, social posts, slides, and custom
  icons. Use when designing, generating, or reviewing such a visual. Gives the token and
  logo sources, type, color, imagery, icon, and copy rules.
---

# Synnax brand

Synnax is control and observability software for teams that run expensive hardware:
rocket engines, fusion machines, test stands. The look is precise and engineered, like
the instruments those teams use. Real work on screen is the decoration.

## Sources

Read values from these files. Never copy a hex value or a size into a design from
memory.

- **Tokens** -> `lyra/src/theme/theme.ts` (`SYNNAX_LIGHT`, `SYNNAX_DARK`). The CSS
  variables (`--pluto-*`) are in `lyra/src/theming/static/theme-{light,dark}.css`.
- **Logo** -> `lyra/static/logo/`: the icon, the title (icon and name), and the name
  alone, each in gradient, white, and black, as SVG and PNG.
- **Marketing type scale** -> `site/www/src/styles/base.css`.
- **Icons** -> `lyra/src/icon/`.
- **Stills and films** -> `studio/scripts/still-*.ts` and `studio/films/`. They drive a
  real Console with demo data.
- **Docs page layout** -> `.claude/skills/docs-refine/design-system.md`.

## Type

- Inter Variable for text. Geist Mono for code, values, and small uppercase labels.
- Headings use weight 500 to 550. Display sizes on the website use tight tracking
  (-0.02em to -0.035em). Large description text uses weight 350.
- Sentence case for every heading, title, button, and label.

## Color

- **Grays** -> the 12-step `gray` scale, `l0` (canvas) to `l11` (emphatic text). It has
  a cool hue and almost no chroma. Use these steps for every neutral.
- **Primary blue** -> the one accent: links, the state that matters, the brand mark.
- **Logo gradient** -> deep indigo to blue, from `Logo.tsx`. The logo, end cards, and
  the brand moment of a piece. Not a fill for general surfaces.
- **Green, red, amber** -> status (success, error, warning). In art, use them only when
  the subject is that state, such as an alarms page.
- **Plot lines** -> `colors.visualization.palettes.default`.

The website is dark. A product still shows one theme, never a light and dark split.

## Imagery

- Show dense, real work: a full schematic with live values, plots of a real hot fire,
  many channels. Use the demo world in `studio/films/coldflow.ts`. Never film or shoot a
  bare canvas or a toy system with one tank and one valve.
- Show the whole Console window, with its nav bar, tabs, and toolbar. Do not crop
  tightly onto one part, and never magnify UI text.
- Capture at device scale 3 and keep that resolution. Never downscale a still.
- Gradients behind a still are soft or deep, never bright or saturated.
- Rejected styles: tinted dot grids, collages, mesh glows, spotlight fades, 3D tilts,
  and magnifier callouts.
- A drafting sheet works: a faint grid, construction lines, corner ticks, and one full
  window.

## Icons

- Precise and engineered, like IBM Carbon, SF Symbols, and the even-weight band of the
  logo. Never cartoony.
- A 24-unit grid, strokes near 1.5 units, square line ends, corner radii of 1 unit or
  less, square details instead of round dots.
- Prefer filled glyphs: solid shapes cut by clean gaps, like the Range, Arc, and Task
  icons. No literal objects or puns, such as a hammer for hardware.
- Check a new icon at 13 to 14 px, next to the icons it sits with.

## Copy on a visual

- ASD-STE100. Short sentences, no em dashes.
- Lead with a concrete fact, number, or contrast. Name the mechanism or the number
  ("10M+ points per second"), never an adjective ("blazing fast").
- Sell the outcome: the pain, then the before and after.
- Never name a customer or show their data.

## Before you deliver

Look at the piece at its real size, in its real place (the page, the feed, the slide),
in the theme it ships in.
