# 53 - Pluto Border Radius Scale

**Feature Name**: Pluto Border Radius Scale <br /> **Status**: Approved <br /> **Start
Date**: 2026-07-31 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC replaces every ad-hoc corner radius in Pluto and Console with a five-step named
ladder generated from the theme spec: `tiny` (2/3rem), `small` (1rem), `medium`
(1.5rem), `large` (2rem), and `huge` (3rem), rendering 4/6/9/12/18px at the default 6px
base size. Steps are rem-based so corners scale with density, emitted as
`round(Nrem, 1px)` so they stay pixel-crisp. The bare `--pluto-border-radius` token
survives as the theme default and points at the tiny step. `Flex.Box`'s `rounded` prop
accepts `boolean | number | Component.Size`, with the size-name form as the house style.

# 1 - Motivation

An audit found roughly 13 distinct corner sizes across Pluto and Console, reached
through two parallel unit systems: the px-emitted `--pluto-border-radius` token (frozen
at 4px) and rem literals plus numeric `rounded` props (scaling with the base size). The
two coincide only at the default base. Evidence that the ladder already wanted to exist:
a hand-computed `0.66666rem` faking a scaling 4px (`color/Swatch.css`), a hardcoded
`--pluto-pack-br: 6px` mirroring a `1rem` dialog radius, 4px spelled four different
ways, and a drift trend in which filled buttons, tabs, and tags had all independently
migrated to 6px while outlined buttons stayed at 4px. Same-role elements disagreed: the
chip role spanned 3/4/6/9px, with 6px and 9px in the same tab strip.

Prior art: Tailwind (2/4/6/8/12/16/24 + full), Material 3 (4/8/12/16/28 + full), Radix
Themes (about 3/4/6/8/12/16 from one scale factor), Fluent 2 (2/4/6/8 + circular), and
GitHub Primer (3/6/12 + full) all use a small named ladder with roughly 1.5x steps. We
align with Primer's shape, extended by the steps our audit showed in real use.

# 2 - Principles

1. **One ladder.** Every chrome corner speaks a named step or derives from one via
   `calc`. Hand-picked radii are reserved for proportional details (below).
2. **Corners scale with density.** Heights, padding, and type are rem; corners are too.
   `round(Nrem, 1px)` keeps them crisp at any base size.
3. **Roles are stated once, here,** not encoded in token names. Size names match
   `Component.Size`, the same vocabulary as the height ladder.
4. **Derived values stay derived.** A nested surface tracking an outer radius uses
   `calc(token - inset)`; concentric ornaments use `calc(token + inset)`.
5. **Proportional details stay local.** Radii that track a component's own geometry
   (checkbox indicators, stadium pills on thin bars, 50% circles, the schematic vessel
   percent system) are not ladder steps.

# 3 - Design

## 3.1 - Tokens

`theme.sizes.border.radius` becomes a five-field object of rem multiples
(`pluto/src/theming/base/theme.ts`), emitted by `toCSSVars` (`pluto/src/theming/css.ts`)
as:

| Token                          | Value                  | Default render |
| ------------------------------ | ---------------------- | -------------- |
| `--pluto-border-radius-tiny`   | `round(0.6667rem,1px)` | 4px            |
| `--pluto-border-radius-small`  | `round(1rem,1px)`      | 6px            |
| `--pluto-border-radius-medium` | `round(1.5rem,1px)`    | 9px            |
| `--pluto-border-radius-large`  | `round(2rem,1px)`      | 12px           |
| `--pluto-border-radius-huge`   | `round(3rem,1px)`      | 18px           |
| `--pluto-border-radius`        | `var(...-tiny)`        | 4px            |

The bare token remains the theme default, so `pluto--rounded`, the roughly 50 existing
`var()` sites, and the canvas renderer keep reading "the default" without naming a step.
The canvas mirror (`vis/draw2d`) computes `Math.round(radius.tiny * base)` px.

## 3.2 - API

`rounded?: boolean | number | Component.Size` on `Flex.Box`. `true` applies the default
class, a size name applies `pluto--rounded-<size>` (class-based, so pack corner
inheritance and seam-squaring rules can still override it; the inline-style number form
defeats those and once forced an `!important` in the task controls bar), and a number
remains an inline `${n}rem` escape hatch for derived one-offs.

## 3.3 - Role assignment

| Step   | Roles                                                                    |
| ------ | ------------------------------------------------------------------------ |
| tiny   | controls: outlined/text buttons, inputs, scrollbars, keycaps, drop zones |
| small  | chips + surfaces: filled buttons, tabs, tags, dialogs, tooltips, rows    |
| medium | cards: cluster/project rows, island chips, large create row, switch      |
| large  | top-level panels: modals, context menus, mosaic, drawers, task panels    |
| huge   | the pre-workspace shell card; ornament rings derive from it via `calc`   |

Filled buttons deliberately sit one step above other button variants; the fill carries
enough visual mass that the rounder corner reads as character rather than inconsistency.
This is the only shape distinction between variants of one control.

# 4 - Implementation

Landed in one pass with this RFC: token generation and regenerated static CSS, the
`rounded` size form, migration of every CSS literal and numeric `rounded` prop to steps,
and the kill list: `--pluto-pack-br` (dead token), `CSS.rounded`/`CSS.sharp` helpers
(dead, with inverted defaults relative to `Flex.Box`), the `0.66666rem` hack, the
legend's duplicate radius declaration, the task form's dead `rounded` prop, and the
label-select create row that hand-copied `Button.Create`'s geometry.

Deliberate pixel changes shipped with the migration: tooltip and Monaco hover 8px to
6px, mosaic create button 9px to 6px, arc type chip and gradient-stop marker 3px to 4px,
nav connection badge 4px to 6px.

# 5 - Resolved Decisions

**Rem over px.** A px ladder preserved 4px on a clean grid but froze corners while every
other dimension scales; the codebase already contained a hand-rolled scaling 4px. The
trade is one off-grid fraction (2/3rem), confined to the theme spec.

**4px kept.** A pure 0.5rem-grid ladder (3/6/9/12) was rejected: the 4px control radius
is the established default and the user chose to keep it, accepting the fractional rem
step.

**Size names over role names.** Role-named tokens (control/chip/card/panel) would
self-document but would be the only role-named size tokens in the system and would fight
the `Component.Size` vocabulary. The role map lives in this RFC instead.

**Default points at tiny.** The lexical oddity ("the default radius is tiny") was
accepted over shifting names up and coining a sixth name for 3rem.

**Number form kept.** Dropping `rounded`'s number form entirely was considered; kept as
an escape hatch for derived values, with size names as the house style.

# 6 - Open Questions

None. A future density/zoom setting would exercise the rem scaling; nothing here blocks
or requires it.
