# 67 Select - Composition API

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-26
- **Related**: [RFC 0066 - Lyra](0066-lyra-primitives-package.md)

## 0 Summary

A select in Lyra takes its options as a data array and draws each one through a render
function. A three-option dropdown needs an entry type, a constant array, and the full
List data stack. This RFC makes composition the default: a static select writes its
options as `Select.Item` children, and a loaded select passes data to the same parts and
renders a `Select.Items` block. Both kinds mix in one dialog, so a fixed "Custom" option
sits next to loaded ranges without a sentinel key. The Tabs redesign (SY-4385) made the
same move and is the precedent throughout.

## 1 Motivation

- **Static options need data machinery**: `Select.Static`
  (`lyra/src/select/Static.tsx:60`) runs every three-option list through
  `List.useStaticData`, a fuzzy search index, and a render function that looks each
  entry up again with `List.useItem` (`:36`). About 150 call sites are static lists: 22
  `Select.Static`, 53 through `Form.buildSelectField`, 31 `Select.Buttons`, and the text
  and flex presets.
- **`Select.Buttons` repeats its children**: it takes `Select.Button` children and a
  `keys` array holding the same keys (`lyra/src/select/Button.tsx:46`).
  `pluto/src/telem/SelectTimeZone.tsx` passes `keys={TIME_ZONES}` and then writes the
  two buttons again.
- **Fixed options hide in the data array**: "Custom" in the range select
  (`console/src/platform/range/Select.tsx:81`), "New panel" in the panel picker
  (`console/src/feature/panel/MovePicker.tsx:50`), the "All" view
  (`console/src/platform/view/Frame.tsx:68`), and the built-in symbol groups
  (`console/src/feature/schematic/toolbar/Symbols.tsx:477`) are pushed into the key
  array, and each render function checks for the special key to draw it differently.

## 2 Vocabulary

- **Part**: One component of a composed select: `Select.Frame`, a trigger,
  `Select.Dialog`, `Select.Search`, `Select.List`, `Select.Items`, `Select.Item`.
- **Fixed item**: A `Select.Item` written as JSX, outside any `Select.Items` block.
- **Data block**: A `Select.Items` element. It renders the keys of the frame's data,
  virtualized or not, through a render function.
- **Shorthand**: A component that assembles the parts for a common case:
  `Select.Simple`, `Select.Buttons`, `Select.Single`, `Select.Multiple`.

## 3 Principles

1. **Static selects never take a data prop**: A fixed set of options is JSX. Data props
   exist only for data that is loaded or built at run time.
2. **One set of parts**: A static select and a loaded select use the same parts. The
   only differences are the data props on `Select.Frame` and whether the dialog holds
   items or a data block.
3. **State up, interaction local, capability by composition**: The Tabs rule from
   SY-4385. `Select.Frame` owns the value; parts own their own behavior; a caller adds a
   capability by adding a part, never a boolean.
4. **Nothing scales with the size of the data**: No part does work per data key that is
   not on screen. Fixed items are few by definition.

## 4 Design

### 4.0 The parts

```tsx
<Select.Frame value={key} onChange={setKey} {...data}>
  <Select.SingleTrigger />
  <Select.Dialog>
    <Select.Search onSearch={search} />
    <Select.List>
      <Select.Item itemKey="custom">Custom</Select.Item>
      <Select.Items emptyContent="No ranges">
        {(p) => <RangeItem {...p} />}
      </Select.Items>
    </Select.List>
    <Button.CreateListItem>New range</Button.CreateListItem>
  </Select.Dialog>
</Select.Frame>
```

| Part                     | Role                                               | Today                           |
| ------------------------ | -------------------------------------------------- | ------------------------------- |
| `Select.Frame`           | Value, `onChange`, `multiple`; optional data props | `Select.Frame`, data required   |
| `Select.SingleTrigger`   | Button showing the selected label                  | same, reads `List.useItem` only |
| `Select.MultipleTrigger` | Tags for each selected key                         | same, reads `List.useItem` only |
| `Select.Dialog`          | The floating surface                               | surface plus search plus list   |
| `Select.Search`          | Search field                                       | `SearchInput`                   |
| `Select.List`            | The dialog's scroll area                           | inside `Select.Dialog`          |
| `Select.Items`           | Data block: render function over the frame's keys  | `List.Items`                    |
| `Select.Item`            | One option, fixed or rendered by a data block      | `Select.Item`               |

The data props on `Select.Frame` are the ones it passes to `List.Frame` today: `data`,
`getItem`, `subscribe`, `virtual`, `itemHeight`, `overscan`, `onFetchMore`. They stay on
the frame rather than on `Select.Items` because the trigger, which sits outside the data
block, reads labels through `getItem`.

`Select.Items` is more than `List.Items` under a new name. It renders nothing while the
dialog is closed (§4.2), and it marks itself as the data block for the arrow-key walk
(§4.1).

Parts take plain strings, such as a trigger `placeholder` and a search `placeholder`.
Only the shorthands take `resourceName` and build those strings from it.

A frame with no dialog is still valid. Tree, Tabs, Mosaic and the 25 Console lists that
use `Select.Frame` for selection alone keep their shape.

### 4.1 Fixed items beside a data block

A fixed option is a `Select.Item` sibling of `Select.Items`. The arrow keys walk the
parts in page order. A fixed item is one step. The data block is its whole key array, in
order, whether or not its rows are mounted. A frame has one data source, so a dialog has
at most one data block. Movement wraps at both ends, and hidden items are skipped.

`Select.List` owns the walk. It finds the fixed items and the data block with a `data-*`
attribute, in DOM order, the way `Tabs.Selector` finds tabs
(`lyra/src/tabs/Selector.tsx:99`). Hover is stored as a key, not an index. Moving onto a
fixed item scrolls it into view; moving inside a data block calls its virtualizer's
`scrollToIndex`. The same ordered key list drives the shift-click range in a multiple
select and `autoSelectOnNone`.

The walk replaces the index model in `lyra/src/select/useHover.ts` and the
`List.useData().data` reads in `lyra/src/select/use.ts:116,172`.

A fixed item is not the same as a button. "New label" and "Connect server" are plain
dialog children: they are clicked, never selected, and the arrow keys skip them.

Headless UI refuses to mix fixed and virtualized options because the ordering is hard.
The trade is real, and the cost lives in one module.

### 4.2 The trigger label

`Dialog.Dialog` unmounts its children when closed (`lyra/src/dialog/Dialog.tsx:52`), so
the trigger cannot read a label from a rendered item.

- **Data keys**: The trigger reads the entry through `getItem` and shows its `name`, as
  `SingleTrigger` does today (`lyra/src/select/SingleTrigger.tsx:55`).
- **Fixed items**: While the dialog is closed, `Select.Dialog` still renders its
  children, but into a detached element that is never attached to the page. An
  unselected fixed item renders nothing. The selected one portals its children into a
  label slot that the trigger registers. In a multiple select, each tag registers its
  own slot. Other children, such as a create button, render into the detached element
  and cost no layout.

`Select.Items` renders nothing while closed, so a list of 10,000 channels costs nothing.

The label uses `createPortal`, not `Portal.In` (`lyra/src/portal/In.ts`). `Portal.In`
moves one element between hosts, and an open dialog shows the label in two places: the
row and the trigger.

Radix portals the selected `ItemText` into `Value` the same way, but keeps every item
mounted in hidden DOM while closed. We keep the portal and drop the hidden DOM.
Rejected: React 19.2 `<Activity mode="hidden">`, which keeps the hidden DOM and still
needs a way to move the label into the button; a label map written by the caller, which
brings back the data array; and React Aria's hidden collection pass, which costs work
per item on every change.

### 4.3 Search

`Select.Search` filters fixed items by their visible text, the way cmdk does, and calls
its `onSearch` prop for the data block when one is given. Each fixed item reads its
`textContent` after it mounts. A filtered-out item sets the `hidden` attribute rather
than unmounting, so its text stays readable when the term changes, and the walk skips
it. Rejected: a `textValue` prop on every item, because plain text children cover the
cases we have.

### 4.4 Scrolling

`Select.List` is the dialog's one scroll area. Fixed items and data blocks inside it
scroll together. Anything outside it, such as the search field or a create button, stays
pinned; a fixed option placed outside it is pinned and still reachable by arrow key.
Radix (`Viewport`) and Base UI (`List`) split the parts the same way.

This needs a change in List. `List.Items` today is both the scroll area and the item
renderer (`lyra/src/list/Items.tsx:102`, `lyra/src/list/Frame.tsx:307`). It splits in
two:

- **`List.Scroll`**: The scroll area. It owns `displayItems` sizing, and the frame
  observes it for fetch-more and virtualization.
- **`List.Items`**: Renders items only. It virtualizes against the nearest
  `List.Scroll`, passing TanStack Virtual's `scrollMargin` the items' `offsetTop` in the
  positioned scroll area, which accounts for fixed items above them.

`Select.List` is `List.Scroll` with the `listbox` role. The 32 files that render
`List.Items` gain a `List.Scroll` wrapper. Rejected: `List.Items` scrolling by itself
when no `List.Scroll` is above it. A forgotten wrapper would then work by accident with
the wrong scroll area instead of failing, so `List.Items` throws without one.

### 4.5 Shorthands

The shorthands assemble the parts. None of them takes a data prop for static options.

- **`Select.Simple`**: `Select.Item` children. It renders the frame, a single trigger,
  the dialog, search, and list. It replaces `Select.Static`, which is deleted.
- **`Select.Buttons`**: `Select.Item` children drawn as toggle buttons. The `keys` prop
  and the `Select.Button` part are deleted. `Select.Item` reads which container it is
  in, so switching a dropdown to a button row changes one word.
- **`Select.Single`** and **`Select.Multiple`**: The enhanced shorthands for loaded
  data: search, loading and error content, virtualization, and paging. They keep their
  props and are rebuilt on the parts. The Pluto domain selects, such as
  `Channel.SelectSingle`, keep using them.

```tsx
<Select.Simple value={v} onChange={setV} resourceName="variant">
  <Select.Item itemKey="success">
    <Icon.Success />
    Success
  </Select.Item>
  <Select.Item itemKey="error">
    <Icon.Error />
    Error
  </Select.Item>
</Select.Simple>
```

`Select.StaticEntry` goes with `Select.Static`; constants typed with it, such as
`VARIANT_DATA`, become `Select.Item` children or retype to `record.KeyedNamed`.

`Form.buildSelectField` builds on `Select.Simple`; its callers pass options as
`inputProps.children`. A list built at run time, such as CSV columns or EtherCAT PDOs,
maps its entries to `Select.Item` children inside `Select.Simple`.

Rejected: one component with a dropdown or buttons variant. Radix (`ToggleGroup`), React
Aria (`ToggleButtonGroup`), Base UI (`ToggleGroup`), Mantine (`SegmentedControl`), and
Ant Design (`Segmented`) all keep the two apart, and the two share almost no props.

### 4.6 Prior art

We checked React Aria, Radix, Base UI, Ariakit, Headless UI, cmdk, Mantine, and
Downshift. They fall into three patterns: a hidden render pass that builds the full
collection (React Aria), items that register when they mount (Radix, cmdk), and the same
parts with an optional data array (Base UI, Headless UI's `virtual`, Ariakit's
renderers, Mantine's `useVirtualizedCombobox`). We follow the third, because
`List.Frame` already owns ordered, virtualized data, and we add fixed items beside a
data block, which none of the third group supports.

### 4.7 What this RFC does not cover

- Typeahead (jumping to an option by typing while the search field is not focused).
- ARIA changes beyond the `listbox` role on `Select.List`, such as
  `aria-activedescendant`.
- Tree's own API. It keeps `Select.Frame` and moves onto `List.Scroll`.

## 5 Implementation phases

Each phase is one PR into `main`. No phase needs a flag: each one moves every caller of
what it changes. Every phase with new behavior lands with specs that pin it, and a phase
that renames a class updates the Playwright selectors in `integration/console`.

- **Phase 1: `List.Scroll`.** Add the part, the scroll context, and `scrollMargin`
  virtualization. `List.Items` uses the nearest `List.Scroll` and scrolls itself when
  there is none, only until Phase 3. Specs: a virtual list under fixed content scrolls
  to the right row, with `getBoundingClientRect` stubbed as the Mosaic specs do.
- **Phase 2: Wrap `List.Items` callers.** Mechanical, `review/bot`: 32 files.
- **Phase 3: Remove the `List.Items` fallback.** `List.Items` requires a `List.Scroll`.
- **Phase 4: Page-order navigation.** Replace the index model in `useHover` and `use.ts`
  with the key walk of §4.1. Existing selects have one data block and behave the same;
  the existing `useHover` and `use` specs must pass unchanged.
- **Phase 5: Rename `Select.Item` to `Select.Item`.** Mechanical, `review/bot`: 42
  call sites.
- **Phase 6: The parts.** `Select.Dialog` becomes the surface; add `Select.Search`,
  `Select.List`, `Select.Items`. Rebuild `Single` and `Multiple` on the parts; their
  specs must pass unchanged.
- **Phase 7: Fixed items.** Fixed `Select.Item` outside a data block, the detached
  closed render and trigger portal of §4.2, and fixed-item search of §4.3. Specs: the
  closed trigger shows a fixed label; arrow keys wrap from a fixed item through a
  virtual block and back; a filtered item is skipped.
- **Phase 8: `Select.Simple`.** Add it; move the Pluto and docs `Select.Static` callers.
- **Phase 9: Forms.** Move `Form.buildSelectField` to `Select.Simple` and its 53 Console
  callers; move the remaining Console `Select.Static` callers; delete `Select.Static`
  and `Select.StaticEntry`.
- **Phase 10: `Select.Buttons` children.** `Select.Item` children, delete `keys` and
  `Select.Button`; move 31 callers and the text and flex presets.
- **Phase 11: Remove sentinel keys.** Range "Custom", "New panel", the "All" view, and
  the built-in symbol groups become fixed items.

**Compatibility**: No stored or wire format changes. Every phase is a source change
inside the TypeScript packages.

## 6 Resolved decisions

1. **One set of parts, static and data modes**: Rejected React Aria's hidden collection
   pass (work per item on every change, heavy internals, and `List.Frame` already orders
   data) and mount registration alone (it cannot virtualize the 13 Pluto domain selects
   or Tree).
2. **Fixed options beside the data block**: Rejected keeping them as sentinel keys in
   the data array, which spreads special-key checks into every render function.
3. **Trigger label by portal from the selected fixed item**: See §4.2 for the rejected
   alternatives. The trade is real: no library does exactly this.
4. **Shorthands, not parts only**: Rejected deleting `Single`, `Multiple`, and a static
   shorthand outright, which turns a one-line select into five lines at 150 sites.
5. **Two components, not a variant**: See §4.5.
6. **Text search in `Select.Simple`**: Rejected dropping search, which removes the
   search box from every static select for keyboard users.
7. **`List.Scroll` split**: See §4.4.

## 7 Open questions

1. Whether `Select.Items` shows its `emptyContent` when fixed items exist but the data
   block is empty. Current lean: yes, in place of the block.
2. Whether `List.useCombinedData` survives Phase 11. The schematic symbol search list
   (`Symbols.tsx:526`) is a plain list, not a select, and still joins two sources.
