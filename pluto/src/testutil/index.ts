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
// imports from "vitest" or exposes a Mock in its signature (e.g. dom.ts, render.ts) —
// those stay in-package and are imported directly by pluto's own specs.
export * from "@/testutil/Synnax";
