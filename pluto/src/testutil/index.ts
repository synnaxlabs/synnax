// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This barrel is the published `@synnaxlabs/pluto/testutil` entry. It must stay
// vitest-free: every export here ships in the bundle graph, and `vi` is a runtime
// import whose types do not resolve portably in consumers. Never add a helper that
// imports from "vitest" or exposes a Mock in its signature (e.g. dom.ts) — those stay
// in-package and are imported directly by pluto's own specs. A helper that genuinely
// needs vitest belongs in a dedicated `./testutil/vitest` subpath with vitest
// externalized; no such helper exists yet, so that subpath does not either.
export * from "@/aether/test";
export * from "@/testutil/render";
export * from "@/testutil/Synnax";
