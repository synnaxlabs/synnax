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

const NEWEST: Record<string, string> = {
  "0.62": "0.62.0",
  "0.61": "0.61.1",
  "0.60": "0.60.1",
};

// Each branch maps to the tags it holds.
const lookups = (
  trains: string[],
  branches: Record<string, string[]> = {},
): Lookups => ({
  trains: async () => trains,
  newest: async (train) => NEWEST[train],
  holds: async (branch, tag) => branches[branch]?.includes(tag) ?? false,
});

describe("plan", () => {
  it("should build next from main and each train from its newest tag", async () => {
    expect(
      await plan(["next", "latest", "previous"], {}, lookups(["0.62", "0.61"])),
    ).toEqual([
      { version: "next", ref: "main", host: "next.docs.synnaxlabs.com" },
      { version: "0.62", ref: "core/v0.62.0", host: "docs.synnaxlabs.com" },
      { version: "0.61", ref: "core/v0.61.1", host: "v0-61.docs.synnaxlabs.com" },
    ]);
  });

  it("should build a train from a branch that holds its newest tag", async () => {
    const branches = { "release/0.61": ["core/v0.61.0", "core/v0.61.1"] };
    expect(await plan(["0.61"], {}, lookups(["0.62", "0.61"], branches))).toEqual([
      {
        version: "0.61",
        ref: "release/0.61",
        host: "v0-61.docs.synnaxlabs.com",
      },
    ]);
  });

  it("should build a train from its newest tag when the branch lacks it", async () => {
    const branches = { "release/0.61": ["core/v0.61.0"] };
    const [{ ref }] = await plan(["0.61"], {}, lookups(["0.62", "0.61"], branches));
    expect(ref).toEqual("core/v0.61.1");
  });

  it("should build 0.60 from its docs branch", async () => {
    const branches = { "release/0.60": ["core/v0.60.1"] };
    const [{ ref }] = await plan(["0.60"], {}, lookups(["0.61", "0.60"], branches));
    expect(ref).toEqual("release/docs-0.60");
  });

  it("should deploy a version once", async () => {
    const deployments = await plan(["latest", "0.61"], {}, lookups(["0.61"]));
    expect(deployments.map(({ version }) => version)).toEqual(["0.61"]);
  });

  it("should add no previous train while only one exists", async () => {
    const deployments = await plan(["latest", "previous"], {}, lookups(["0.61"]));
    expect(deployments.map(({ version }) => version)).toEqual(["0.61"]);
  });

  it("should add a released train that has a site", async () => {
    const deployments = await plan(
      ["next", "latest"],
      { released: "0.60" },
      lookups(["0.61", "0.60"]),
    );
    expect(deployments.map(({ version }) => version)).toEqual(["next", "0.61", "0.60"]);
  });

  it("should skip a released train with no site", async () => {
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

  it("should reject a train with no site", async () => {
    await expect(plan(["0.62"], {}, lookups(["0.61"]))).rejects.toThrow(
      "Train 0.62 has no docs site",
    );
  });

  it("should reject a plan when no train has a site", async () => {
    await expect(plan(["latest"], {}, lookups([]))).rejects.toThrow(
      "No train has a docs site",
    );
  });
});
