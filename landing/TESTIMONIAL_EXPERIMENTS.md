# Testimonial Card Style Experiments

## #1 — Gradient border glow
**Verdict: Yes** — "oh yeah, sexy"
- Replaces solid border with gradient using `::after` + mask-composite trick
- Lightened brand color top-left → transparent middle → darkened brand color bottom-right
- 1px border via padding on the pseudo-element

## #2 — Inner radial spotlight
**Verdict: Yes (subtle)** — "looks nice but should be a bit more subtle"
- Radial gradient at 15% 15% (top-left), 8% lighter than brand color, fading at 45%
- Layered on top of the existing linear gradient

## #3 — Brand-to-black diagonal fade
**Verdict: No** — "no that looks bad"

## #4 — Frosted glass overlay band
**Verdict: No** — "fucking horrible holy shit"

## #5 — Animated gradient shimmer
**Verdict: No** — "horrifying"

## #6 — Double border with gap
**Verdict:** Pending (retry with outline approach)

## #7 — Top-edge highlight line
**Verdict:** Pending

## #8 — Depth layering with offset shadow card
**Verdict:** Pending

## #9 — Vignette darkening
**Verdict:** Pending

## #10 — Iridescent border shift
**Verdict:** Pending
