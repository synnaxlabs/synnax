// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

const suite = (
  job: string,
  product: string,
  lang: string,
  { kind = "unit", tags = ["unit"] }: { kind?: string; tags?: string[] } = {},
) => ({
  job,
  product,
  lang,
  kind,
  tags,
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
  hardware: {
    id: 2,
    url: "https://github.com/synnaxlabs/synnax/actions/runs/2",
    sha: "def",
    started_at: "2026-10-27T02:00:00Z",
    ended_at: "2026-10-27T02:05:00Z",
  },
  products: [
    { key: "core", name: "Core" },
    { key: "driver", name: "Driver" },
    { key: "toolchain", name: "Toolchain" },
  ],
  suites: [
    suite("gorp", "core", "go"),
    suite("x-go", "core", "go"),
    suite("eslint-config", "toolchain", "ts"),
    suite("stylelint-config", "toolchain", "ts"),
    suite("integration-driver", "driver", "py", { kind: "system", tags: ["e2e"] }),
    suite("hitl", "driver", "py", { kind: "system", tags: ["e2e", "hardware"] }),
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
    [
      3,
      "configs/stylelint/rules/b.spec.ts › reports",
      "passed",
      null,
      null,
      "configs/stylelint/rules/b.spec.ts#L1",
    ],
    [4, "driver/ni › NIRead", "passed", null, 1, "integration/tests/driver/ni.py#L1"],
    [5, "hitl/ni_cdaq › CDAQRead", "failed", "timeout", 1, "integration/a.py#L1"],
    [5, "hitl/crio_ni › CRIORead", "passed", null, 1, "integration/b.py#L1"],
  ],
};

vi.stubGlobal("fetch", async () => new Response(JSON.stringify(DATA)));
const { hardware, hardwareTotals, products } = await import("@/util/reliability");

const subsystems = (key: string) => products.find((p) => p.key === key)?.subsystems;

describe("reliability", () => {
  it("should count Gorp tests with the shared libraries under X", () => {
    expect(subsystems("core")).toEqual({ x: { name: "X", count: 2 } });
  });

  it("should count the ESLint and Stylelint rule tests under Toolchain", () => {
    expect(subsystems("toolchain")).toEqual({
      lint: { name: "Lint rules", count: 2 },
    });
  });

  it("should count hardware tests apart from other end-to-end tests", () => {
    expect(subsystems("driver")).toEqual({
      e2e: { name: "End-to-end", count: 1 },
      hardware: { name: "Real hardware", count: 2 },
    });
  });

  it("should count the states of the hardware tests", () => {
    expect(hardwareTotals).toMatchObject({ passed: 1, failed: 1 });
    expect(hardware?.id).toEqual(2);
  });
});
