# 50 - Pluto Grayscale Rearchitecture

**Feature Name**: Pluto Grayscale Rearchitecture <br /> **Status**: Implemented <br />
**Start Date**: 2026-07-29 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC re-architects the Pluto 12-slot gray ramp (`--pluto-gray-l0` through `l11`) for
both themes. It assigns each slot exactly one role in four non-overlapping bands, moves
the ramp from pure gray to a whisper tint derived from the Synnax primary hue, rebuilds
the values in OKLCH with even perceptual steps, deletes the `contrast` prop, and unifies
the press and keyboard-focus vocabularies. Exact channel values are tunable parameters;
the band roles and state rules are the contract.

# 1 - Motivation

A census of the pre-redesign ramp found structural problems, not value problems:

1. **Slots have no single role.** The light ramp's own comments label l4 through l7 as
   borders, but l4 is a de facto pressed fill across the button variants, l6 carries
   five conflicting roles, and l7 has been colonized as faint text. Tuning any slot for
   one consumer breaks another.
2. **Surfaces and component fills share slots.** Elevated chrome and component rest fill
   both landed on l2, producing the invisible-control class of bugs (a silence button on
   an l2 surface with an l2 fill).
3. **Contrast is broken at both ends.** In the dark theme the l1 and l2 surface steps
   are nearly invisible (2 to 4 OKLab lightness points at a floor of L 0.085), while the
   text band jumps 22 points at once. Secondary text at l8 fails WCAG AA on both themes
   (2.9:1 light, 4.0:1 dark); placeholders are not exempt under 1.4.3.
4. **Seven press vocabularies and four keyboard-focus geometries** coexist, two
   components have no press feedback, and the focus halos paint their gap color with a
   guessed surface level (Switch guesses l1 where Checkbox guesses l0).
5. **The `contrast` prop is half machinery.** Only values 1 through 3 have CSS behind
   them, `filled` ignores it, and the only principled surface link in the codebase is
   the `Menu.background` context.
6. **Known bugs.** `theming/css.ts` generates the l9 alpha variants from l11, and
   `telem/control/Chip.tsx` references a nonexistent `--pluto-gray-l12`.

# 2 - Vocabulary

- **Slot**: one of the 12 ramp positions l0 through l11.
- **Band**: a contiguous group of slots sharing a role family.
- **Variant-var protocol**: the Button convention where variants set `--pluto-bg`,
  `--pluto-hover-bg`, `--pluto-active-bg` (and border equivalents) and shared rules swap
  them on hover and press.
- **Surface**: a background something sits on (canvas, pane, dialog, chrome).
- **Fill**: the background of an interactive component itself.

# 3 - Principles

1. **One slot, one role.** Every slot belongs to exactly one band and is named by that
   role. A consumer that wants a different look moves to the slot whose role matches,
   never bends a slot's value.
2. **Bands do not overlap.** Surfaces stop at l2; fills start at l3. A component at rest
   is always at least one full step from any legal surface, so per-surface indexing is
   unnecessary (the Radix and Linear model of absolute steps).
3. **Dark cockpit.** The dark canvas stays dark. Contrast comes from step spacing above
   the floor, not from lifting the floor to the industry cluster (Linear sits at OKLab L
   0.139, Radix slate at 0.179; we float l0 at ~0.115).
4. **Quiet but present states.** Hover, press, and selection are each one deliberate
   step, expressed only through tokens. No transforms, no opacity tricks, no bespoke
   hexes.
5. **Whisper temperature.** The ramp carries a trace of the primary hue (OKLCH hue 258),
   strongest in the mid-tones, near zero at the extremes. Grays read warm-of-life
   without ever reading as blue.
6. **AA is a floor, not a goal.** Every text slot clears 4.5:1 on every surface it may
   legally sit on, placeholders included. Disabled text is exempt and becomes an alpha
   token rather than a ramp slot.

# 4 - Design

## 4.0 - The band table

| Slot | Band    | Role                                           |
| ---- | ------- | ---------------------------------------------- |
| l0   | Surface | App canvas (mosaic background)                 |
| l1   | Surface | Raised surface (panes, cards)                  |
| l2   | Surface | Elevated chrome (dialogs, menus, toolbars)     |
| l3   | Fill    | Component rest fill                            |
| l4   | Fill    | Hover fill                                     |
| l5   | Fill    | Pressed fill                                   |
| l6   | Border  | Subtle separator (pane seams, dividers)        |
| l7   | Border  | Default control border                         |
| l8   | Border  | Strong border (hover, emphasis)                |
| l9   | Text    | Secondary text, placeholders, icons (AA 4.5:1) |
| l10  | Text    | Primary body text                              |
| l11  | Text    | Emphatic text (headings, selected rows)        |

Selection lives off the gray ramp entirely; see 4.5. Disabled text is
`--pluto-text-disabled` (l9 at 45% alpha), not a slot.

## 4.1 - Value model

Values are generated in OKLCH from four sliders: floor lightness, per-slot chroma, band
step sizes, and text anchors. Even perceptual steps inside a band, deliberate jumps
between bands. The generator and verification live in working scripts; the checked-in
theme carries the resulting hex literals with role comments.

The ramps (all slider-tunable, none interview-locked):

```
Dark:  #040506 #0A0B0D #111315 #191C20 #23252A #2C2F34
       #36393F #44484D #5D6166 #A2A5A8 #CFD1D4 #F1F2F3
Light: #FDFDFF #F6F7F9 #EFF1F4 #E8EAED #DFE2E6 #D6D9DE
       #CCD0D5 #BBBEC3 #9EA2A7 #63666C #2E3034 #07080A
```

Verified properties: dark surface and fill steps run 3.6 to 4.0 OKLab points (previously
2 to 4 at an invisible floor), light surface steps 1.9 to 2.7 (previously 1.2 to 2.4),
l9 clears 4.5:1 against every surface in both themes (dark 8.2:1 on canvas, light
5.7:1), and the 22-point text chasms redistribute to 10 to 23 point steps that all land
above AA.

## 4.2 - Tint policy

Whisper tint, hue-locked to the Synnax primary (`#3774D0`, OKLCH hue 258). Chroma runs
0.002 to 0.010, peaking in the mid-tones. Rejected alternatives: pure gray (the old
ramp; reads dead), and assertive tint at Primer's chroma 0.014+ (visibly colors the UI
and competes with schematic and channel colors, which need neutral backdrops to pop). A
warmer light theme (Linear's 2026 refresh precedent) remains a tunable parameter, not a
lock.

## 4.3 - Deleting the `contrast` prop

With non-overlapping bands, a component's fixed slots work on every legal surface, so
per-surface indexing machinery is unnecessary. The `contrast` prop on Button (and its
pass-throughs), the `contrast-1/2/3` CSS blocks, and the dead emitted classes are
deleted. Call sites migrate to nothing.

One escape hatch survives: the `Menu.background`-style context, the single principled
surface link in the codebase, kept for chrome that must know it sits on elevated l2.
Rejected: a Spectrum-style contrast-indexed token matrix (13 grays per background
layer); the machinery cost is not justified when the band model removes the problem by
construction.

## 4.4 - Press policy

`:active` stays, with exactly one vocabulary: press is one ramp step past hover on the
fill. The fill band is exposed as three tokens (`--pluto-fill-rest`,
`--pluto-fill-hover`, `--pluto-fill-press`, mapping l3, l4, l5) consumed through the
variant-var protocol. A surface may re-point the ladder: elevated chrome steps it toward
the canvas so its controls recess instead of lighten. Text and shadow variants rest
transparent and join the band at hover. All seven old press vocabularies (hardcoded
fills, opacity drops, transform nudges, borrowed hover states) collapse into this rule;
components with no press feedback gain it. Linear's `scale(0.97)` flourish is rejected:
fractional scaling shimmers 0.5px hairline borders, and press stays purely in the color
system.

## 4.5 - Selection

Selection is a dedicated token family, not a gray slot:

- `--pluto-selected-bg`: `--pluto-primary-z-20`
- `--pluto-selected-active-bg`: `--pluto-primary-z-30`
- `--pluto-selected-border-color`: `--pluto-primary-z-45`
- `--pluto-selected-color`: `--pluto-primary-p2`

The fills are gray rotated toward the primary hue at unchanged lightness, so a selected
segment or toggle is tonally quiet but unambiguously "on"; the glyph shifts to a primary
tone and the border tints to match. List rows use a softer variant of the same rotation.

The interview originally merged pressed and selected at l5, matching Radix step 5. On
real controls the merge failed: a selected toggle in gray l5 was indistinguishable from
a pressed one and read as disabled next to its unselected siblings. Selection moved to
the tinted family after several showcase rounds; its exact feel is still open (section
8).

## 4.6 - Focus model

Two treatments, one geometry each:

1. **Editing focus** (text fields, `:focus-within`, always on): the existing border swap
   to primary plus the inset 0.5px shadow. `flush` inputs keep suppressing it.
2. **Keyboard focus** (everything else, `:focus-visible` only): one rule,
   `outline: 1px solid var(--pluto-primary-z); outline-offset: 2px`. The gap is
   transparent, so the painted-gap halos die along with their guessed gap colors. Primer
   and Radix Themes precedent; modern engines follow `border-radius` on outlines.

## 4.7 - Migration map

Old slots map to new slots by the role the site was actually using:

| Old slot (role as used)       | New slot            |
| ----------------------------- | ------------------- |
| l0, l1, l2 as surfaces        | same                |
| l1, l2 as component rest fill | l3                  |
| l2, l3 as hover fill          | l4                  |
| l3, l4 as pressed fill        | l5                  |
| any slot as selected fill     | 4.5 selected tokens |
| l4, l5 as border              | l6 or l7            |
| l6, l7 as border              | l7 or l8            |
| l7, l8 as text or icon        | l9                  |
| l9 as secondary text          | l9                  |
| l10, l11 as text              | same                |

Alpha variants follow their base slot. Bug fixes ride along: the l9 alpha generation bug
in `theming/css.ts`, the `--pluto-gray-l12` reference in `telem/control/Chip.tsx` and
the spec that locked it in, and the stale slot comment in `select/Button.css`.

# 5 - Implementation

Landed as a single-branch cutover, no coexistence: new ramps with role comments in
`theming/base/theme.ts`, the fill and selection token families in `theming/theme.css`,
`--pluto-text-disabled`, the css.ts and Chip fixes, contrast deletion, and the press and
focus unification across Pluto and Console consumers. The keyboard-focus rule is
declared by each adopting component rather than one global selector, so opting a
component in stays a local change. Showcase iteration on the slider parameters
continues; value changes are retunes, not re-architecture.

# 6 - What This RFC Does Not Cover

- Where color returns to the UI (the deliberate re-introduction map).
- A warmer light theme.
- Per-surface 0.5px box-shadow rings for the hairline corner artifact.
- Flush-input adoption in the command palette and project picker.

# 7 - Resolved Decisions

- **Deep re-architecture over values-only retune.** A retune inherits the role
  conflicts; every slot keeps colliding consumers. The trade is a large one-time
  migration, and it is accepted.
- **Selection left the gray ramp.** The interview locked pressed and selected merged at
  l5; implementation showed the merged gray selection reading as disabled on compact
  controls, and selection landed as the primary-tinted token family (4.5). The trade is
  real: selection is no longer a pure ramp step, and the tinted fills are one more token
  family to maintain.
- **Contrast prop deleted, not fixed.** Fixing it means maintaining a token matrix
  nobody else in our reference set carries. The trade: rare chrome needs the Menu-style
  surface context escape hatch.
- **Floor stays below the industry cluster.** Lifting to Linear or Radix levels reads
  brighter but sacrifices the dark-cockpit character and viz color pop. The trade: less
  headroom below l3, mitigated by 3.5+ point steps.
- **No transform on press.** Linear ships `scale(0.97)`; we reject it for
  hairline-border shimmer and consistency of the color-only state system.

# 8 - Open Questions

1. **Slider parameters.** Exact floor lightness, per-slot chroma (including a possible
   warmer light theme), band step sizes, and text anchors are tuned visually in the
   showcase inside the locked shape.
2. **Selection feel.** The tinted family in 4.5 is the fifth variant tried and still
   under visual tuning; the token names are the contract, their values are not.
