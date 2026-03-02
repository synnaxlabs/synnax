# Company Page Layout Options

All options use the same content blocks. The differences are in **layout, visual
treatment, and spatial rhythm**.

## Content Blocks (shared)

- **Mission statement** — h3, grid-background
- **YC badge** — logo + "Backed by Y Combinator"
- **Engineers-from logos** — NASA, SpaceX, Lockheed Martin, Firefly
- **Founder cards** — photo, name, title, LinkedIn

---

## Option A: Asymmetric Bento Grid

Testimonials-style layout. Mission card spans the left column full-height,
YC + engineers-from stack on the right. Founders below in a separate row with
gradient-masked decorative border around the whole grid.

```
┌──────────────────────────┬───────────────────┐
│                          │                   │
│  MISSION STATEMENT       │  BACKED BY YC     │
│  (grid bg, full height)  │  logo centered    │
│  h3 + CTA button         │                   │
│                          ├───────────────────┤
│                          │  ENGINEERS FROM   │
│                          │  4 logos, 2×2     │
│                          │                   │
├──────────────────────────┴───────────────────┤
│                                              │
│  ┌─────────────────┐  ┌─────────────────┐   │
│  │  [photo]         │  │  [photo]         │  │
│  │  Emiliano        │  │  Patrick         │  │
│  │  CEO             │  │  CTO             │  │
│  └─────────────────┘  └─────────────────┘   │
│                                              │
└──────────────────────────────────────────────┘
```

- Decorative gradient-masked border (::before, fades diagonally like testimonials)
- Noise texture overlay on mission card
- Cards use box-shadow depth, not flat borders
- Radial gradient bg field behind the whole section

---

## Option B: Full-Bleed Hero + Credibility Strip

Mission statement breaks out of the card — large type floating on a radial
gradient field (like the homepage hero). Credibility in a single horizontal
strip. Founders in colored gradient cards (like testimonial cards).

```
┌──────────────────────────────────────────────┐
│                                              │
│  (radial gradient bg field + noise)          │
│                                              │
│  "Our team is grounded in an obsession..."   │
│  h2, light weight, max-width 900px           │
│                                              │
│  [Get In Touch →]                            │
│                                              │
├──────────────────────────────────────────────┤
│  [YC]  ·  NASA  ·  SpaceX  ·  LM  ·  FF    │
│  single row, all monochrome, dividers        │
├──────────────────────────────────────────────┤
│                                              │
│  ┌─────────────────┐  ┌─────────────────┐   │
│  │ gradient card    │  │ gradient card    │  │
│  │ (brand color)    │  │ (brand color)    │  │
│  │ [photo]          │  │ [photo]          │  │
│  │ name + title     │  │ name + title     │  │
│  │ noise overlay    │  │ noise overlay    │  │
│  └─────────────────┘  └─────────────────┘   │
│                                              │
└──────────────────────────────────────────────┘
```

- Mission is NOT in a card — it floats on the gradient field like the hero section
- Credibility strip: YC logo + company logos in one row with subtle vertical dividers
- Founder cards use the testimonial card treatment (gradient bg, noise, gradient border)
- Entrance animations (text-reveal, card-reveal)

---

## Option C: Stacked Cinematic

Each section occupies near-full viewport height. Scroll through distinct scenes.
Heavy use of radial gradients positioning and scale.

```
┌──────────────────────────────────────────────┐
│                                              │
│               SCENE 1 (~80vh)                │
│                                              │
│  Mission statement, centered                 │
│  h2, generous whitespace above and below     │
│  Grid background covers full section         │
│  [Get In Touch →] centered below             │
│                                              │
├──────────────────────────────────────────────┤
│                                              │
│               SCENE 2 (~50vh)                │
│                                              │
│  Left: YC logo large + "Backed by YC"       │
│  Right: 4 company logos in 2×2 grid          │
│  Radial gradient anchored bottom-left        │
│                                              │
├──────────────────────────────────────────────┤
│                                              │
│               SCENE 3 (~70vh)                │
│                                              │
│  Two founder photos side by side, large      │
│  Names + titles below each                   │
│  Conic gradient glow behind photos           │
│  (like screenshot-frame-outer treatment)     │
│                                              │
└──────────────────────────────────────────────┘
```

- Each scene has its own background treatment (grid, gradient, glow)
- Sections separated by `<Divider>` like the homepage
- Photos get the screenshot-frame treatment (nested borders, conic gradient glow, shadow stack)
- Most dramatic and spacious option

---

## Option D: Two-Column Split

Left column is fixed content (mission + credibility). Right column is visual
(founder photos stacked). Creates a magazine-style reading flow.

```
┌────────────────────────┬─────────────────────┐
│                        │                     │
│  MISSION STATEMENT     │                     │
│  h3, grid background   │   [EB photo]        │
│                        │   large, rounded    │
│  [Get In Touch →]      │   conic glow        │
│                        │   name + title      │
│                        │                     │
│  ─────────────────     │                     │
│                        │   [PD photo]        │
│  [YC logo]             │   large, rounded    │
│  Backed by YC          │   conic glow        │
│                        │   name + title      │
│  NASA SpaceX LM FF    │                     │
│  (stacked or 2×2)      │                     │
│                        │                     │
└────────────────────────┴─────────────────────┘
```

- Uses the integrations section's two-column pattern (2fr 3fr grid)
- Left side: text-heavy with credibility stacked below mission
- Right side: visual-heavy with large founder photos
- Gradient-masked decorative border around entire grid
- Photos use the hero screenshot treatment (nested frame, glow, multi-shadow)

---

## Option E: Badge Row + Bento

Compact credibility in a single badge row at top, then a 2×2 bento grid below
with mission spanning top and founders on bottom.

```
┌──────────────────────────────────────────────┐
│  [YC]  NASA  SpaceX  Lockheed  Firefly       │
│  badge row, monochrome, subtle bg            │
├────────────────────────┬─────────────────────┤
│                        │                     │
│  MISSION STATEMENT     │  MISSION CONTINUED  │
│  (spans full width)    │  or key metric      │
│  grid bg, h3, CTA      │                     │
│                        │                     │
├────────────────────────┼─────────────────────┤
│                        │                     │
│  [EB photo]            │  [PD photo]         │
│  gradient card         │  gradient card      │
│  noise overlay         │  noise overlay      │
│  name + title          │  name + title       │
│                        │                     │
└────────────────────────┴─────────────────────┘
```

- Top row: unified credibility strip (YC + all company logos, no separation)
- Mission card spans full width
- Founder cards use testimonial-style gradient treatment
- Tightest layout — everything above the fold
- Decorative border with diagonal mask around entire bento

---

## Decision

Pick an option (or combine elements) and I'll implement it with the full visual
treatment: gradient fields, noise textures, shadow stacks, gradient-masked
borders, and entrance animations matching the rest of the site.
