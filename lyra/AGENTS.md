TypeScript development rules for this package: @../docs/claude/toolchains/typescript.md

CSS rules: @../docs/claude/toolchains/css.md

Lyra is subpath-only (`@synnaxlabs/lyra/<module>`) and Synnax-blind: no Aether, no
`@synnaxlabs/client`, no Pluto. After adding or removing a `src/<module>/index.ts`, run
`pnpm exports` here; the build fails on a stale exports map.
