# 70 Color picker

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-27
- **Related**: [RFC 0061 - Theme-resolved color](0061-theme-resolved-color.md),
  [RFC 0066 - Lyra primitives package](0066-lyra-primitives-package.md)

## 0 Summary

The color picker drops `react-color` and becomes a Lyra component built from scratch. It
is one panel: an HSV editor at the top, then theme and palette swatches, then recent
colors below a divider. An Auto button clears an optional color back to the theme. The
browser's native picker is never used.

## 1 Motivation

- **Consistency takes effort**: most picks in Synnax come from a small set: a red or a
  green for a schematic state, a label color, a line color. The old picker put a large
  gradient first and presets in a thin strip, so users tuned slightly different reds by
  hand.
- **Absence is unreachable**: RFC 0061 makes color fields optional, with absence meaning
  the theme picks. The picker had no way back to absent. The log toolbar passed an
  `onDelete` for this, and `Color.Swatch` dropped it silently.
- **Styles fight the dependency**: `Picker.css` held a dozen `!important` rules and
  structural selectors (`> div:nth-child(2)`) to restyle `SketchPicker`'s inline styles.
- **The dependency is dead weight**: `react-color` is unmaintained, and Pluto was its
  only user.

## 2 Prior art

Figma and Chrome DevTools put an HSV square first, then hue and alpha sliders, a format
menu, and an eyedropper; their users mostly make new colors. Grafana splits "Colors" and
"Custom" into two tabs, which hides the custom path behind a click. Linear and GitHub
labels offer swatches plus a hex box and nothing else. We take the Figma editor and put
both swatch groups below it, with no tabs, so a preset and a custom color are each one
click away.

## 3 Principles

1. **One panel, no tabs**: every action is at most one click away. The panel is taller
   in exchange.
2. **Reuse is one click**: presets and recent colors are always in view. The editor is
   for the color that is not there yet.
3. **The stored value stays RGBA**: HSV is the input model only. No wire or storage
   format changes.
4. **The picker is Synnax-blind**: the whole `Color` namespace lives in Lyra (RFC 0066).

## 4 Design

### 4.0 Layout

From the top:

- **Plane**: the saturation and value square. Saturation runs left to right, value runs
  bottom to top.
- **Hue slider** and **alpha slider**: round thumbs on rounded tracks. The alpha track
  sits on a checkerboard.
- **Input row**: a hex box, an alpha percent box, a copy button, and an eyedropper.
- **Swatches**: below a divider, the Auto button (optional fields only, two cells wide,
  with the button selection tier), the theme's primary, secondary, warning, and error
  colors, then the visualization palette. A grid of ten columns. The swatch that equals
  the value carries `--pluto-selected-ring`, a new selection tier for items whose fill
  is their content: a gray-l11 ring outside a focus-ring-offset gap.
- **Recent**: the user's recent colors, below a divider, shown only when there are any.

### 4.1 Color model

The editor holds the color as HSVA. RGB cannot carry a hue through gray, black, or
white, so the held HSVA keeps the hue when the user drags to an edge and back. The held
value resyncs only when the caller's value changes to a color the HSVA does not produce.

A move of the plane or the hue on a fully transparent color makes it opaque. Otherwise
the move changes nothing visible.

`x` gains `color.hsva` and `color.fromHSVA` (unrounded, so round trips are exact).
`color.fromCSS` gains 4-digit hex, space syntax with a slash alpha, percentages, and
`hsl()`.

### 4.2 Input row

- **Hex box**: shows six digits without a `#`. A complete 6 or 8 digit hex applies as
  the user types. Shorter forms apply on Enter or blur, since `fff` is also the start of
  `ffffff`. A paste of any CSS color (`rgb(255 0 0 / 50%)`, `#f00`, `red`) applies at
  once. A hex without alpha digits keeps the current alpha. Anything that does not parse
  reverts on blur.
- **Alpha box**: a percentage from 0 to 100, with the drag handle every Lyra numeric
  input has.
- **Copy**: copies the full hex, alpha digits included when the color is translucent.
- **Eyedropper**: shown only where the `EyeDropper` API exists (Chromium: the Console in
  Chrome, and Windows WebView2). WebKit has no such API, so macOS and Linux desktop
  builds do not show it. Escape inside the eyedropper is a normal exit, not an error.

### 4.3 Auto

`Color.Swatch` takes a `fallback`: the color the theme paints while the value is absent.
A `fallback` makes the value optional:

- The picker leads its grid with an Auto button. A click calls `onChange(undefined)`.
- While the value is absent, the trigger shows the fallback with an Auto icon, and the
  tooltip says the theme picks the color.
- Without a `fallback`, the value is required and the Auto button never shows.

The prop types are a union, so a required caller keeps
`onChange: (value: color.Color) => void` and never sees `undefined`.

### 4.4 Color input

`Color.Input` joins a swatch to a hex text box, the shape Figma, Sketch, Penpot, and
Webflow use for a fill. The user types or pastes a color into the box, or clicks the
swatch to open the picker. An empty box means Auto: the box shows "Auto" and the swatch
shows the fallback with the Auto icon. `Color.Field`, every form color, uses it. The
bare `Color.Swatch` stays for dense places: the line plot line list, the control legend,
and table cells.

Every swatch is a square at the standard control height (`--pluto-height-<size>`), so it
lines up with labels and inputs. A translucent border shows on every color and surface.

### 4.5 Keyboard and pointer

The plane and both sliders take focus and have the `slider` role. Arrow keys move one
percent, or ten with Shift. A press sets the value at once, with no drag threshold, and
the drag captures the pointer.

## 5 Implementation phases

- **Phase 1**: one pull request. It adds the `x` conversions, moves `Color` from Pluto
  to Lyra, replaces the picker, adds `fallback` to the optional callers, and removes
  `react-color`. It updates the unit, integration, and Studio selectors that named the
  `react-color` DOM.

The move and the rewrite share a pull request because a Lyra `Color` that still imports
`react-color` would put the dependency in Lyra for one release.

## 6 Resolved decisions

1. **One panel over tabs**: tabs make the panel shorter but put a click in front of
   either half. The trade is real: the panel is about 30 rem tall.
2. **Swatches at the bottom**: presets and recent colors sit together below the editor,
   split by a divider, so all one-click picks are in one place. The trade is real: a
   preset is further from the trigger than with a swatches-first order.
3. **HSV over OKLCH**: an OKLCH square is perceptually even, but its out-of-gamut edge
   is irregular and reads as a defect to a first-time user. The theme already corrects
   lightness at render (RFC 0061 §2), so OKLCH gives less here than in a design tool.
   The trade is real: equal steps in the square do not look equal.
4. **Hex and alpha only**: separate R, G, and B boxes and a format menu were rejected.
   Paste of any CSS color covers bringing a color in; copy covers taking it out.
5. **Eyedropper where supported, no native plugin**: a Tauri plugin could sample the
   screen on WebKit, but it needs the screen capture permission on macOS. The trade is
   real: most desktop users never see the button.
6. **Auto only on optional fields**: a required field offering Auto would fail
   validation on click.

## 7 Open questions

None.
