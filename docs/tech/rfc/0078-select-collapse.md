# 78 Select collapse

- **Author**: Emiliano Bonilla
- **Date**: 2026-10-03
- **Related**: [RFC 0067 - Select](0067-select-composition-api.md),
  [PR #3148 - SY-5069: Configure Arc timing from the Console](https://github.com/synnaxlabs/synnax/pull/3148)

## 0 Summary

As a select gets narrow, its name fades at the end, and the icon and caret stay in view.
A select can opt in to a second step: below a fixed width it shows only its icon, with
the name in a tooltip. The dropdown list stays inside the window.

## 1 Motivation

A select trigger is a flex row: icon, name, caret. The name was a bare text node, which
cannot shrink, so a narrow select pushed its caret out of the button. Each narrow layout
then hid parts by hand in its own CSS. The list never moved back inside the window and
never shrank to fit it.

## 2 Principles

1. **Lyra owns the behavior**: A layout sizes the select. It never hides the select's
   parts.
2. **CSS before measurement**: Each step is a CSS rule, not a resize observer.

## 3 Design

### 3.0 The name fades

`Select.Label` always renders the name in `Text.Text el="span" overflow="fade"`, as
`Tag` does. The frame of a trigger that shows a name gets `min-width: 10rem`, so it can
shrink below the name and still shows the icon and caret.

### 3.1 Opt-in collapse

`collapsible` on the single trigger makes it an inline-size container. Below `16rem` the
name and caret hide, and the icon centers. The caller must size the select, because
containment drops the name from the trigger's width. The tooltip shows the name only
when the name is hidden or cut off, checked on each hover through a function form of
`Tooltip.Dialog`'s `hide`. A fixed `Select.Item` gives its name as `textValue`, which
defaults to string children, as in Radix and React Spectrum. The search matches it too.

### 3.2 The list fits the window

`Dialog.Frame` keeps a non-modal dialog 6 px from the window edges and writes the space
left on its side to `--pluto-dialog-available-width` and
`--pluto-dialog-available-height`. Each height cap becomes `--pluto-dialog-max-height`,
and the dialog uses the smaller of the two.

### 3.3 The name includes the selection

A trigger with an `aria-label` and a selection sets `aria-labelledby` to itself and its
label, so a screen reader hears "Performance High", not only "Performance".

## 4 Implementation phases

- **Phase 0: All of §3.** One pull request, by request.

## 5 Resolved decisions

1. **Label in `Select.Label`, not `Dialog.Trigger`**: A wrapper around every trigger
   child hides an icon from `Text.isSquare`, so icon-only triggers lose their square
   shape. It also fades trailing content, such as the Core badge dot.
2. **Collapse is opt-in**: Container queries need the parent to set the width. A select
   that sizes to its name would shrink to zero. React Spectrum and Fluent measure with a
   resize observer instead. That works everywhere but costs a render pass on each
   resize. The trade is real.
3. **JS positioning stays**: CSS anchor positioning works in Chrome, Safari 26, and
   Firefox 147. On Linux, Tauri uses the system WebKitGTK, so support depends on the
   distribution.
4. **Fade, not ellipsis**: Radix, Spectrum, and Ant ellipsize. Lyra already fades in
   `Tag` and tree items, so selects match them.
