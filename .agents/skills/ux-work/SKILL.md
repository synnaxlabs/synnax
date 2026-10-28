---
name: ux-work
description:
  Rules and process for UI and UX work on the Console, Lyra and Pluto components, the
  Portal, and the docs site. Use whenever a change affects how something looks or
  behaves (CSS, layout, spacing, color, type, icons, motion, modals, forms, menus,
  keycaps, overflow, interaction flows) and whenever you propose a visual design. Loads
  `docs/claude/ux.md`, then enforces the process: start from the named problem, scope
  every selector, build in the real app, judge renders in both themes, and show one
  proposal against today.
---

# UX work

Read `docs/claude/ux.md` before you change or propose anything. It holds the rules and
the prior art. This skill holds the process.

## Process

1. **Start from the named problem.** Say the problem in one sentence. Keep every value
   that does not cause it, even when a reference product uses another value.
2. **Fix it in one place.** Prefer a fix in the Lyra component or token that all callers
   share over an override at each call site. Use rem and the existing tokens.
3. **List what each selector matches.** A rule on `.pluto-btn` also styles inputs,
   switches, list items, menu items, tabs, and select triggers. Before you show a CSS
   change, list every component that its selectors match. Never put a global change (a
   token value, font smoothing) in the same change as a scoped one.
4. **Build it in the real app.** Change a running dev server, never a mock page:
   `pnpm dev:console-vite` serves the Console to a browser, and `pnpm dev:portal` and
   `pnpm dev:docs` serve the sites.
5. **Judge the renders.** Capture light and dark with headless Playwright
   (`page.emulateMedia({ colorScheme })`). The Console follows that setting only while
   its theme is System. Read values with `getComputedStyle`; never guess them. Critique
   each capture as a designer: hierarchy, shared edges, spacing, clutter. Fix and
   capture again until it holds up.
6. **Show one proposal against today.** For a visual choice, show a few real options
   next to each other on one page, with today first. Never show a grid of variants built
   from reference numbers.
7. **Write down what locks.** When the founder accepts a value or a rule, add it to
   `docs/claude/ux.md` in the same PR.
8. **Update the tests that read the UI.** A changed placeholder, label, role name, or
   class can break Console specs and the helpers in `integration/console/`. Search both
   for the old text, and update them in the same PR.
