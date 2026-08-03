# 50 - Pluto Visual Language

**Feature Name**: Pluto Visual Language <br /> **Status**: Implemented <br /> **Start
Date**: 2026-07-29 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC re-architects the three systems that decide how every surface, control, and
corner in Pluto and Console looks.

**The gray ramp.** The 12 slots `--pluto-gray-l0` through `l11` each get exactly one
role in four non-overlapping bands, the ramp moves from pure gray to a whisper tint
derived from the Synnax primary hue, the values are rebuilt in OKLCH with even
perceptual steps, the `contrast` prop is deleted, and the press and keyboard-focus
vocabularies are unified.

**The clickable vocabulary.** `Button.Variant` collapses to a pure emphasis ladder
(`filled | outlined | text`), `preview` becomes a boolean modifier and `shadow` an
Input-only variant, the `ghost` prop is replaced by a container-owned reveal contract,
selection styling closes into three named tiers delivered through the button var
protocol, the div-chassis Button becomes the sanctioned nested-interactive pattern with
a hardened focus and activation contract, and the console's hand-rolled house styles
consolidate into single definitions.

**The radius scale.** Every ad-hoc corner radius is replaced by a five-step named ladder
generated from the theme spec: `tiny`, `small`, `medium`, `large`, and `huge`, rendering
4/6/9/12/18px at the default 6px base size.

Exact channel values, tier colors, and step sizes are tunable parameters. The band
roles, state rules, axes, tiers, contracts, and the ladder itself are the contract.

# 1 - Motivation

## 1.1 - The gray ramp

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

## 1.2 - Clickables

A four-track census (Button internals, Pluto clickable inventory, Console usage census,
external prior art) found structural problems:

1. **The variant enum mixes three unrelated kinds of thing.** `filled | outlined | text`
   are an emphasis ladder; `preview` is an interactivity mode that is never written
   literally (its only producer is form preview mode, `pluto/src/form/useField.ts:102`);
   `shadow` has zero Button consumers, and all eight production sites are edit-in-place
   Inputs whose CSS already half-lives in `input/Input.css:122-160`.
2. **Four selection languages plus six strays.** A control tier (`theme.css:49-64`), a
   subtle tier for tabs (`theme.css:69-78`), a list-row tier inlined in
   `list/Item.css:25-29`, and then raw one-offs: `menu/Item.css:17-19` (`primary-z-40`),
   the console nav rail (`Nav.css:16-37`), the arc create modal
   (`CreateModal.css:19-22`), table alphas (`table/Table.css:124`), DateTime day cells
   selecting by variant swap, and Steps expressing progress via `disabled`.
3. **`ghost` is a naming collision with the entire industry.** In every surveyed system,
   ghost names a transparent low-emphasis variant (our `text`). Our reveal-on-hover
   behavior is a row behavior everywhere else (VS Code tree actions, Linear rows), and
   ours has no home: defined in `list/Item.css:48-59`, duplicated in
   `KeyValueEditor.css` without the focus-visible reveal (keyboard-invisible delete
   buttons), and a third way in `label/Edit.css`.
4. **Nested interactives work by eight unnamed mechanisms**, re-derived per site:
   default stopPropagation, `propagateClick`, `preventClick`, per-site stopPropagation,
   dblclick shields, a CSS `:active:not(:has())` exclusion, pointer-events, and
   preventDefault-keep-drag. The pack z-index rules are `button`-tag-qualified
   (`theme.css:12-20`), so div-chassis Buttons silently miss them, and div-chassis
   Buttons have no built-in keyboard activation.
5. **Console re-derives house styles in CSS.** The quiet l9-to-l11 button four times,
   the pinned-create-row three times, three competing toolbar-button definitions with
   one class declared in two files, filter-chip CSS byte-identical in three files, and a
   hand-rolled copy of `Empty.Action`.
6. **Dead machinery.** `button/color.ts:parseColor` (unreferenced), the
   `.pluto-btn__trigger` CSS block (no emitter), the `.pluto--shadow` rule
   (unreachable), `pluto--highlight-hover` (emitted, zero CSS), and `Button.Toggle`'s
   `checkedVariant`/`uncheckedVariant` pair (both default outlined; paint comes from the
   checked class).

## 1.3 - Corner radii

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

# 2 - Vocabulary

- **Slot**: one of the 12 ramp positions l0 through l11.
- **Band**: a contiguous group of slots sharing a role family.
- **Surface**: a background something sits on (canvas, pane, dialog, chrome).
- **Fill**: the background of an interactive component itself.
- **Variant-var protocol**: the Button convention where variants set `--pluto-bg`,
  `--pluto-hover-bg`, `--pluto-active-bg` (and border equivalents) and shared rules swap
  them on hover and press.
- **Variant**: the rest-state emphasis of a control's chassis, chosen statically at the
  call site. Never changes at runtime.
- **Modifier**: an orthogonal boolean or enum prop that composes with any variant
  (`preview`, `status`, size, the reveal marker).
- **State**: a runtime condition with its own paint (`selected`, hover, press,
  disabled), layered over the variant.
- **Tier**: the named token set that paints the `selected` state for one family of
  controls.
- **Chassis**: the Button-rendered shell of a composite control (an input's frame, a
  tab, a tag, a list row), possibly rendered as a non-`button` element.
- **Interactive container**: a chassis that contains other interactive elements.
- **Reveal**: the container-owned behavior that hides marked child actions until the
  container is hovered or the action is keyboard-focused.
- **Step**: one rung of the radius ladder, named with a `Component.Size`.

# 3 - Principles

## 3.1 - The gray ramp

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

## 3.2 - Clickables

1. **Two axes, never conflated.** Variant answers "how loud is this control at rest, as
   an action." A selection tier answers "what does on look like for this family." Every
   selectable control is a coordinate (variant, tier). Selection is never expressed by
   swapping variants.
2. **Closed vocabularies.** Three variants, three tiers. A new rung or tier is an RFC
   amendment, not a feature decision. Features may re-point tier tokens in context; they
   may never write a selection `background` rule.
3. **One delivery mechanism.** All state paint flows through the button var protocol,
   extending the press policy of 4.4. A tier is a token set plus one rule mapping tokens
   onto the protocol.
4. **The container owns nesting.** Interactive containers are Pluto-internal machinery
   reached through named components. Conflict handling and keyboard contracts are
   properties of the container pattern, not per-call-site inventions.
5. **Console composes; Pluto defines.** Console-only idioms get one named console
   definition built from Pluto parts. Pluto's vocabulary grows only for general-purpose
   needs.

## 3.3 - Corner radii

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

# 4 - The gray ramp

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

Selection lives off the gray ramp entirely; see 5.2. Disabled text is
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

## 4.5 - Focus model

Two treatments, one geometry each:

1. **Editing focus** (text fields, `:focus-within`, always on): the existing border swap
   to primary plus the inset 0.5px shadow. `flush` inputs keep suppressing it.
2. **Keyboard focus** (everything else, `:focus-visible` only): one rule,
   `outline: 1px solid var(--pluto-primary-z); outline-offset: 2px`. The gap is
   transparent, so the painted-gap halos die along with their guessed gap colors. Primer
   and Radix Themes precedent; modern engines follow `border-radius` on outlines.

## 4.6 - Migration map

Old slots map to new slots by the role the site was actually using:

| Old slot (role as used)       | New slot            |
| ----------------------------- | ------------------- |
| l0, l1, l2 as surfaces        | same                |
| l1, l2 as component rest fill | l3                  |
| l2, l3 as hover fill          | l4                  |
| l3, l4 as pressed fill        | l5                  |
| any slot as selected fill     | 5.2 selection tiers |
| l4, l5 as border              | l6 or l7            |
| l6, l7 as border              | l7 or l8            |
| l7, l8 as text or icon        | l9                  |
| l9 as secondary text          | l9                  |
| l10, l11 as text              | same                |

Alpha variants follow their base slot. Bug fixes ride along: the l9 alpha generation bug
in `theming/css.ts`, the `--pluto-gray-l12` reference in `telem/control/Chip.tsx` and
the spec that locked it in, and the stale slot comment in `select/Button.css`.

# 5 - Clickable vocabulary

## 5.1 - The emphasis ladder

`Button.Variant = "filled" | "outlined" | "text"`. Default stays `outlined`.

- `filled`: the primary action. Keeps its custom-color machinery and the press-and-hold
  delay overlay.
- `outlined`: the standard control. The only variant whose rest fill reads the theme
  fill band (`--pluto-fill-rest`).
- `text`: the quiet action, transparent at rest. This is what the industry calls ghost;
  we keep the name `text` (roughly 130 call sites, Material precedent).

`preview` becomes a boolean modifier on Button and Input. It implies today's behavior:
`preventClick`, no focus ring, transparent chassis, `tabIndex -1`. Form plumbing becomes
`preview={mode === "preview"}`. Per-input preview renderings (Boolean's True/False text,
Numeric hiding its drag handle, Select's trigger mapping) are kept.

`shadow` moves to Input. `Input.Variant` decouples from Button into its own enum:
`"outlined" | "text" | "shadow"`. For an input, an invisible chassis until hover is a
legitimate chassis rung (Fluent's `transparent` input appearance). Button's `--shadow`
CSS block dies with the unreachable `.pluto--shadow` rule; the shadow styling
consolidates in `input/Input.css`.

`Button.Toggle` loses `checkedVariant`/`uncheckedVariant`. A toggle is an `outlined`
chassis whose checked state speaks the control tier; the variant never swaps. DateTime
day cells migrate from variant-swap selection to a real selected state. Steps stops
expressing progress through `disabled` and gets a real current/complete treatment during
migration.

`status` remains the orthogonal tone axis crossing all variants (the Polaris
variant-times-tone shape).

## 5.2 - Selection tiers

Selection is a dedicated token family, not a gray slot. Three fully-populated token sets
live in `theming/theme.css` beside the fill band, each with `bg`, `hover-bg`,
`active-bg`, `border-color`, and `color` slots:

| Tier    | Speaks for                            | Basis                              |
| ------- | ------------------------------------- | ---------------------------------- |
| control | segments, toggles, buttons by default | primary over l1 at 15/20/65        |
| subtle  | selected tabs in unfocused panes      | neutral grays: l4, l5, l6, l7, l11 |
| list    | rows: list, menu, tree                | l5 alphas, chroma boosted 0.025    |

The control tier is also the pane-focused tab's paint, and its glyph sits at
`primary-z`.

The control tier's fills are gray rotated toward the primary hue at unchanged lightness,
so a selected segment or toggle is tonally quiet but unambiguously "on"; the glyph
shifts to a primary tone and the border tints to match. Primary flattens are opaque over
l1 so plot pixels never bleed through a selected chip. The subtle tier is deliberately
neutral: primary is reserved for the focused pane's tab, which speaks the control tier.
The list tier leaves text color untouched because rows carry multi-color content.

Delivery: one shared rule per tier re-points the six protocol vars. `select/Button.css`
and `button/Toggle.css` (byte-identical today) collapse into the single control-tier
rule. `pluto--checked` and `pluto--selected` merge into `pluto--selected`; ARIA stays
per-widget (`aria-pressed` vs `aria-selected`).

Context overrides re-point tier tokens only. `panel/Mosaic.css` (the overlaid leaf's
monochrome re-point) is the sanctioned example and survives unchanged in mechanism.

Stray migration: `menu/Item.css` and the console nav rail fill speak the list tier (the
rail keeps its primary indicator ornament); the arc create modal speaks the control
tier; DateTime day cells and Steps get real states per 5.1. Table cells are a sanctioned
exception and keep their own selection paint.

The tier colors remain open showcase-judged parameters; the tiers' existence and slots
are the contract.

## 5.3 - Reveal

The `ghost` prop and class are retired; the name is not reused for anything.

Reveal is a two-sided contract. The child action carries a marker (only marked children
hide; a row's icon and label never do). The container's CSS defines the trigger: hidden
at rest, revealed on container hover and on the marked child's own `:focus-visible`, so
the keyboard path is structural. One definition replaces the three current
implementations. The marker is a `reveal` boolean prop on Button and Input emitting
`pluto--reveal`; containers opt in with the `pluto--reveals` class.

Sites: list-row delete/favorite actions, KeyValueEditor rows, label editor rows, and the
row-selection checkboxes currently using `ghost={!selected}`.

## 5.4 - The interactive container pattern

The div-chassis Button with manually restored semantics is the named pattern. No DOM
restructure: ARIA has no legal nested-button DOM, and the surveyed alternatives (sibling
plus draft `aria-actions`, grid-role rework) either lack support or live on the deferred
a11y list.

The conflict zoo shrinks to three sanctioned mechanisms:

1. **Child wins by default.** Every Button stops propagation. Manual `stopPropagation`
   on Buttons is redundant and migrates to nothing.
2. **Parent yields or child shares, deliberately.** `preventClick` dead-zones a
   container that must yield to its children (selected tab, Boolean label).
   `propagateClick` lets a child share its click with the row (menu items, metadata
   rows).
3. **Non-Button nested controls shield themselves.** Native inputs, drag handles, and
   editables keep explicit stopPropagation, named as part of the pattern.

Hardening, part of this RFC's deliverable:

- The chassis owns keyboard activation: a Button rendered as a non-`button` element that
  is focusable activates on Enter/Space from the component, not from per-consumer
  keydown handlers. Tabs' hand-rolled handler migrates onto it.
- Each container archetype declares a focus contract: tab (roving tabIndex, Delete
  closes), row (container focusable, revealed actions reachable via focus-visible), tag
  and input-shell (shell not focusable, inner control is).
- `el="div"` is Pluto-internal. Console reaches the pattern only through named
  components; the one stray (`platform/task/controls/Status.tsx`) is absorbed.
- The pack z-index rules drop their `button` tag qualification so div-chassis members
  participate.
- The propagation and activation matrix gets spec coverage in `button/` and container
  specs, so a regression is a red test.

**Inline glyph actions.** The micro-buttons nested inside chassis (tab close, tag close,
the legend visibility toggle) are a named sub-pattern of the container: a small square
`text` Button with fill feedback suppressed and all feedback on the glyph (l9 rest, l11
hover, `error-z` press when destructive), `tabIndex -1`, hidden and shown via the reveal
contract with per-container animation freedom, keyboard path on the parent. Not a
variant (fill-suppressed `text`) and not a tier (nothing is selected). One shared
definition replaces the chromeless `!important` blocks in `tabs/Tabs.css` and
`tag/Tag.css`.

## 5.5 - Checkbox and switch

A separate visual family, affirmed. Checked paint stays their own primary-fill language,
deliberately outside the tiers: a checked checkbox is form data, not selection. The
label-chassis DOM share (`el="label"` + `preventClick` wrapping a native input) is
behavior reuse and stays; it inherits the input-shell focus contract from 5.4. In
selectable rows the row speaks the list tier while the checkbox speaks checkbox
language; there is no double-painting.

## 5.6 - Console consolidation

One named definition per idiom, at the console platform layer:

- **Quiet button** (l9 text, l11 on hover): one definition; the four CSS copies die.
- **Pinned create-row** (button dressed as the terminal list row): one primitive; the
  three geometry copies die.
- **Toolbar button**: the `ToolbarButton` / `CopyLinkToolbarButton` twins collapse into
  the `Toolbar.Action` family; the double-declared `.console-toolbar-button` class dies.
- **Filter chips**: one class replaces the three byte-identical copies.
- The arc toolbar adopts `Empty.Action` instead of its hand-rolled copy.

Feature wrappers with real logic (TareButton, EnableDisableButton, StartStopButton,
ConfigureButton) stay where they are.

## 5.7 - Kill list

`button/color.ts:parseColor`; the `.pluto-btn__trigger` CSS block; the unreachable
`.pluto--shadow` rule; the `pluto--highlight-hover` class emission;
`checkedVariant`/`uncheckedVariant`; the `ghost` prop, class, and all three reveal CSS
copies; Button's `--shadow` CSS block; `select/Button.css` and `button/Toggle.css` as
separate files' selected rules; the six stray selection rules; the four quiet-button CSS
copies; the three create-row copies; the duplicate toolbar button component and class;
two of the three filter-chip copies; the arc toolbar's `Empty.Action` copy.

# 6 - Radius scale

## 6.1 - Tokens

`theme.sizes.border.radius` is a five-field object of rem multiples
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

## 6.2 - API

`rounded?: boolean | number | Component.Size` on `Flex.Box`. `true` applies the default
class, a size name applies `pluto--rounded-<size>` (class-based, so pack corner
inheritance and seam-squaring rules can still override it; the inline-style number form
defeats those and once forced an `!important` in the task controls bar), and a number
remains an inline `${n}rem` escape hatch for derived one-offs.

## 6.3 - Role assignment

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

# 7 - Implementation

Landed as a single-branch cutover, no coexistence.

**Ramp.** New ramps with role comments in `theming/base/theme.ts`, the fill and
selection token families in `theming/theme.css`, `--pluto-text-disabled`, the css.ts and
Chip fixes, contrast deletion, and the press and focus unification across Pluto and
Console consumers. The keyboard-focus rule is declared by each adopting component rather
than one global selector, so opting a component in stays a local change.

**Clickables.** Four seams, each green on its own: the vocabulary and API changes
(variant enum collapse, `preview` boolean, `shadow` to Input, `Input.Variant`
decoupling, `checkedVariant` removal, DateTime and Steps state fixes, dependent-free
kills from 5.7); the selection tiers (token sets, class merge, per-tier delivery rules,
stray migration); the container pattern and reveal (chassis activation and focus
contracts, pack z-index fix, conflict-mechanism cleanup, the reveal contract replacing
`ghost`, the shared inline-glyph-action definition, spec coverage for the matrix); and
the console consolidation of 5.6.

**Radius.** Token generation and regenerated static CSS, the `rounded` size form,
migration of every CSS literal and numeric `rounded` prop to steps, and the kill list:
`--pluto-pack-br` (dead token), `CSS.rounded`/`CSS.sharp` helpers (dead, with inverted
defaults relative to `Flex.Box`), the `0.66666rem` hack, the legend's duplicate radius
declaration, the task form's dead `rounded` prop, and the label-select create row that
hand-copied `Button.Create`'s geometry. Deliberate pixel changes shipped with the
migration: tooltip and Monaco hover 8px to 6px, mosaic create button 9px to 6px, arc
type chip and gradient-stop marker 3px to 4px, nav connection badge 4px to 6px.

Compatibility: all changes are internal to the monorepo; no persisted data, wire format,
or external API is touched. Breaking TypeScript changes (variant unions, removed props)
are absorbed by same-repo call-site migrations in the same change.

# 8 - What This RFC Does Not Cover

- Where color returns to the UI (the deliberate re-introduction map).
- A warmer light theme.
- Per-surface 0.5px box-shadow rings for the hairline corner artifact.
- Flush-input adoption in the command palette and project picker.
- Retuning the control tier's colors (open showcase parameter, tracked in the
  selected-state tuning thread).
- The deferred a11y program: listbox/tree ARIA, roving rows, dialog focus traps,
  keyboard paths for drag-only affordances.
- Tabs geometry and strip behaviors (resize-on-close, vertical tabs) from the tabs
  second pass.
- Schematic symbol primitives that bypass Button by design (the raw-`button` toggle
  symbol, macOS traffic lights).
- Dialog, menu-frame, and list virtualization behavior; only their selection paint is in
  scope.

# 9 - Resolved Decisions

## 9.1 - The gray ramp

- **Deep re-architecture over values-only retune.** A retune inherits the role
  conflicts; every slot keeps colliding consumers. The trade is a large one-time
  migration, and it is accepted.
- **Selection left the gray ramp.** The interview locked pressed and selected merged at
  l5; implementation showed the merged gray selection reading as disabled on compact
  controls, and selection landed as the tinted tier family (5.2). The trade is real:
  selection is no longer a pure ramp step, and the tinted fills are one more token
  family to maintain.
- **Contrast prop deleted, not fixed.** Fixing it means maintaining a token matrix
  nobody else in our reference set carries. The trade: rare chrome needs the Menu-style
  surface context escape hatch.
- **Floor stays below the industry cluster.** Lifting to Linear or Radix levels reads
  brighter but sacrifices the dark-cockpit character and viz color pop. The trade: less
  headroom below l3, mitigated by 3.5+ point steps.
- **No transform on press.** Linear ships `scale(0.97)`; we reject it for
  hairline-border shimmer and consistency of the color-only state system.

## 9.2 - Clickables

- **Fused variant enum kept growing** (add rungs for selected-tab, quiet, etc.):
  rejected. Selection is a runtime state; encoding it as a variant forces
  variant-swapping (the DateTime fossil) and cannot express selected-hover and
  selected-press without a cross-product leak (Carbon's `danger--tertiary`). The trade
  is real: a selected tab visually sits between filled and outlined, and the two-ladder
  model asks readers to hold two loudness scales.
- **Collapsing to fewer selection tiers** (tabs adopt the control tier): rejected. It
  would re-litigate two user-approved looks (tabs, list rows) to fix the one unapproved
  tier, and the industry deliberately keeps toggle, tab, and list selection as distinct
  languages.
- **Renaming `text` to `ghost`** for industry alignment: rejected. Roughly 130 call
  sites of churn, and reusing a name that meant something else here for years plants a
  second confusion.
- **Reveal as a Button prop under a new name**: rejected as the defining mechanism.
  Ownership had to move to the container to make the keyboard reveal structural; the
  marker prop survives as ergonomics only.
- **DOM restructure for nested interactives** (sibling overlay, grid roles): rejected
  for now. `aria-actions` is a draft; the grid rework lives on the deferred a11y list.
  The trade is real: the div chassis is, by strict ARIA, a workaround, and we are
  hardening it rather than replacing it.
- **Promoting the quiet button into Pluto**: rejected for now. It is a color treatment
  expressible with existing props, used only by console today. Cheap to promote later if
  it spreads.
- **Form preview rendering plain Text instead of dead controls**: rejected. The
  per-input preview renderings are battle-tested and preserve layout.
- **Table selection as a fourth tier**: rejected. Table cells are not Buttons and keep
  their own selection paint; migrating them would force the Button chassis onto a grid
  whose focus, spanning, and edit semantics it does not model.

## 9.3 - Corner radii

- **Rem over px.** A px ladder preserved 4px on a clean grid but froze corners while
  every other dimension scales; the codebase already contained a hand-rolled scaling
  4px. The trade is one off-grid fraction (2/3rem), confined to the theme spec.
- **4px kept.** A pure 0.5rem-grid ladder (3/6/9/12) was rejected: the 4px control
  radius is the established default and the user chose to keep it, accepting the
  fractional rem step.
- **Size names over role names.** Role-named tokens (control/chip/card/panel) would
  self-document but would be the only role-named size tokens in the system and would
  fight the `Component.Size` vocabulary. The role map lives in this RFC instead.
- **Default points at tiny.** The lexical oddity ("the default radius is tiny") was
  accepted over shifting names up and coining a sixth name for 3rem.
- **Number form kept.** Dropping `rounded`'s number form entirely was considered; kept
  as an escape hatch for derived values, with size names as the house style.

# 10 - Open Questions

1. **Ramp slider parameters.** Exact floor lightness, per-slot chroma (including a
   possible warmer light theme), band step sizes, and text anchors are tuned visually in
   the showcase inside the locked shape.
2. **Selection feel.** The tier values in 5.2 are still under visual tuning; the token
   names and slots are the contract, their values are not.

A future density or zoom setting would exercise the radius ladder's rem scaling; nothing
here blocks or requires it.
