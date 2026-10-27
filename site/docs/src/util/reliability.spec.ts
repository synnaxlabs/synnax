// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

const suite = (job: string, product: string, lang: string) => ({
  job,
  product,
  lang,
  kind: "unit",
  tags: ["unit"],
  os: "ubuntu",
  conclusion: "success",
  url: `https://github.com/synnaxlabs/synnax/actions/runs/1/job/${job}`,
  count: 1,
});

const DATA = {
  run: {
    id: 1,
    url: "https://github.com/synnaxlabs/synnax/actions/runs/1",
    sha: "abc",
    version: "0.60.0",
    branch: "main",
    started_at: "2026-10-27T00:00:00Z",
    ended_at: "2026-10-27T01:00:00Z",
    jobs: 3,
    manual_passes: [],
    manual_note: null,
  },
  products: [
    { key: "core", name: "Core" },
    { key: "toolchain", name: "Toolchain" },
  ],
  suites: [
    suite("gorp", "core", "go"),
    suite("x-go", "core", "go"),
    suite("eslint-config", "toolchain", "ts"),
  ],
  tests: [
    [0, "gorp/table_test.go › creates", "passed", null, null, "gorp/table_test.go#L1"],
    [1, "x/go/set_test.go › adds", "passed", null, null, "x/go/set_test.go#L1"],
    [
      2,
      "configs/eslint/rules/a.spec.ts › reports",
      "passed",
      null,
      null,
      "configs/eslint/rules/a.spec.ts#L1",
    ],
  ],
};

vi.stubGlobal("fetch", async () => new Response(JSON.stringify(DATA)));
const { products } = await import("@/util/reliability");

const subsystems = (key: string) => products.find((p) => p.key === key)?.subsystems;

describe("reliability", () => {
  it("should count Gorp tests with the shared libraries under X", () => {
    expect(subsystems("core")).toEqual({ x: { name: "X", count: 2 } });
  });

  it("should count the lint rule tests under Toolchain", () => {
    expect(subsystems("toolchain")).toEqual({
      eslint: { name: "Lint rules", count: 1 },
    });
  });
});
