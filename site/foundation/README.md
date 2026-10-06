# Foundation site

The landing page for `https://foundation.synnaxlabs.com`, built with Astro, React, Lyra,
and the shared Synnax site infrastructure.

From the repository root:

```sh
pnpm install --frozen-lockfile
pnpm dev:foundation
pnpm build:foundation
```

The package reuses Lyra components and theme tokens, Synnax media assets, and
`@synnaxlabs/site-common` for the mobile drawer, footer, favicons, robots, and sitemap.
The contact form uses the existing Synnax Formspree endpoint. No runtime environment
variables or application server are required.

## Loading strategy

- The hero, navigation, section copy, and nine unlock cards are static HTML/SVG. Inter
  and Geist Mono Latin WOFF2 fonts are preloaded. Other font subsets load only when the
  browser needs their characters; fonts are never embedded in CSS.
- System, engineering, and deployment islands hydrate within 600 pixels of the viewport.
  Their expensive geometry is omitted from the initial HTML and fetched in separate
  chunks after hydration. The containers reserve their dimensions.
- Each deployment has its own chunk. The carousel loads the selected scene, warms tabs
  on hover/focus, and uses idle time to prefetch neighboring scenes only while the
  carousel is visible. Automatic prefetch respects data saver and slow networks.
- Failed scene requests can be retried. Rapid changes cannot display a stale scene. Only
  one deployment scene is mounted at a time.
- Hero/system animation pauses offscreen and in hidden tabs. Carousel packet motion
  pauses offscreen. All diagrams respect reduced-motion preferences.
- The contact form hydrates when visible and retains a native POST fallback.
- Vercel serves static output at the edge. Hashed assets use immutable caching.

The October 2026 optimization reduced generated HTML from 574,938 to approximately
161,000 bytes (118 KB to 41 KB with gzip). These are artifact sizes, not browser
loading-time benchmarks. The React runtime and Lyra controls are shared across islands
and load when an interactive section approaches the viewport.

## Validation

```sh
pnpm check-types:foundation
pnpm lint:foundation
pnpm stylelint:foundation
pnpm knip:foundation
pnpm --filter @synnaxlabs/foundation test --run
pnpm exec prettier --check site/foundation
pnpm build:foundation
```

Tests cover menu/focus behavior, contact validation and submission failures, diagram
state changes, carousel keyboard navigation, deferred loading and retries, and animation
visibility. CI uses the repository's shared TypeScript workflow.

## Hosting and previews

The Vercel project is `synnax/foundation`, linked to `synnaxlabs/synnax`. Its root is
`site/foundation`, with workspace files outside the root included. The package's
`vercel.json` installs the frozen pnpm workspace and runs `pnpm build:foundation` from
the repository root. The Node version follows the repository's Node 24 runtime.

From the repository root, an authenticated Vercel CLI can create a preview:

```sh
vercel link --project foundation --scope synnax
vercel deploy --scope synnax
```

Pull requests on the connected repository can also receive Vercel previews. Production
promotion and assigning `foundation.synnaxlabs.com` are separate deployment actions; the
canonical URL and sitemap already use that domain. Preview access follows the project's
Vercel protection settings.

## Diagram structure

`IndustrialGeometry.tsx` supplies projection and geometric primitives;
`IndustrialNodes.tsx` supplies shared Foundation nodes and site pads. System and
field-deployment scenes reuse these primitives. Engineering scenes illustrate
reliability, data transport, and deployment behavior with a separate projection.

The deployment order is process plants, aerospace, quantum labs, energy storage, and
marine fleets. All scenes are illustrative architecture, with no customer telemetry or
live hardware connection. Product copy presents Foundation as available.
