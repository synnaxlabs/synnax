# Schematic Symbol Style Guide

## Design Philosophy

Synnax schematics are mission-critical operational displays. They need to be clear
enough for high-stakes decision-making and beautiful enough that operators and
engineers feel confident and proud of what they've built. These are not competing
goals. The visual quality of a schematic directly affects trust, readability, and the
willingness of users to invest time in building great layouts.

The target aesthetic is the intersection of modern developer tool polish (Linear,
Vercel, Stripe, JetBrains) and functional industrial display design (HP-HMI
principles, ISA-101). Not the austere gray-only approach of traditional HP-HMI, and
not the skeuomorphic 3D pipes of legacy SCADA. Something that feels engineered,
precise, and modern.

### Taste References

- **Linear**: dark mode excellence, precise typography, engineering confidence
- **Vercel**: clean, purposeful, every element earns its place
- **Stripe**: subtle depth, refined borders, clear hierarchy
- **JetBrains**: information-dense but organized, professional tooling aesthetic

### Guiding Principles

1. **Visual cohesion above all.** The biggest current problem is disjointedness.
   Every symbol, whether it is a valve, a value readout, or a status light, must feel
   like it belongs to the same design system and to the wider Synnax UI. Consistent
   stroke weights, proportions, corner treatments, and spacing.

2. **Beauty and clarity are the same thing.** Users should be able to choose color
   for both aesthetic and functional reasons. The system should make it easy to build
   schematics that are beautiful by default, not require expertise to avoid ugly
   results. Color is a tool for both meaning and beauty, not restricted to one.

3. **Precision is polish.** The difference between "functional" and "refined" is in
   pixel-perfect geometry, systematic spacing, aligned connection points, and
   considered typography. Engineers notice misalignment. It erodes trust.

4. **The value is the hero.** In data display symbols, the actual data (the number,
   the state name) should be the most visually prominent element. Containers, borders,
   labels, and units are supporting elements that should never compete with the data.

5. **Depth without skeuomorphism.** Flat + thin borders alone can feel flimsy. But 3D
   bevels and gradients feel dated. The right answer is subtle depth cues: a soft
   shadow, a transparent fill, a slight background tint. Enough to give elements
   presence without making them feel like physical objects.

6. **Part of Synnax, not apart from it.** Schematics should feel native to the wider
   console UI, not like a separate embedded application. The visual language should
   relate to Pluto's design system. Exactly how is an open question that requires
   experimentation.

## Current Problems

### What's Wrong Today

- **Borders feel too thick.** The 2px borders on data display symbols (value, state
  indicator) give them a heavy, 90s-widget feel.
- **Units badge steals attention.** The solid-background units section in the value
  display prioritizes the units over the actual value. The colored badge competes with
  the number.
- **Data symbols look like generic UI widgets.** Value, state indicator, and light
  feel like they were borrowed from a form builder rather than designed for a
  schematic.
- **Engineering symbols are decent but plain.** The 2px outline with transparent
  interior is functional but boring. There may be opportunities to add visual richness
  without sacrificing clarity.
- **Tanks are just boxes.** They don't visually communicate "vessel" or
  "containment." A fill level indicator would help (planned future feature).
- **Grouping panels look clunky.** When users surround groups of symbols with boxes
  and backgrounds (e.g., "Oil Cart", "Generator Speed Controls"), the result tends to
  look heavy and unrefined.
- **Overall disjointedness.** The schematic doesn't feel like a cohesive whole, and
  it doesn't feel integrated with the rest of the Synnax console.

### What's Working

- The engineering symbols (valves, pumps, flowmeters) follow recognizable standards
  and are generally well-proportioned.
- The color system foundation (user-assigned colors, contrast-aware text) is flexible
  and sound.
- `vectorEffect: "non-scaling-stroke"` keeps line weights consistent at any zoom.
- The handle/connection point system works well.
- The schematic background (blank in operational mode, dot grid in edit mode) is fine.

## Border & Depth Strategy

The current 2px borders are too heavy, but simply thinning them without compensation
makes symbols feel flimsy. The solution is to reduce border weight while adding subtle
depth through other means.

### Approach (Requires Experimentation)

Multiple options to prototype and compare:

- **Thinner border + subtle background fill.** A 1px border with a very faint
  tinted or transparent fill inside. The fill gives the element presence without
  relying on a heavy border.
- **Thinner border + soft shadow.** A 1px border with a subtle box-shadow. This
  adds depth in a way that feels modern (Linear, Stripe both use this) but needs
  testing in the schematic context.
- **Thinner border + transparent fill of the border color.** The interior becomes a
  very low-opacity version of the outline color. This was specifically mentioned as
  a direction that could feel beautiful and modern.

The right answer likely varies by symbol type. Engineering symbols (valves) may want
a different treatment than data display symbols (value). Experimentation will
determine what works.

## Color Philosophy

### Not Pure Gray, Not Unrestrained Color

Synnax does NOT follow the strict HP-HMI "gray by default" philosophy. Users should
be free to use color for both beauty and function. However, the system should make it
easy to use color well:

- Carefully chosen color for a mix of beauty and functionality.
- Users can make their schematics generally gray if they want (they already can).
- The default symbol color provides a neutral starting point.
- Alarm colors (red, amber, yellow) retain their semantic power because they are
  distinct from the aesthetic colors users choose.

### Color for Data

- Live values should be visually distinct from static labels and units.
- The value itself should be the most prominent element, not its container.
- Units and secondary information should be lower contrast.

### Alarm Colors

| Priority | Color | Usage |
|----------|-------|-------|
| Critical | Red | Immediate action required |
| Warning | Amber/Orange | Attention needed soon |
| Advisory | Yellow | Awareness, no immediate action |

## Symbol-Specific Design

### Value Display

The most common data symbol. Shows a live telemetry reading with units.

**Current problems:**
- 2px border feels heavy, like a 90s widget.
- Solid-background units badge competes with the actual value.
- Visually indistinguishable from the state indicator.

**Design direction:**
- The numeric value is the hero. Bold, prominent, in a tabular-figures font so
  digits don't shift as values change.
- Units displayed quieter: lighter weight, lower contrast, adjacent to but visually
  subordinate to the number.
- Container should be minimal. Reduce border weight and explore alternatives to the
  solid-background units badge.
- Multiple variations should be prototyped and compared before committing to a
  direction. Options range from "units as quiet suffix on the same line" to "units
  in a visually separated but non-competing section."

**Typography:**
- Tabular figures (fixed-width numerals) for values.
- Only meaningful precision. Don't display digits that change too fast to be useful.

### Light (Status Indicator)

Binary on/off indicator. Currently a flat SVG circle.

**Current problems:**
- Plain circle is too simple, doesn't convey "indicator" or "status."
- Equal visual weight in on and off states.
- No visual richness.

**Design direction (Apollo-inspired):**
- Consider a **rectangular indicator with text label**, inspired by Apollo-era
  annunciator panels. The label tells the operator what the light means without
  needing a separate text element.
- Interior fill: a transparent-ish variant of the border color, creating depth.
- **On state**: fills with the assigned color, label becomes readable against it.
  Possibly a very subtle glow (1-2px soft shadow in the indicator color).
- **Off state**: muted, low-contrast. Present but not demanding attention.
- This approach makes the light more informative (built-in label) and more visually
  distinctive than a bare circle.
- The relationship between the light and the state indicator should be considered.
  They are both about showing status (binary vs. multi-state). They may benefit from
  a shared visual language, but this needs experimentation.

### State Indicator

Shows which of several discrete states equipment is in (e.g., "Running", "Fault").

**Current problems:**
- Looks nearly identical to the value display (bordered rectangle with text).
- Always shows a colored background, even during normal states.
- No visual hierarchy between normal and abnormal states.

**Design direction:**
- Must be visually distinct from the value display. Value shows a number with units.
  State indicator shows a categorical label.
- Consider a **status pill** shape: compact, more rounded than the value display.
- Normal states could be muted. Abnormal states take on semantic alarm colors.
- May share visual language with the redesigned light symbol, since both communicate
  status. Needs experimentation.

### Engineering Symbols (Valves, Pumps, Flowmeters)

Generally well-executed. The direction is refinement, not redesign.

**Current assessment:** decent but the 2px outline + transparent interior is "a bit
boring." There may be ways to add visual richness.

**Potential directions:**
- Subtle interior treatment: a very faint fill, a transparent tint of the stroke
  color, or a slight gradient that adds depth without becoming skeuomorphic.
- Audit stroke consistency across all symbols.
- Standardize proportions within each family (all valves share bounding box ratios,
  all pumps share circle diameters).
- Pixel-perfect geometry: ensure symmetry, consistent angles, clean intersections.

### Tanks & Vessels

**Current problem:** tanks are basically boxes. They don't visually communicate
"vessel" or "containment."

**Design direction:**
- The silhouette should suggest a vessel. Rounded bottom edges, subtle top
  treatment, or other cues that distinguish a tank from a generic rectangle.
- Fill level indicator is a planned future feature that will significantly improve
  the tank's visual communication.
- Default proportions should feel balanced and vessel-like.

### Grouping Panels

**Current problem:** when users surround groups of symbols with boxes and labeled
backgrounds (e.g., "Oil Cart," "Facility Controls"), the result looks clunky.

**Design direction:**
- Panels need a refined treatment: thinner borders, subtle backgrounds, clean label
  positioning.
- Should feel like a Vercel or Linear card component, not a Windows XP group box.
- The panel should recede behind its contents, providing organization without
  competing visually with the symbols inside it.

## Dark Mode

Both light and dark modes should feel equally polished. The same relationships
apply:

- Symbols are neutral by default.
- User-chosen colors work well against either background.
- The contrast calculation system already handles this for text.
- Specific color values will differ between modes but the visual relationships
  should be consistent.

## Animation & Interaction

- **Hover states**: subtle increase in contrast or highlight. Not dramatic.
- **State transitions**: smooth, brief (150-200ms). No bouncing or overshooting.
- **Active/pressed**: slightly more prominent than hover, still restrained.
- Running indicators (if added): very understated. A slow visual cue, not a
  spinning animation.

## Accessibility

- Never rely on color alone to communicate state. Pair with shape, text, or
  position.
- All text and meaningful visual elements must meet WCAG AA contrast requirements.
- Alarm colors should be distinguishable under common forms of color vision
  deficiency. Pairing colors with distinct shapes is one proven approach.

## Implementation Strategy

### Approach

Gradual execution driven by a cohesive vision. Each change should move toward the
unified style, not just fix one symbol in isolation. The order:

1. **Establish the visual direction** with 2-3 prototype variations of the value
   display, light, and state indicator. Compare options visually in the showcase.
2. **Pick a direction** based on how the prototypes look together and alongside
   existing engineering symbols.
3. **Apply the chosen direction** to the target symbols (value, light, state
   indicator).
4. **Extend to engineering symbols** with refinements that bring them into the same
   visual family.
5. **Address grouping panels** and other supporting elements.
6. **Audit and polish** across all symbols for consistency.

### Breaking Changes

Existing schematics may look different after updates. This is acceptable. Visual
improvement is the priority over backwards visual compatibility.

### Showcase

A symbol showcase page exists in the Pluto dev server (`pnpm dev:pluto`, select
"Schematic Symbols"). This displays every symbol organized by category and should be
used as the primary visual comparison tool during the redesign process.

## Open Questions

- What is the right border weight for data display symbols? 1px? 1.5px? Depends on
  whether we add other depth cues?
- Should the light and state indicator share a visual language, or remain distinct
  symbol types?
- What interior treatment works for engineering symbols? Transparent tint? Subtle
  fill? Needs prototyping.
- How should grouping panels be styled to feel refined rather than clunky?
- Should the value display container have a border at all, or can spacing and
  typography alone define it?
- What specific font or font-feature-settings should be used for tabular figures in
  value displays?
