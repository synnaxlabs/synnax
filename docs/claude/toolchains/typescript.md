# TypeScript Development

## TypeScript/JavaScript Packages

The monorepo uses **PNPM workspaces** with **catalog dependencies** for centralized
version management:

- **Alamos** (`/alamos/ts/`) - Distributed instrumentation and observability with
  OpenTelemetry integration
- **Client** (`/client/ts/`) - TypeScript client library for Synnax server API
- **Console** (`/console/`) - Main Tauri desktop application
- **Drift** (`/drift/`) - Redux state synchronization for multi-window Tauri apps
- **Freighter** (`/freighter/ts/`) - Protocol-agnostic transport layer (HTTP, WebSocket,
  gRPC)
- **Media** (`/x/media/`) - Media utilities and helpers
- **Pluto** (`/pluto/`) - High-performance React visualization component library
- **X** (`/x/ts/`) - Shared TypeScript utilities and helpers

All packages use:

- **Vite** for bundling with dual CJS/ESM output
- **TypeScript 5.9+** with strict mode
- **Vitest** for testing
- **ESLint 9** with flat config format
- **Turbo** for build orchestration and caching

## Build Commands

- `pnpm build` - Build all packages using Turbo
- `pnpm build:alamos` - Build Alamos instrumentation library
- `pnpm build:client` - Build only the client libraries
- `pnpm build:console` - Build only the Console application
- `pnpm build:drift` - Build Drift state synchronization library
- `pnpm build:freighter` - Build Freighter transport layer
- `pnpm build:media` - Build Media utilities
- `pnpm build:pluto` - Build only the Pluto component library
- `pnpm build:x` - Build X utility library
- `pnpm check-types` - Type check all TypeScript packages
- `pnpm check-types:console` - Type check only Console

## Development & Testing

- `pnpm dev:console` - Start Console in development mode (Tauri)
- `pnpm dev:console-vite` - Start Console Vite dev server only
- `pnpm dev:pluto` - Start Pluto development server
- `pnpm test` - Run all tests across packages
- `pnpm test:console` - Run Console tests
- `pnpm test:pluto` - Run Pluto tests
- `pnpm watch` - Watch mode for all packages

## Code Quality

- `pnpm lint` - Lint all packages with ESLint
- `pnpm fix` - Auto-fix linting issues across packages
- `pnpm lint:console` - Lint only Console package
- `pnpm fix:console` - Fix linting issues in Console

## Code Style

- **Formatter**: Prettier (configured in `.prettierrc`)
  - 88 character line length
  - Configured plugins for XML, Astro, shell scripts, TOML
- **Linter**: ESLint 9 with flat config (`eslint.config.ts`)
  - React plugin with strict JSX rules
  - TypeScript ESLint with type-checked rules
  - Simple import sort plugin for automatic import ordering
  - Consistent type imports: `import { type Foo } from "bar"`
- **Import style**: Prefer absolute imports over relative (`@/components` not
  `../../../components`)
- **React patterns**: Function components, hooks, no prop-types (use TypeScript)
- **Exports**: Dual CJS/ESM via Vite build

### Key ESLint Rules

- `@typescript-eslint/consistent-type-imports` - Inline type imports
- `simple-import-sort/imports` - Automatic import sorting
- `simple-import-sort/exports` - Automatic export sorting
- `react/react-in-jsx-scope` - Off (not needed in React 17+)
- `react/jsx-curly-brace-presence` - Never use braces for strings
- `react/jsx-filename-extension` - Only `.jsx` and `.tsx` files
- `react/jsx-boolean-value` - Enforce consistent boolean attribute style
- `react/jsx-no-constructed-context-values` - Prevent re-renders
- `@typescript-eslint/no-floating-promises` - Require await/void for promises
- `@typescript-eslint/no-unused-vars` - Allow underscore-prefixed vars

## Testing with Vitest

### Structure

- Test files use `*.spec.ts` extension
- Tests co-located with source code
- BDD-style with `describe`/`it` blocks

### Example

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

describe("Feature Name", () => {
  describe("Sub-feature", () => {
    beforeEach(() => {
      // Setup
    });

    it("should do something specific", () => {
      expect(result).toEqual(expected);
    });
  });
});
```

### Key Features

- **Mocking:** Uses `vi.fn()` for function mocks
- **Assertions:** Standard expect API (`toEqual`, `toBe`, `toHaveBeenCalledTimes`)
- **Setup/Teardown:** `beforeEach`, `beforeAll`, `afterAll`
- **Async testing:** Native async/await support
- **React Testing:** Uses `@testing-library/react` for component tests
- **Hooks testing:** `@testing-library/react` for custom hooks

## Package Structure

Each package follows this structure:

```
package-name/
├── src/
│   ├── index.ts              # Main entry point
│   ├── feature/
│   │   ├── feature.ts
│   │   └── feature.spec.ts   # Tests co-located
│   └── ...
├── dist/                     # Vite build output
│   ├── index.js              # ESM
│   ├── index.cjs             # CommonJS
│   └── src/                  # Type declarations
├── package.json
├── tsconfig.json
├── tsconfig.vite.json
├── vite.config.ts
├── eslint.config.ts
└── README.md
```

### package.json Pattern

```json
{
  "name": "@synnaxlabs/package-name",
  "type": "module",
  "exports": {
    ".": {
      "types": "./dist/src/index.d.ts",
      "import": "./dist/index.js",
      "require": "./dist/index.cjs"
    }
  },
  "scripts": {
    "build": "tsc --noEmit && vite build",
    "check-types": "tsc --noEmit",
    "test": "vitest",
    "lint": "eslint",
    "fix": "eslint --fix",
    "madge": "madge --circular src"
  }
}
```

## Circular Dependency Detection

Use `madge` to detect circular dependencies:

```bash
pnpm madge         # Check all packages
pnpm madge:console # Check only Console
```

## Common Gotchas

- **Console**: Has both Tauri (`dev:console`) and Vite-only (`dev:console-vite`)
  development modes
- **PNPM catalog**: Shared dependency versions in `pnpm-workspace.yaml`; use `catalog:`
  prefix in package.json
- **Turbo**: Build cache can cause issues; clear with `npx turbo clean`
- **Absolute imports**: TypeScript paths configured in tsconfig.json, ensure bundler
  respects them
- **Vitest**: Always import from "vitest", not Jest
- **React 19**: Using latest React 19.1.1 - check for breaking changes from React 18

## Development Best Practices

- **Always prefer absolute imports** over relative imports in TypeScript projects
- **Vitest for testing** - always use Vitest APIs, not Jest
- **Dependency injection & composition** - prefer composition over singletons and
  inheritance
- **Type everything** - use TypeScript strict mode, avoid `any`
- **Test co-location** - keep tests next to source files for better discoverability
