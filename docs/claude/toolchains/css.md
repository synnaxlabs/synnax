# CSS

Rules for the stylesheets in Lyra, Pluto, the Console, and the sites. Each rule names
the check that enforces it, or says that review enforces it.

## Files and class names

- A component imports its own stylesheet from the file next to it. A module entry never
  imports a stylesheet; the build rejects one.
- Class names are BEM with a package prefix: a block (`pluto-btn`), an element
  (`pluto-btn__icon`), and a modifier (`pluto-btn--outlined`, or a shared
  `pluto--small`). Lyra and Pluto use `pluto`, and the Console uses `console`. TSX
  builds them with `CSS.B`, `CSS.BE`, `CSS.BM`, `CSS.BEM`, and `CSS.M`.
- `selector-class-pattern` rejects a name that breaks the pattern. It cannot see a name
  that no code emits, such as `pluto-height-small` for `pluto--height-small`.
- Style classes, never IDs. `selector-max-id` rejects an ID selector.

## Cascade layers

- The Console declares `@layer lyra, vendor, pluto` (`console/vite.config.ts`) and
  leaves its own CSS unlayered. A later layer wins, and unlayered CSS beats every layer,
  whatever the specificity.
- `!important` reverses the layer order: an important declaration in `lyra` beats one in
  the Console.
- The sites put Lyra in a layer named `pluto` (`site/docs/astro.config.ts`).

## Precedence

- Never let load order decide between two rules. The dev server and the build load
  stylesheets in different orders, so a tie can resolve one way in dev and the other way
  in the build (SY-5103).
- Write a component's defaults in `:where()`, which adds no specificity, as
  `lyra/src/button/Button.css` does. Then any rule of the component outranks them.
- One rule owns each property of an element. To change a Lyra component from outside,
  set its inputs (for a button: `--pluto-bg`, `--pluto-hover-bg`), never its outputs, so
  its hover and press rules still apply.

## Tokens and values

- Use theme tokens (`var(--pluto-*)`) for colors, sizes, and type, not raw values.
- `lyra/src/theming/static/*.css` is generated. Run `pnpm generate-theme` in `lyra/`
  after a theme change.
- Put static styles in CSS, never in a `style={{...}}` object. Pass only a truly dynamic
  value inline, as a custom property.

## Lint

- `pnpm lint:<package>` runs ESLint and Stylelint, and `pnpm fix:<package>` fixes both.
  Stylelint reads `.css` files and the `<style>` blocks of `.astro` files.
- A disable comment states its reason, and Stylelint reports one that hides nothing:

  ```css
  /* stylelint-disable-next-line declaration-no-important -- Monaco styles inline */
  ```

- Hand-written vendor prefixes are allowed only for `user-select` and `backdrop-filter`,
  which Safari needs. Write the prefixed form first. When the standard form comes first,
  the build minifier keeps only the prefixed one (SY-4726).

## Verify

- Check a CSS fix in the built app (`pnpm build:console-vite`), not only on the dev
  server.
