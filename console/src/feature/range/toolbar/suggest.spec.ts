// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { suggest } from "@/feature/range/toolbar/suggest";
import { uniqueRangeName } from "@/platform/range/testutil";

const client = createTestClient();

const T0 = TimeStamp.now().sub(TimeSpan.hours(1));

/** Creates a range under `parent` that starts `minutes` after T0. */
const createAt = async (
  name: string,
  minutes: number,
  parent?: ranger.Key,
  c?: string,
): Promise<ranger.Range> =>
  await client.ranges.create({
    name,
    parent,
    color: c == null ? undefined : color.construct(c),
    timeRange: new TimeRange(
      T0.add(TimeSpan.minutes(minutes)),
      T0.add(TimeSpan.minutes(minutes + 1)),
    ),
  });

/** Creates a campaign with a run whose phases are Chill, Ignition, then Mainstage. */
const createCampaign = async () => {
  const campaign = await createAt(uniqueRangeName("campaign"), 0);
  const first = await createAt(uniqueRangeName("run"), 1, campaign.key);
  const chill = await createAt("Chill", 2, first.key, "#3774D0");
  const ignition = await createAt("Ignition", 3, first.key, "#DC4F2B");
  await createAt("Mainstage", 4, first.key, "#7A4FDB");
  const second = await createAt(uniqueRangeName("run"), 10, campaign.key);
  return { campaign, chill, ignition, second };
};

describe("suggest", () => {
  it("should start a run with the first phase of the run before it", async () => {
    const { chill, second } = await createCampaign();
    expect(await suggest(client, second.key)).toEqual({
      name: "Chill",
      color: chill.color,
    });
  });

  it("should follow a phase with the next phase of the run before it", async () => {
    const { ignition, second } = await createCampaign();
    const chill = await createAt("Chill", 11, second.key);
    expect(await suggest(client, second.key, chill.key)).toEqual({
      name: "Ignition",
      color: ignition.color,
    });
  });

  it("should add one to a trailing number and keep its width and color", async () => {
    const parent = await createAt(uniqueRangeName("parent"), 0);
    const last = await createAt("HF-09", 1, parent.key, "#3774D0");
    expect(await suggest(client, parent.key)).toEqual({
      name: "HF-10",
      color: last.color,
    });
  });

  it("should suggest no name for a last child with no trailing number", async () => {
    const parent = await createAt(uniqueRangeName("parent"), 0);
    await createAt("Hotfire", 1, parent.key);
    expect(await suggest(client, parent.key)).toEqual({ name: "" });
  });

  it("should suggest no name and log the error when the read fails", async () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(await suggest(client, uuid.create())).toEqual({ name: "" });
    expect(NotFoundError.matches(error.mock.calls[0][0])).toBe(true);
  });
});
