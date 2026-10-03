# 61 Theme-resolved color

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-25
- **Related**:
  [RFC 0044 - Oracle optionality, defaults, and input/output types](0044-oracle-optionality-defaults-mutation.md)

## 0 Summary

A stored color is intent, not paint. The value carries hue, chroma, and alpha. The theme
carries lightness. An absent value means the theme picks the whole color.

This RFC states the rules and the single pipeline that resolves a stored color against a
theme.

## 1 The problem

### 1.0 Three meanings, two encodings

A color field carries one of three meanings:

- **Absent**: The theme picks the color.
- **Chosen**: The user picked this color.
- **Transparent**: The user wants nothing painted.

Today the first and third share an encoding. An unchosen field is stored as
`color.ZERO`, which is transparent black, and every consumer reads that value back as
"unset". `pluto/src/schematic/node/common/primitive/primitive.spec.tsx` pins the
behavior: a `color.ZERO` input clears the `--pluto-symbol-color` custom property so the
stylesheet falls back to the theme.

Deliberate transparency is therefore unrepresentable. RFC 0044 already bans this shape:
a sentinel claims the question was answered when it was not.

### 1.1 Two groups of color fields, and no stated reason

A symbol's `color` is adjusted by the theme before it is painted. Its hue and chroma
survive and its lightness is reset, so the color stays readable on a light or a dark
canvas.

`background_color`, `text_color`, `staleness_color`, and `axis_color` are not adjusted.
They are painted exactly as stored.

Nothing records why the two groups differ.

### 1.2 Theme colors are written into storage

Creating a symbol resolves theme colors and stores the result. A schematic created under
one theme therefore holds colors that belong to that theme, and the stored answer never
changes when the theme does.

## 2 Rules

1. **A color field is optional**: Absence means the theme picks the color.
2. **Every present value is a choice**: Fully transparent included. No value ever stands
   in for absence.
3. **A derived color is never stored**: Creating a symbol leaves an unchosen field
   absent.
4. **The value owns hue, chroma, and alpha; the theme owns lightness**: Readability is a
   property of the surface behind the color, not of what the user meant.
5. **Every color field has a role**: The role decides what absence picks, and how far
   lightness may move.

## 3 Resolution

### 3.0 The pipeline

Resolution runs at render, never at write. It has two steps.

**Seed.** Take the stored color. When the field is absent, take the role's theme color
instead.

**Band.** Clamp the seed's lightness into the role's band. Hue, chroma, and alpha pass
through unchanged. A seed with no chroma carries no hue to preserve, so it snaps to the
theme's extreme: black on a light canvas, white on a dark one.

Absence is not a branch in this pipeline. It only changes where the seed comes from.

### 3.1 Roles

- **Symbol foreground** (`color`, `axis_color`): Absence seeds from the theme's text
  color. The band lifts dark colors on a dark canvas and caps pale colors on a light
  one.
- **Surface fill** (`background_color`): Absence seeds from the theme's surface color.
  The band is inverted against the foreground band, so a light fill stays light on a
  light canvas.
- **Label text** (`text_color`): Absence seeds from the theme's text color. The band
  holds contrast against the fill behind the text, not against the canvas.
- **Status** (`staleness_color`): Absence seeds from the theme's warning color. A user
  value is honored and shares the symbol foreground band.

The symbol foreground band is already implemented in `pluto/src/theming/theme.css`,
parameterized per theme:

|                   | light | dark |
| ----------------- | ----- | ---- |
| lightness floor   | 0     | 0.4  |
| lightness ceiling | 0.8   | 1    |
| achromatic snap   | 0     | 1    |
| chroma threshold  | 0.05  | 0.09 |

## 4 Migration

Every stored `color.ZERO` means "absent" today, because the encoding made deliberate
transparency unreachable. Rewriting `color.ZERO` to absent therefore cannot misread a
user's intent: no user was able to express the meaning the new encoding gives it.
