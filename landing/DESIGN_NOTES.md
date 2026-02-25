# Landing Page Design Notes

## Screenshot Frame Effects - Feedback

### Option 1: Perspective Tilt + Ambient Underlight
- Don't like the tilt effect at all
- **Verdict: No**

### Option 2: Noise-Textured Gradient Field / Mesh Gradients
- Liked the grainy multi-color gradient concept
- Very much a fan of mesh gradients — want to explore this direction further
- Preferred it covering the full page rather than scoped to the screenshot area
- Now lives as a fixed full-page background layer in Layout.astro
- Color palette needs work — current blue/amber is a starting point, not final
- **Verdict: Keep and iterate. Explore color palettes + mesh gradient techniques**

### Option 3: Scanning Gradient Border (Animated)
- Interesting concept, blue spotlight rotating around the screenshot perimeter
- Only ~30% of border lit at a time, 8s rotation
- Rating: 5/10 — has potential but not a standout on its own
- **Verdict: Maybe as a subtle accent combined with other effects**

### Option 4: Floating UI Fragments
- Skipped — not feasible to make look good with abstract shapes
- **Verdict: Skip**

### Option 5: Mouse-Tracked Spotlight
- Skipped per user preference
- **Verdict: Skip**

### Option 6: Chromatic Rim Light + Layered Elevation Shadows
- 4-layer box-shadow stack + blue rim light on top edge
- Rating: 3/10
- **Verdict: No**

### Option 7: Entrance Reveal Animation (Blur + Scale on Load)
- Screenshot starts invisible, blurred, scaled down, shifted — reveals over 1.2s
- cubic-bezier(0.16, 1, 0.3, 1) easing, 0.3s delay
- User reaction: "That's beautiful"
- **Verdict: Yes — keep this**

### Option 8: Bottom Fade Crop
- Screenshot fades to transparent over the bottom ~40%
- Initial reaction negative, then reconsidered — may work with tuning
- Current: mask from solid at 60% to transparent at 100%
- **Verdict: Maybe — keep as option, needs tuning**

### Option 9: Multi-Layer Shadow Stack
- 5-layer shadow from tight contact shadow to wide ambient
- User likes the concept, prefers subtle version over cranked-up
- **Verdict: Keep — subtle is right**

### Option 10: Dot-Grid Background + Radial Vignette
- Dot grid (0.75px dots, 20px spacing, gray-l8) extending 120px beyond screenshot
- Radial mask fading from center outward
- Rating: 6/10
- **Verdict: Maybe — decent but not a standout**

## General Preferences
- Subtle over garish — first pass is usually too strong, dial back
- Clean baseline: screenshot has outline with -1px offset (trims 1px from image edges)
- Each option should be shown in isolation, not stacked on previous ones
- Full-page effects preferred over scoped-to-screenshot effects

## Typography
- H1: 11rem, weight 500, line-height 120%, letter-spacing -0.02em
- Hero description: h4 level, rendered as `<p>`, color gray-l8, max-width 800px
- Nav links: h5 level, weight 400, color gray-l8

## Layout
- Nav height: 9rem, sticky, blur backdrop
- Logo: 4.5rem height
- Content max-width: 1400px
- Content padding: 7rem (4rem at 800px, 3rem at 600px)
- Hero top padding: 17.5rem
- Gap between hero text and screenshot: 13rem
