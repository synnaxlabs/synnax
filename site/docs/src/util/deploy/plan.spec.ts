// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Lookups, plan } from "@/util/deploy/plan";

const NEWEST: Record<string, string> = { "0.61": "0.61.0", "0.60": "0.60.1" };

const lookups = (trains: string[], branches: string[] = []): Lookups => ({
  trains: async () => trains,
  newest: async (train) => NEWEST[train],
  branched: async (train) => branches.includes(train),
});

describe("plan", () => {
  it("should build next from main and each train from its newest tag", async () => {
    expect(
      await plan(["next", "latest", "previous"], {}, lookups(["0.61", "0.60"])),
    ).toEqual([
      { version: "next", ref: "main", host: "next.docs.synnaxlabs.com" },
      { version: "0.61", ref: "core/v0.61.0", host: "docs.synnaxlabs.com" },
      { version: "0.60", ref: "core/v0.60.1", host: "v0-60.docs.synnaxlabs.com" },
    ]);
  });

  it("should build a train from its release branch", async () => {
    expect(await plan(["0.60"], {}, lookups(["0.61", "0.60"], ["0.60"]))).toEqual([
      {
        version: "0.60",
        ref: "release/core-0.60",
        host: "v0-60.docs.synnaxlabs.com",
      },
    ]);
  });

  it("should deploy a version once", async () => {
    const deployments = await plan(["latest", "0.61"], {}, lookups(["0.61"]));
    expect(deployments.map(({ version }) => version)).toEqual(["0.61"]);
  });

  it("should add no previous train while only one exists", async () => {
    const deployments = await plan(["latest", "previous"], {}, lookups(["0.61"]));
    expect(deployments.map(({ version }) => version)).toEqual(["0.61"]);
  });

  it("should add a released train that has a stable release", async () => {
    const deployments = await plan(
      ["next", "latest"],
      { released: "0.60" },
      lookups(["0.61", "0.60"]),
    );
    expect(deployments.map(({ version }) => version)).toEqual(["next", "0.61", "0.60"]);
  });

  it("should skip a released train with no stable Core release", async () => {
    const deployments = await plan(
      ["next", "latest"],
      { released: "0.62" },
      lookups(["0.61", "0.60"]),
    );
    expect(deployments.map(({ version }) => version)).toEqual(["next", "0.61"]);
  });

  it("should build one version from a given ref", async () => {
    expect(await plan(["0.60"], { ref: "abc123" }, lookups(["0.61", "0.60"]))).toEqual([
      { version: "0.60", ref: "abc123", host: "v0-60.docs.synnaxlabs.com" },
    ]);
  });

  it("should reject a ref with more than one version", async () => {
    await expect(
      plan(["next", "latest"], { ref: "abc123" }, lookups(["0.61"])),
    ).rejects.toThrow("A ref builds exactly one version, not 2");
  });

  it("should reject a train with no stable release", async () => {
    await expect(plan(["0.62"], {}, lookups(["0.61"]))).rejects.toThrow(
      "Train 0.62 has no stable Core release",
    );
  });

  it("should reject a plan when GitHub lists no train", async () => {
    await expect(plan(["latest"], {}, lookups([]))).rejects.toThrow(
      "GitHub lists no stable Core release",
    );
  });
});
