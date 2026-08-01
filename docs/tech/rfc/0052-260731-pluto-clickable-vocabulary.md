# 52 - Pluto Clickable Vocabulary

**Feature Name**: Pluto Clickable Vocabulary <br /> **Status**: Approved <br /> **Start
Date**: 2026-07-31 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC defines one vocabulary for every clickable thing in Pluto and Console. It
collapses `Button.Variant` to a pure emphasis ladder (`filled | outlined | text`),
extracts `preview` as a boolean modifier and `shadow` as an Input-only variant, replaces
the `ghost` prop with a container-owned reveal contract, closes selection styling into
three named tiers delivered through the button var protocol, codifies the div-chassis
Button as the sanctioned nested-interactive pattern with a hardened focus and activation
contract, and consolidates the console's hand-rolled house styles into single
definitions. Exact tier colors are tunable parameters; the axes, tiers, and contracts
are the shape.

# 1 - Motivation

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

# 2 - Vocabulary

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

# 3 - Principles

1. **Two axes, never conflated.** Variant answers "how loud is this control at rest, as
   an action." A selection tier answers "what does on look like for this family." Every
   selectable control is a coordinate (variant, tier). Selection is never expressed by
   swapping variants.
2. **Closed vocabularies.** Three variants, three tiers. A new rung or tier is an RFC
   amendment, not a feature decision. Features may re-point tier tokens in context; they
   may never write a selection `background` rule.
3. **One delivery mechanism.** All state paint flows through the button var protocol
   (`--pluto-bg`, `--pluto-border-color`, and their hover/active pairs), extending the
   press policy of RFC 0050. A tier is a token set plus one rule mapping tokens onto the
   protocol.
4. **The container owns nesting.** Interactive containers are Pluto-internal machinery
   reached through named components. Conflict handling and keyboard contracts are
   properties of the container pattern, not per-call-site inventions.
5. **Console composes; Pluto defines.** Console-only idioms get one named console
   definition built from Pluto parts. Pluto's vocabulary grows only for general-purpose
   needs.

# 4 - Design

## 4.1 - The emphasis ladder

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

## 4.2 - Selection tiers

Three fully-populated token sets live in `theming/theme.css` beside the fill band, each
with `bg`, `hover-bg`, `active-bg`, `border-color`, and `color` slots:

| tier    | speaks for              | today's basis                           |
| ------- | ----------------------- | --------------------------------------- |
| control | segments, toggles       | `--pluto-selected-*` (20/30/65 + p2)    |
| subtle  | tabs and tab-like chips | `--pluto-selected-subtle-*` (15/35)     |
| list    | rows: list, menu, tree  | `list/Item.css` chroma nudge, tokenized |

Delivery: one shared rule per tier re-points the six protocol vars. `select/Button.css`
and `button/Toggle.css` (byte-identical today) collapse into the single control-tier
rule. `pluto--checked` and `pluto--selected` merge into `pluto--selected`; ARIA stays
per-widget (`aria-pressed` vs `aria-selected`).

Context overrides re-point tier tokens only. `panel/Mosaic.css:69-70` (the overlaid
leaf's monochrome re-point) is the sanctioned example and survives unchanged in
mechanism.

Stray migration: `menu/Item.css` and the console nav rail fill speak the list tier (the
rail keeps its primary indicator ornament); the arc create modal speaks the control
tier; DateTime day cells and Steps get real states per 4.1. Table cell/header selection
is resolved in Phase 2: either the list tier fits or the table is documented here as the
one sanctioned exception (cells on a data grid are a canvas-adjacent surface; the
translucent-over-content constraint that forced the opaque flatten does not apply the
same way).

The control tier's exact colors remain an open showcase-judged parameter; the tier's
existence and slots are the contract.

## 4.3 - Reveal

The `ghost` prop and class are retired; the name is not reused for anything.

Reveal is a two-sided contract. The child action carries a marker (only marked children
hide; a row's icon and label never do). The container's CSS defines the trigger: hidden
at rest, revealed on container hover and on the marked child's own `:focus-visible`, so
the keyboard path is structural. One definition replaces the three current
implementations. Consumers keep prop-level ergonomics; whether the marker is a renamed
boolean on Button/Input or a class is an implementation parameter.

Sites: list-row delete/favorite actions, KeyValueEditor rows, label editor rows, and the
row-selection checkboxes currently using `ghost={!selected}`.

## 4.4 - The interactive container pattern

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
- The pack z-index rules (`theme.css:12-20`) drop their `button` tag qualification so
  div-chassis members participate.
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

## 4.5 - Checkbox and switch

A separate visual family, affirmed. Checked paint stays their own primary-fill language,
deliberately outside the tiers: a checked checkbox is form data, not selection. The
label-chassis DOM share (`el="label"` + `preventClick` wrapping a native input) is
behavior reuse and stays; it inherits the input-shell focus contract from 4.4. In
selectable rows the row speaks the list tier while the checkbox speaks checkbox
language; there is no double-painting.

## 4.6 - Console consolidation

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

## 4.7 - Kill list

`button/color.ts:parseColor`; the `.pluto-btn__trigger` CSS block; the unreachable
`.pluto--shadow` rule; the `pluto--highlight-hover` class emission;
`checkedVariant`/`uncheckedVariant`; the `ghost` prop, class, and all three reveal CSS
copies; Button's `--shadow` CSS block; `select/Button.css` and `button/Toggle.css` as
separate files' selected rules; the six stray selection rules; the four quiet-button CSS
copies; the three create-row copies; the duplicate toolbar button component and class;
two of the three filter-chip copies; the arc toolbar's `Empty.Action` copy.

# 5 - Implementation Phases

Each phase is green and independently reviewable; boundaries follow the architectural
seams (API, paint, behavior, console).

1. **Vocabulary and API.** Variant enum collapse, `preview` boolean, `shadow` to Input,
   `Input.Variant` decoupling, `checkedVariant` removal, DateTime/Steps state fixes,
   dead-code kills from 4.7 that have no dependents. Mechanical; no intended visual
   change outside DateTime/Steps.
2. **Selection tiers.** Token sets in `theme.css`, class merge, per-tier delivery rules,
   stray migration, the table-exception resolution. Visual changes confined to the six
   strays.
3. **Container pattern and reveal.** Chassis activation and focus contracts, pack
   z-index fix, conflict-mechanism cleanup, the reveal contract replacing `ghost`, the
   shared inline-glyph-action definition, spec coverage for the matrix.
4. **Console consolidation.** The 4.6 definitions and adoptions.

Compatibility: all changes are internal to the monorepo; no persisted data, wire format,
or external API is touched. Breaking TypeScript changes (variant unions, removed props)
are absorbed by the same-repo call-site migrations within each phase.

# 6 - Resolved Decisions

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
  Ownership had to move to the container to make the keyboard reveal structural; a
  marker prop may survive as ergonomics only.
- **DOM restructure for nested interactives** (sibling overlay, grid roles): rejected
  for now. `aria-actions` is a draft; the grid rework lives on the deferred a11y list.
  The trade is real: the div chassis is, by strict ARIA, a workaround, and we are
  hardening it rather than replacing it.
- **Promoting the quiet button into Pluto**: rejected for now. It is a color treatment
  expressible with existing props, used only by console today. Cheap to promote later if
  it spreads.
- **Form preview rendering plain Text instead of dead controls**: rejected. The
  per-input preview renderings are battle-tested and preserve layout.

# 7 - What This RFC Does Not Cover

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

# 8 - Open Questions

All four parameters were settled during implementation:

1. Reveal marker ergonomics: a `reveal` boolean prop on Button and Input emitting
   `pluto--reveal`; containers opt in with the `pluto--reveals` class.
2. Tier token naming: the control tier keeps the bare `--pluto-selected-*` names; subtle
   and list tiers are namespaced (`--pluto-selected-subtle-*`,
   `--pluto-selected-list-*`).
3. Table selection: sanctioned exception. Table cells are not Buttons and keep their own
   selection paint; migrating them would force the Button chassis onto a grid whose
   focus, spanning, and edit semantics it does not model.
4. Steps: the current step emits `pluto--selected` and takes the default control tier;
   future steps stay `disabled`.
