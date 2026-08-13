# 50 - Pluto Visual Language

**Feature Name**: Pluto Visual Language <br /> **Status**: Implemented <br /> **Start
Date**: 2026-07-29 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC re-architects the three systems that decide how every surface, control, and
corner in Pluto and Console looks.

**The gray ramp.** The 12 slots `--pluto-gray-l0` through `l11` each get exactly one
role in four non-overlapping bands, the ramp moves from pure gray to a whisper tint
derived from the Synnax primary hue, the values are rebuilt in OKLCH, the `contrast`
prop is deleted, and the press and keyboard-focus vocabularies are unified.

**The clickable vocabulary.** `Button.Variant` collapses to a pure emphasis ladder
(`filled | outlined | text`), `preview` becomes a boolean modifier and `shadow` an
Input-only variant, the `ghost` prop is replaced by a container-owned reveal contract,
selection closes into three named tiers, and the div-chassis Button becomes the
sanctioned nested-interactive pattern with a hardened focus and activation contract.

**The radius scale.** Every ad-hoc corner radius is replaced by a five-step named ladder
generated from the theme spec: `tiny`, `small`, `medium`, `large`, and `huge`, rendering
4/6/9/12/18px at the default 6px base size.

Channel values, tier colors, and step sizes are tunable. The band roles, state rules,
axes, tiers, and the ladder are the contract.

# 1 - Motivation

## 1.1 - The gray ramp

A census of the pre-redesign ramp found structural problems, not value problems:

1. **Slots have no single role.** The light ramp's own comments label l4 through l7 as
   borders, but l4 is a de facto pressed fill, l6 carries five conflicting roles, and l7
   has been colonized as faint text. Tuning any slot for one consumer breaks another.
2. **Surfaces and component fills share slots.** Elevated chrome and component rest fill
   both landed on l2, producing the invisible-control class of bugs.
3. **Contrast is broken at both ends.** The dark l1 and l2 surface steps are nearly
   invisible (2 to 4 OKLab points at a floor of L 0.085) while the text band jumps 22
   points at once. Secondary text at l8 fails WCAG AA on both themes (2.9:1 light, 4.0:1
   dark); placeholders are not exempt under 1.4.3.
4. **Seven press vocabularies and four keyboard-focus geometries** coexist, two
   components have no press feedback, and the focus halos paint their gap color with a
   guessed surface level (Switch guesses l1 where Checkbox guesses l0).
5. **The `contrast` prop is half machinery.** Only values 1 through 3 have CSS behind
   them, `filled` ignores it, and the only principled surface link in the codebase is
   the `Menu.background` context.
6. **Known bugs.** `theming/css.ts` generates the l9 alpha variants from l11, and
   `telem/control/Chip.tsx` references a nonexistent `--pluto-gray-l12`.

## 1.2 - Clickables

1. **The variant enum mixes three unrelated kinds of thing.** `filled | outlined | text`
   are an emphasis ladder; `preview` is an interactivity mode never written literally
   (its only producer is `form/useField.ts`); `shadow` has zero Button consumers and
   eight edit-in-place Input sites whose CSS already half-lives in `input/Input.css`.
2. **Four selection languages plus six strays.** A control tier, a subtle tier for tabs,
   a list-row tier inlined in `list/Item.css`, and then raw one-offs: `menu/Item.css`,
   the console nav rail, the arc create modal, table alphas, DateTime day cells
   selecting by variant swap, and Steps expressing progress via `disabled`.
3. **`ghost` is a naming collision with the entire industry.** Everywhere else, ghost
   names a transparent low-emphasis variant (our `text`), and reveal-on-hover is a row
   behavior. Ours has no home: defined in `list/Item.css`, duplicated in
   `KeyValueEditor.css` without the focus-visible reveal (keyboard-invisible delete
   buttons), and a third way in `label/Edit.css`.
4. **Nested interactives work by eight unnamed mechanisms**, re-derived per site:
   default stopPropagation, `propagateClick`, `preventClick`, per-site stopPropagation,
   dblclick shields, a CSS `:active:not(:has())` exclusion, pointer-events, and
   preventDefault-keep-drag. The pack z-index rules are `button`-tag-qualified, so
   div-chassis Buttons silently miss them and have no built-in keyboard activation.
5. **Console re-derives house styles in CSS.** The quiet l9-to-l11 button four times,
   the pinned create-row three times, three competing toolbar-button definitions with
   one class declared in two files, filter chips byte-identical in three files, and a
   hand-rolled copy of `Empty.Action`.
6. **Dead machinery.** `button/color.ts:parseColor`, the `.pluto-btn__trigger` block (no
   emitter), the `.pluto--shadow` rule (unreachable), `pluto--highlight-hover` (emitted,
   zero CSS), and `Button.Toggle`'s `checkedVariant`/`uncheckedVariant` pair (both
   default outlined; paint comes from the checked class).

## 1.3 - Corner radii

An audit found roughly 13 distinct corner sizes reached through two parallel unit
systems: the px-emitted `--pluto-border-radius` token (frozen at 4px) and rem literals
plus numeric `rounded` props, which scale with the base size. The two coincide only at
the default base. The ladder already wanted to exist: a hand-computed `0.66666rem`
faking a scaling 4px, a hardcoded `--pluto-pack-br: 6px` mirroring a `1rem` dialog
radius, 4px spelled four different ways, and filled buttons, tabs, and tags all drifting
to 6px while outlined buttons stayed at 4px. The chip role alone spanned 3/4/6/9px, with
6px and 9px in the same tab strip.

Tailwind, Material 3, Radix Themes, Fluent 2, and Primer all use a small named ladder
with roughly 1.5x steps. We align with Primer's shape, extended by the steps the audit
showed in real use.

# 2 - Vocabulary

- **Slot**: one of the 12 ramp positions. **Band**: a contiguous group of slots sharing
  a role family.
- **Surface**: a background something sits on (canvas, pane, dialog, chrome). **Fill**:
  the background of an interactive component itself.
- **Variant-var protocol**: the Button convention where variants set `--pluto-bg`,
  `--pluto-hover-bg`, `--pluto-active-bg` (and border equivalents) and shared rules swap
  them on hover and press.
- **Variant**: the rest-state emphasis of a control's chassis, chosen statically at the
  call site. **Tier**: the token set that paints the `selected` state for one family of
  controls.
- **Chassis**: the Button-rendered shell of a composite control (an input's frame, a
  tab, a tag, a list row), possibly rendered as a non-`button` element. An **interactive
  container** is a chassis that contains other interactive elements.
- **Reveal**: the container-owned behavior that hides marked child actions until the
  container is hovered or the action is keyboard-focused.

# 3 - Principles

## 3.1 - The gray ramp

1. **One slot, one role.** A consumer that wants a different look moves to the slot
   whose role matches, never bends a slot's value.
2. **Bands do not overlap.** Surfaces stop at l2; fills start at l3. A component at rest
   is always one full step from any legal surface, so per-surface indexing is
   unnecessary (the Radix and Linear model of absolute steps).
3. **Dark cockpit.** The dark canvas stays dark. Contrast comes from step spacing above
   the floor, not from lifting the floor to the industry cluster.
4. **Quiet but present states.** Hover, press, and selection are each one deliberate
   step, expressed only through tokens. No transforms, no opacity tricks, no bespoke
   hexes.
5. **Whisper temperature.** The ramp carries a trace of the primary hue, strongest in
   the mid-tones, near zero at the extremes. Grays never read as blue.
6. **AA is a floor, not a goal.** Every text slot clears 4.5:1 on every surface it may
   legally sit on, placeholders included. Disabled text is exempt and becomes an alpha
   token rather than a slot.

## 3.2 - Clickables

1. **Two axes, never conflated.** Variant answers "how loud is this control at rest, as
   an action." A tier answers "what does on look like for this family." Every selectable
   control is a coordinate (variant, tier).
2. **Closed vocabularies.** Three variants, three tiers. A new rung or tier is an RFC
   amendment. Features may re-point tier tokens in context; they may never write a
   selection `background` rule.
3. **One delivery mechanism.** All state paint flows through the variant-var protocol. A
   tier is a token set plus one rule mapping tokens onto the protocol.
4. **The container owns nesting.** Conflict handling and keyboard contracts are
   properties of the container pattern, not per-call-site inventions.
5. **Console composes; Pluto defines.** Console-only idioms get one named console
   definition built from Pluto parts. Pluto's vocabulary grows only for general-purpose
   needs.

## 3.3 - Corner radii

1. **One ladder.** Every chrome corner speaks a named step or derives from one via
   `calc`.
2. **Corners scale with density.** Heights, padding, and type are rem; corners are too.
   `round(Nrem, 1px)` keeps them crisp at any base size.
3. **Roles are stated once, here,** not encoded in token names. Size names match
   `Component.Size`, the same vocabulary as the height ladder.
4. **Proportional details stay local.** Radii that track a component's own geometry
   (checkbox indicators, stadium pills, 50% circles, the schematic vessel percent
   system) are not ladder steps.

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
between bands. Whisper tint, hue-locked to the Synnax primary (`#3774D0`, OKLCH hue
258), with chroma running 0.002 to 0.010 and peaking in the mid-tones.

```
Dark:  #040506 #0A0B0D #111315 #191C20 #23252A #2C2F34
       #36393F #44484D #5D6166 #A2A5A8 #CFD1D4 #F1F2F3
Light: #FDFDFF #F6F7F9 #EFF1F4 #E8EAED #DFE2E6 #D6D9DE
       #CCD0D5 #BBBEC3 #9EA2A7 #63666C #2E3034 #07080A
```

Dark surface and fill steps run 3.6 to 4.0 OKLab points, light steps 1.9 to 2.7, l9
clears 4.5:1 against every surface in both themes, and the 22-point text chasms
redistribute to steps that all land above AA.

## 4.2 - Deleting the `contrast` prop

With non-overlapping bands, a component's fixed slots work on every legal surface, so
per-surface indexing machinery is unnecessary. The `contrast` prop, the `contrast-1/2/3`
CSS blocks, and the dead emitted classes are deleted; call sites migrate to nothing. One
escape hatch survives: the `Menu.background`-style context, kept for chrome that must
know it sits on elevated l2.

## 4.3 - Press policy

Press is one ramp step past hover on the fill. The fill band is exposed as three tokens
(`--pluto-fill-rest`, `--pluto-fill-hover`, `--pluto-fill-press`, mapping l3, l4, l5)
consumed through the variant-var protocol. A surface may re-point the ladder: elevated
chrome steps it toward the canvas so its controls recess instead of lighten. Text and
shadow variants rest transparent and join the band at hover. All seven old press
vocabularies collapse into this rule; components with no press feedback gain it.

## 4.4 - Focus model

Two treatments, one geometry each:

1. **Editing focus** (text fields, `:focus-within`, always on): the existing border swap
   to primary plus the inset 0.5px shadow. `flush` inputs keep suppressing it.
2. **Keyboard focus** (everything else, `:focus-visible` only): one rule,
   `outline: 1px solid var(--pluto-primary-z); outline-offset: 2px`. The gap is
   transparent, so the painted-gap halos die along with their guessed gap colors.

## 4.5 - Migration map

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

Alpha variants follow their base slot.

# 5 - Clickable vocabulary

## 5.1 - The emphasis ladder

`Button.Variant = "filled" | "outlined" | "text"`. Default stays `outlined`.

- `filled`: the primary action. Keeps its custom-color machinery and the press-and-hold
  delay overlay.
- `outlined`: the standard control. The only variant whose rest fill reads the theme
  fill band.
- `text`: the quiet action, transparent at rest. This is what the industry calls ghost;
  we keep the name `text` (roughly 130 call sites, Material precedent).

`preview` becomes a boolean modifier on Button and Input, implying today's behavior:
`preventClick`, no focus ring, transparent chassis, `tabIndex -1`. Per-input preview
renderings (Boolean's True/False text, Numeric hiding its drag handle, Select's trigger
mapping) are kept. `shadow` moves to Input, whose variant enum decouples from Button
into `"outlined" | "text" | "shadow"`: for an input, an invisible chassis until hover is
a legitimate rung (Fluent's `transparent` input appearance). `status` remains the
orthogonal tone axis crossing all variants.

`Button.Toggle` loses `checkedVariant`/`uncheckedVariant`. A toggle is an `outlined`
chassis whose checked state speaks the control tier; the variant never swaps. DateTime
day cells migrate from variant-swap selection to a real selected state, and Steps stops
expressing progress through `disabled`.

## 5.2 - Selection tiers

Selection is a dedicated token family, not a gray slot. Three token sets live in
`theming/theme.css` beside the fill band, each with `bg`, `hover-bg`, `active-bg`,
`border-color`, and `color` slots:

| Tier    | Speaks for                            | Basis                              |
| ------- | ------------------------------------- | ---------------------------------- |
| control | segments, toggles, buttons by default | primary over l1 at 15/20/65        |
| subtle  | selected tabs in unfocused panes      | neutral grays: l4, l5, l6, l7, l11 |
| list    | rows: list, menu, tree                | l5 alphas, chroma boosted 0.025    |

The control tier is gray rotated toward the primary hue at unchanged lightness, so a
selected segment or toggle is tonally quiet but unambiguously "on". It also paints the
pane-focused tab, which is why the subtle tier stays neutral. Primary flattens are
opaque over l1 so plot pixels never bleed through a selected chip. The list tier leaves
text color untouched because rows carry multi-color content.

One shared rule per tier re-points the six protocol vars. `select/Button.css` and
`button/Toggle.css` (byte-identical today) collapse into the single control-tier rule.
`pluto--checked` and `pluto--selected` merge into `pluto--selected`; ARIA stays
per-widget. Context overrides re-point tier tokens only; `panel/Mosaic.css`'s monochrome
re-point of the overlaid leaf is the sanctioned example.

Strays migrate: `menu/Item.css` and the console nav rail fill speak the list tier (the
rail keeps its primary indicator ornament), and the arc create modal speaks the control
tier. Table cells are an exception and keep their own selection paint.

## 5.3 - Reveal

The `ghost` prop and class are retired; the name is not reused. Reveal replaces them as
a two-sided contract. The child action carries a marker, a `reveal` boolean on Button
and Input emitting `pluto--reveal`, so only marked children hide and a row's icon and
label never do. The container opts in with `pluto--reveals` and its CSS defines the
trigger: hidden at rest, revealed on container hover and on the marked child's own
`:focus-visible`, so the keyboard path is structural. One definition replaces three.
Sites: list-row delete and favorite actions, KeyValueEditor rows, label editor rows, and
the row-selection checkboxes previously using `ghost={!selected}`.

## 5.4 - The interactive container pattern

The div-chassis Button with manually restored semantics is the named pattern. No DOM
restructure: ARIA has no legal nested-button DOM, and the alternatives (sibling plus
draft `aria-actions`, grid-role rework) either lack support or live on the deferred a11y
list.

The conflict zoo shrinks to three sanctioned mechanisms. **Child wins by default**:
every Button stops propagation, so manual `stopPropagation` on Buttons migrates to
nothing. **Parent yields or child shares, deliberately**: `preventClick` dead-zones a
container that must yield to its children (selected tab, Boolean label), and
`propagateClick` lets a child share its click with the row (menu items, metadata rows).
**Non-Button nested controls shield themselves**: native inputs, drag handles, and
editables keep explicit stopPropagation.

The chassis owns keyboard activation: a focusable Button rendered as a non-`button`
element activates on Enter/Space from the component, and Tabs' hand-rolled handler
migrates onto it. Each archetype declares a focus contract: tab (roving tabIndex, Delete
closes), row (container focusable, revealed actions reachable via focus-visible), tag
and input-shell (shell not focusable, inner control is). `el="div"` is Pluto-internal;
console reaches the pattern only through named components. The pack z-index rules drop
their `button` tag qualification so div-chassis members participate.

**Inline glyph actions.** The micro-buttons nested inside chassis (tab close, tag close,
the legend visibility toggle) are a named sub-pattern: a small square `text` Button with
fill feedback suppressed and all feedback on the glyph (l9 rest, l11 hover, `error-z`
press when destructive), `tabIndex -1`, revealed via 5.3, keyboard path on the parent.
Not a variant and not a tier. One shared definition replaces the chromeless `!important`
blocks in `tabs/Tabs.css` and `tag/Tag.css`.

## 5.5 - Checkbox and switch

Checked paint stays their own primary-fill language, outside the tiers: a checked
checkbox is form data, not selection. The label-chassis DOM share (`el="label"` +
`preventClick` wrapping a native input) stays and inherits the input-shell focus
contract. In selectable rows the row speaks the list tier while the checkbox speaks
checkbox language; there is no double-painting.

## 5.6 - Console consolidation

One named definition per idiom, at the console platform layer:

- **Quiet button** (l9 text, l11 on hover): the four CSS copies die.
- **Pinned create-row** (button dressed as the terminal list row): the three geometry
  copies die.
- **Toolbar button**: the `ToolbarButton` / `CopyLinkToolbarButton` twins collapse into
  the `Toolbar.Action` family; the double-declared `.console-toolbar-button` class dies.
- **Filter chips**: one class replaces three byte-identical copies.
- The arc toolbar adopts `Empty.Action` instead of its hand-rolled copy.

Feature wrappers with real logic (TareButton, EnableDisableButton, StartStopButton,
ConfigureButton) stay where they are.

# 6 - Radius scale

## 6.1 - Tokens

`theme.sizes.border.radius` is a five-field object of rem multiples emitted by
`toCSSVars` as:

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
The canvas mirror computes `Math.round(radius.tiny * base)` px.

## 6.2 - API

`rounded?: boolean | number | Component.Size` on `Flex.Box`. `true` applies the default
class and a size name applies `pluto--rounded-<size>`. Both are class-based, so pack
corner inheritance and seam-squaring rules can still override them; the number form
stays an inline `${n}rem` escape hatch for derived one-offs.

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

Landed as a single-branch cutover, no coexistence. The ramps and token families are
generated into `theming/base/theme.ts` and `theming/theme.css`; every CSS literal and
numeric `rounded` prop migrates to a step. The keyboard-focus rule is declared by each
adopting component rather than one global selector, so opting a component in stays a
local change. The propagation and activation matrix gains spec coverage.

The kill list is everything named dead in 1.2, plus the three reveal CSS copies,
Button's `--shadow` block, the six stray selection rules, the duplicated quiet-button,
create-row, toolbar-button, and filter-chip definitions, `--pluto-pack-br`, the
`CSS.rounded` and `CSS.sharp` helpers, and the `0.66666rem` hack.

Deliberate pixel changes beyond the token swap: tooltip and Monaco hover 8px to 6px,
mosaic create button 9px to 6px, arc type chip and gradient-stop marker 3px to 4px, nav
connection badge 4px to 6px. All changes are internal to the monorepo; no persisted
data, wire format, or external API is touched.

# 8 - What This RFC Does Not Cover

- Where color returns to the UI, and a possible warmer light theme.
- The deferred a11y program: listbox and tree ARIA, roving rows, dialog focus traps,
  keyboard paths for drag-only affordances.
- Tabs geometry and strip behaviors (resize-on-close, vertical tabs).
- Schematic symbol primitives that bypass Button by design (the raw-`button` toggle
  symbol, macOS traffic lights).
- Dialog, menu-frame, and list behavior; only their selection paint is in scope.

# 9 - Resolved Decisions

## 9.1 - The gray ramp

- **Deep re-architecture over values-only retune.** A retune inherits the role
  conflicts; every slot keeps colliding consumers. The cost is a large one-time
  migration.
- **Selection left the gray ramp.** The interview locked pressed and selected merged at
  l5. On real controls a selected toggle in gray l5 was indistinguishable from a pressed
  one and read as disabled beside its unselected siblings. Selection became the tinted
  tier family, at the price of one more token family to maintain.
- **Contrast prop deleted, not fixed.** Fixing it means maintaining a Spectrum-style
  contrast-indexed token matrix nobody else in our reference set carries.
- **Floor stays below the industry cluster.** Linear sits at OKLab L 0.139 and Radix
  slate at 0.179; we float l0 at ~0.115. Lifting to their levels reads brighter but
  sacrifices the dark-cockpit character and viz color pop.
- **Whisper tint over assertive tint.** Primer's chroma 0.014+ visibly colors the UI and
  competes with schematic and channel colors, which need neutral backdrops to pop.
- **No transform on press.** Linear ships `scale(0.97)`; fractional scaling shimmers
  0.5px hairline borders, and press stays purely in the color system.

## 9.2 - Clickables

- **Growing the fused variant enum** (rungs for selected-tab, quiet, and so on):
  rejected. Encoding a runtime state as a variant forces variant-swapping (the DateTime
  fossil) and cannot express selected-hover and selected-press without a cross-product
  leak (Carbon's `danger--tertiary`). The cost is that a selected tab visually sits
  between filled and outlined, so readers hold two loudness scales.
- **Collapsing to fewer selection tiers** (tabs adopt the control tier): rejected. It
  would re-litigate two approved looks to fix the one unapproved tier, and the industry
  deliberately keeps toggle, tab, and list selection as distinct languages.
- **Renaming `text` to `ghost`** for industry alignment: rejected. Roughly 130 call
  sites of churn, and reusing a name that meant something else here for years plants a
  second confusion.
- **Reveal as a Button prop under a new name**: rejected as the defining mechanism.
  Ownership had to move to the container to make the keyboard reveal structural.
- **DOM restructure for nested interactives**: rejected for now. `aria-actions` is a
  draft and the grid rework lives on the deferred a11y list. By strict ARIA the div
  chassis is a workaround, and we are hardening it rather than replacing it.
- **Table selection as a fourth tier**: rejected. Migrating table cells would force the
  Button chassis onto a grid whose focus, spanning, and edit semantics it does not
  model.

## 9.3 - Corner radii

- **Rem over px.** A px ladder preserved 4px on a clean grid but froze corners while
  every other dimension scales, and the codebase already contained a hand-rolled scaling
  4px. The cost is one off-grid fraction (2/3rem), confined to the theme spec.
- **4px kept.** A pure 0.5rem-grid ladder (3/6/9/12) was rejected: 4px is the
  established control radius, and keeping it is worth the fractional rem step.
- **Size names over role names.** Role-named tokens (control/chip/card/panel) would
  self-document but would be the only role-named size tokens in the system and would
  fight the `Component.Size` vocabulary. The role map lives in 6.3 instead.
