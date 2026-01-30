# Documentation Site Visual Style Guide

This document outlines the current state of article styling and our plan for improving
the visual style from A- to A+.

## Design Philosophy

The goal is not to redesign, but to refine. We're looking for consistency, restraint,
and attention to the subtle details that separate good documentation from elegant
documentation.

---

## Current State Audit

### Unit System

The site uses Pluto's 6px base unit (`1rem = 6px`). All values below are in this system
unless noted.

---

## 1. Typography

### Font Family

| Element  | Current                      | Assessment |
| -------- | ---------------------------- | ---------- |
| Headings | "Inter Variable", sans-serif | ✅ Good    |
| Body     | "Inter Variable", sans-serif | ✅ Good    |
| Code     | "Geist Mono", monospace      | ✅ Good    |
| UI       | "Inter Variable", sans-serif | ✅ Good    |

**Notes:** Inter and Geist Mono are excellent modern choices. No changes needed.

### Font Size Scale

| Element     | Current            | Pixels | Assessment                   |
| ----------- | ------------------ | ------ | ---------------------------- |
| h1          | 8rem               | 48px   | ✅ Good                      |
| h2          | 5.5rem             | 33px   | ✅ Good                      |
| h3          | 4rem               | 24px   | ✅ Good                      |
| h4          | 3rem               | 18px   | ✅ Good                      |
| h5          | 2.666rem           | 16px   | ✅ Good                      |
| Body (p)    | 2.666rem           | 16px   | ✅ Good                      |
| small       | 2.333rem           | 14px   | ✅ Good                      |
| tiny        | 2.166rem           | 13px   | ✅ Good                      |
| Code inline | var(--pluto-small) | 14px   | ⚠️ Review - may be too small |
| Code block  | inherits           | 16px   | ✅ Good                      |

**Mobile (≤800px):**

| Element | Current  | Pixels |
| ------- | -------- | ------ |
| h1      | 5rem     | 30px   |
| h2      | 4rem     | 24px   |
| h3      | 3.25rem  | 19.5px |
| h4      | 2.75rem  | 16.5px |
| h5      | 2.5rem   | 15px   |
| p       | 2.333rem | 14px   |

### Font Weight Distribution

| Element             | Current | Assessment                             |
| ------------------- | ------- | -------------------------------------- |
| h1                  | 450     | ⚠️ Light for a page title              |
| h2                  | 500     | ✅ Good                                |
| h3                  | 500     | ✅ Good                                |
| h4                  | 550     | ✅ Good                                |
| h5                  | 450     | ✅ Good                                |
| Body (p)            | 350     | ⚠️ Very light - may affect readability |
| Body (p) mobile     | 375     | Bumped for mobile, still light         |
| small               | 350     | ⚠️ Very light                          |
| strong              | 550     | ✅ Good                                |
| Article description | 350     | Intentionally light, OK                |
| Breadcrumb          | 375     | ✅ Good                                |
| Breadcrumb current  | 450     | ✅ Good                                |

**Concern:** Body weight at 350 is unusually light. Most docs use 400. This may cause
readability issues on some displays or for some users. Worth testing.

### Line Height

| Element  | Current | Assessment                                    |
| -------- | ------- | --------------------------------------------- |
| Headings | 150%    | ⚠️ Too loose for headings - should be tighter |
| Body (p) | 175%    | ✅ Excellent for readability                  |
| small    | 150%    | ✅ Good                                       |
| Code     | inherit | ✅ Good                                       |

**Issue:** Headings with 150% line-height feel loose. Large headings (h1, h2) typically
benefit from tighter line-height (120-130%) to feel cohesive, especially for multi-line
headings.

### Letter Spacing

| Element      | Current | Assessment                                    |
| ------------ | ------- | --------------------------------------------- |
| All elements | 0       | ⚠️ Large headings could use slight tightening |

**Missing:** No letter-spacing adjustments. Large headings (h1, h2) often benefit from
slight negative tracking (-0.01em to -0.02em) for a more premium feel.

### Text Color

| Element            | Current                 | Assessment |
| ------------------ | ----------------------- | ---------- |
| Primary text (p)   | var(--pluto-gray-l10)   | ✅ Good    |
| Secondary (desc)   | var(--pluto-gray-l9)    | ✅ Good    |
| Muted (breadcrumb) | var(--pluto-gray-l8)    | ✅ Good    |
| Links              | var(--pluto-primary-p1) | ✅ Good    |
| Code               | var(--pluto-text-color) | ✅ Good    |

### Content Width

| Property        | Current   | Assessment                  |
| --------------- | --------- | --------------------------- |
| Article basis   | 840px     | ✅ Good - ~75 chars at 16px |
| Max width       | 1340px    | ✅ Good                     |
| Article padding | 5rem 8rem | ✅ Good                     |

---

## 2. Spacing

### Header Margins

| Element       | Space Above | Space Below | Assessment                          |
| ------------- | ----------- | ----------- | ----------------------------------- |
| h1, h2, h3... | 5rem (30px) | 2rem (12px) | ⚠️ Same above all - lacks hierarchy |
| h1, h3        | 6rem (36px) | 2rem (12px) | Slight variation, but inconsistent  |

**Issue:** All headings have the same top margin (5rem). This doesn't create visual
hierarchy between major sections (h2) and subsections (h3, h4). Consider:

- h2: 8rem above (major section break)
- h3: 5rem above (subsection)
- h4: 4rem above (minor section)

### Paragraph Spacing

| Element            | Current      | Assessment                     |
| ------------------ | ------------ | ------------------------------ |
| Between paragraphs | 1.5rem (9px) | ⚠️ Tight - consider increasing |

**Note:** 9px between paragraphs is quite tight. Most docs use 16-24px (1em-1.5em of
body text). This might be contributing to a slightly cramped feel.

### List Spacing

| Property            | Current      | Assessment    |
| ------------------- | ------------ | ------------- |
| List padding-left   | 5rem (30px)  | ✅ Good       |
| List padding mobile | 2rem (12px)  | ✅ Good       |
| List item margin    | 0.5rem (3px) | ⚠️ Very tight |

**Issue:** 3px between list items is very tight. Consider 8-12px for better scanability.

### Code Block Spacing

| Property      | Current                | Assessment               |
| ------------- | ---------------------- | ------------------------ |
| Margin        | 5rem 1.5rem (30px 9px) | ⚠️ Asymmetric horizontal |
| Padding       | 3rem (18px)            | ✅ Good                  |
| Border radius | 1rem (6px)             | ✅ Good                  |

**Note:** The 1.5rem horizontal margin indents code blocks slightly from body text. This
is a stylistic choice - some prefer flush alignment. Worth reviewing.

### Table Spacing

| Property     | Current                    | Assessment |
| ------------ | -------------------------- | ---------- |
| Cell padding | 1.25rem 1.5rem (7.5px 9px) | ⚠️ Tight   |
| th padding   | 1.5rem (9px)               | ⚠️ Tight   |

**Issue:** Table cell padding feels cramped. Consider 12-16px for more breathing room.

### Image/Media Spacing

| Property      | Current                         | Assessment |
| ------------- | ------------------------------- | ---------- |
| Margin        | 5rem 0 (30px)                   | ✅ Good    |
| Border radius | 1rem (6px)                      | ✅ Good    |
| Box shadow    | 0 0 1rem 0 var(--pluto-gray-l4) | ✅ Good    |

### Divider Spacing

| Property | Current | Assessment               |
| -------- | ------- | ------------------------ |
| Margin   | 5% 0    | ⚠️ Percentage is unusual |

---

## 3. Code Blocks

### Block Styling

| Property   | Current                   | Assessment |
| ---------- | ------------------------- | ---------- |
| Background | var(--pluto-gray-l1)      | ✅ Good    |
| Border     | var(--pluto-border) (1px) | ✅ Good    |
| Radius     | 1rem (6px)                | ✅ Good    |
| Padding    | 3rem (18px)               | ✅ Good    |

### Syntax Highlighting

| Token     | Current                   | Assessment |
| --------- | ------------------------- | ---------- |
| Text      | var(--pluto-text-color)   | ✅ Good    |
| Comments  | var(--pluto-gray-l7)      | ✅ Good    |
| Strings   | var(--pluto-secondary-m1) | ✅ Good    |
| Functions | #556bf8                   | ✅ Good    |
| Keywords  | #cc255f                   | ✅ Good    |

### Copy Button

| State    | Current                     | Assessment |
| -------- | --------------------------- | ---------- |
| Default  | opacity: 0                  | ✅ Good    |
| Hover    | opacity: 1, 0.2s transition | ✅ Good    |
| Position | top-right, 2.3rem offset    | ✅ Good    |

### Inline Code

| Property   | Current                       | Assessment                |
| ---------- | ----------------------------- | ------------------------- |
| Background | var(--pluto-gray-l2)          | ✅ Good                   |
| Border     | 1px var(--pluto-gray-l3)      | ✅ Good                   |
| Padding    | 0.25rem 0.75rem (1.5px 4.5px) | ⚠️ Small                  |
| Radius     | 0.5rem (3px)                  | ✅ Good                   |
| Font size  | var(--pluto-small-size)       | ⚠️ Small relative to body |

**Issue:** Inline code at 14px when body is 16px creates a noticeable size difference.
Consider keeping inline code closer to body size (15-16px).

---

## 4. Links

| State   | Current                               | Assessment                |
| ------- | ------------------------------------- | ------------------------- |
| Default | var(--pluto-primary-p1), no underline | ✅ Good                   |
| Hover   | underline                             | ✅ Good                   |
| Visited | (none)                                | ✅ Good (modern approach) |

**Consideration:** Some prefer underlines always visible for accessibility. Current
approach is clean but relies on color alone to indicate links.

### Heading Anchors

| Property   | Current                   | Assessment |
| ---------- | ------------------------- | ---------- |
| Visibility | opacity: 0, show on hover | ✅ Good    |
| Color      | var(--pluto-gray-l7)      | ✅ Good    |
| Position   | margin-left: 2rem         | ✅ Good    |
| Transition | 0.2s                      | ✅ Good    |

---

## 5. Tables

| Property       | Current                           | Assessment |
| -------------- | --------------------------------- | ---------- |
| Border style   | Full grid (horizontal + vertical) | ⚠️ Heavy   |
| Header weight  | 600                               | ✅ Good    |
| Cell alignment | start (left)                      | ✅ Good    |
| Row striping   | None                              | ✅ Good    |
| Cell font size | var(--pluto-small-size)           | ⚠️ Small   |

**Issues:**

1. Full grid borders can feel heavy. Consider horizontal-only borders (like Stripe).
2. Table cell text at 14px when body is 16px creates hierarchy but may be too small for
   data-heavy tables.

---

## 6. Callouts / Details

| Property   | Current                   | Assessment |
| ---------- | ------------------------- | ---------- |
| Background | var(--pluto-gray-l1)      | ✅ Good    |
| Border     | var(--pluto-border)       | ✅ Good    |
| Radius     | 1rem (6px)                | ✅ Good    |
| Summary bg | var(--pluto-gray-l2)      | ✅ Good    |
| Padding    | 1.666rem 2rem (10px 12px) | ✅ Good    |
| Animation  | max-height 0.3s           | ✅ Good    |

**Note:** The details/collapse pattern is well-implemented. No major issues.

### Note Component

| Property    | Current                 | Assessment |
| ----------- | ----------------------- | ---------- |
| Padding     | 2rem 3rem (12px 18px)   | ✅ Good    |
| Font size   | var(--pluto-small-size) | ⚠️ Small   |
| Font weight | 450                     | ✅ Good    |

---

## 7. Images & Figures

| Property      | Current                       | Assessment |
| ------------- | ----------------------------- | ---------- |
| Width         | 100%                          | ✅ Good    |
| Border        | var(--pluto-border)           | ✅ Good    |
| Radius        | 1rem (6px)                    | ✅ Good    |
| Shadow        | 0 0 1rem var(--pluto-gray-l4) | ✅ Good    |
| Caption style | italic, centered, gray-l8     | ✅ Good    |

---

## 8. Breadcrumbs

| Property     | Current                | Assessment |
| ------------ | ---------------------- | ---------- |
| Separator    | Caret icon (>)         | ✅ Good    |
| Text color   | var(--pluto-gray-l8)   | ✅ Good    |
| Current page | gray-l10, weight 450   | ✅ Good    |
| Hover        | gray-l10, no underline | ✅ Good    |

---

## Summary: Strengths

1. **Font choices** - Inter Variable and Geist Mono are excellent
2. **Color system** - Well-defined grayscale hierarchy
3. **Code blocks** - Clean styling, good syntax colors, nice copy button behavior
4. **Images** - Good treatment with border, shadow, radius
5. **Responsive** - Good mobile breakpoint adjustments
6. **Details/collapse** - Well-implemented with smooth animation
7. **Breadcrumbs** - Clean and functional
8. **Content width** - Good max-width for readability

---

## Summary: Areas for Improvement

### High Priority

1. **Heading line-height** (150% is too loose)
   - Current: 150% for all headings
   - Target: 120-130% for h1/h2, 135-140% for h3/h4

2. **Body text weight** (350 is unusually light)
   - Current: 350
   - Target: 400 (standard) or 375 (if intentionally light)

3. **Paragraph spacing** (too tight)
   - Current: 1.5rem (9px)
   - Target: 2.5-3rem (15-18px)

4. **List item spacing** (too tight)
   - Current: 0.5rem (3px)
   - Target: 1.5-2rem (9-12px)

5. **Heading hierarchy** (all same top margin)
   - Current: 5rem for all
   - Target: h2: 8rem, h3: 5rem, h4: 4rem

### Medium Priority

6. **Table borders** (full grid is heavy)
   - Consider: horizontal-only borders

7. **Table cell padding** (cramped)
   - Current: 1.25rem 1.5rem
   - Target: 2rem 2.5rem

8. **Inline code size** (smaller than body)
   - Current: --pluto-small-size (14px)
   - Target: 0.9em of body (14.4px) or same as body

9. **Letter-spacing on large headings**
   - Current: 0
   - Target: -0.01em to -0.02em on h1/h2

### Low Priority

10. **Code block horizontal margin** (slight indent)
    - Current: 1.5rem horizontal
    - Consider: 0 for flush alignment

11. **Divider margin** (percentage-based)
    - Current: 5% 0
    - Consider: fixed rem values for consistency

---

## How to Audit in Browser

To verify these values and see computed styles:

1. Open the docs site in Chrome/Firefox
2. Right-click any element → "Inspect"
3. In DevTools, look at the "Computed" tab (not "Styles")
4. Key properties to check:
   - `font-size` (in px)
   - `line-height` (in px)
   - `font-weight`
   - `margin-top`, `margin-bottom`
   - `padding`
   - `letter-spacing`

**Specific elements to inspect:**

- Paragraph: any `<p>` in article body
- Headings: `<h2>`, `<h3>` with class `pluto-text`
- Inline code: `<code>` not inside `.astro-code`
- Code block: `.astro-code`
- Table cell: `<td>` and `<th>`
- List item: `<li>`

---

## Action Items

### Phase 1: Quick Wins (Typography)

- [ ] Tighten heading line-height (150% → 125-135%)
- [ ] Add negative letter-spacing to h1/h2
- [ ] Increase body weight (350 → 400)
- [ ] Increase paragraph margin (1.5rem → 2.5rem)
- [ ] Increase list item margin (0.5rem → 1.5rem)

### Phase 2: Spacing Hierarchy

- [ ] Differentiate heading top margins (h2 > h3 > h4)
- [ ] Increase table cell padding
- [ ] Review code block horizontal alignment

### Phase 3: Polish

- [ ] Consider horizontal-only table borders
- [ ] Review inline code sizing
- [ ] Audit all spacing for consistency with scale

---

## Notes

_Space for ongoing observations and decisions._
