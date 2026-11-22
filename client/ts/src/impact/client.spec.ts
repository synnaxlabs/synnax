// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { NotFoundError, PlanChangedError } from "@/errors";
import { group } from "@/group";
import { impact } from "@/impact";
import { lineplot } from "@/lineplot";
import { resource } from "@/resource";
import { task } from "@/task";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: id.create(),
    leaseholder: 1,
    virtual: true,
    dataType: DataType.FLOAT32,
  });

const createPlot = async (ch: channel.Key): Promise<lineplot.LinePlot> => {
  const proj = await client.projects.create({ name: id.create() });
  return await client.lineplots.create({
    project: proj.key,
    name: id.create(),
    channels: { y1: [ch] },
  });
};

const createReadTask = async (ch: channel.Key): Promise<task.Task> => {
  const rack = await client.racks.create({ name: id.create() });
  const device = await client.devices.create({
    key: id.create(),
    rack: rack.key,
    location: "127.0.0.1:502",
    name: "plc",
    make: "modbus",
    model: "modbus",
    properties: {},
  });
  return await rack.createTask({
    name: id.create(),
    type: "modbus_read",
    config: {
      device: device.key,
      channels: [{ type: "coil", key: "coil", channel: ch }],
    },
  });
};

/**
 * Reports whether retrieve finds nothing. The client's cache learns of a delete through
 * its change stream, so a caller polls it.
 */
const isGone = async (retrieve: () => Promise<unknown>): Promise<boolean> =>
  await retrieve().then(
    () => false,
    (e: unknown) => NotFoundError.matches(e),
  );

const deleteOf = (...keys: channel.Key[]): impact.Change => ({
  type: "delete_channels",
  keys,
});

describe("impact", () => {
  it("should plan a delete and apply it with the plan's token", async () => {
    const ch = await createChannel();
    const plot = await createPlot(ch.key);
    const plan = await client.impact.plan(deleteOf(ch.key));
    expect(plan.entries).toEqual([
      { record: lineplot.resourceID(plot.key), effect: "detach", choices: [] },
    ]);
    expect(plan.hidden).toEqual(0);
    const applied = await client.impact.apply(deleteOf(ch.key), { token: plan.token });
    expect(applied.entries).toEqual(plan.entries);
    await expect
      .poll(
        async () => await isGone(async () => await client.channels.retrieve(ch.key)),
      )
      .toBe(true);
  });

  it("should apply a delete with no token and return its change", async () => {
    const ch = await createChannel();
    await createPlot(ch.key);
    const applied = await client.impact.apply(deleteOf(ch.key));
    expect(applied.change).toEqual(deleteOf(ch.key));
    expect(applied.entries).toEqual([]);
    await expect
      .poll(
        async () => await isGone(async () => await client.channels.retrieve(ch.key)),
      )
      .toBe(true);
  });

  it("should refuse a stale token with the plan as it is now", async () => {
    const ch = await createChannel();
    const stale = await client.impact.plan(deleteOf(ch.key));
    const plot = await createPlot(ch.key);
    const err = await client.impact
      .apply(deleteOf(ch.key), { token: stale.token })
      .catch((e: unknown) => e);
    expect(err).toSatisfy(PlanChangedError.matches);
    const now = impact.planOf(err as PlanChangedError);
    expect(now.entries).toEqual([
      { record: lineplot.resourceID(plot.key), effect: "detach", choices: [] },
    ]);
    expect(now.token).not.toEqual(stale.token);
    expect((await client.channels.retrieve(ch.key)).key).toEqual(ch.key);
  });

  it("should delete a task that a pick names with the channel", async () => {
    const ch = await createChannel();
    const tsk = await createReadTask(ch.key);
    const record = resource.idToString(task.resourceID(tsk.key));
    const plan = await client.impact.plan(deleteOf(ch.key), {
      picks: { [record]: "delete" },
    });
    await client.impact.apply(deleteOf(ch.key), {
      token: plan.token,
      picks: { [record]: "delete" },
    });
    await expect
      .poll(
        async () =>
          await isGone(async () => await client.tasks.retrieve({ key: tsk.key })),
      )
      .toBe(true);
    await expect
      .poll(
        async () => await isGone(async () => await client.channels.retrieve(ch.key)),
      )
      .toBe(true);
  });

  it("should plan a group delete and apply it with the plan's token", async () => {
    const g = await client.groups.create({ name: id.create(), scope: "channel" });
    const change: impact.Change = { type: "delete_groups", keys: [g.key] };
    const plan = await client.impact.plan(change);
    expect(plan).toEqual({
      entries: [],
      kept: 0,
      hidden: 0,
      token: plan.token,
      change,
    });
    expect(plan.token).toHaveLength(16);
    expect(await client.impact.apply(change, { token: plan.token })).toEqual(plan);
    await expect
      .poll(
        async () =>
          await isGone(async () => await client.groups.retrieve({ key: g.key })),
      )
      .toBe(true);
  });

  it("should plan an ungroup and apply it, moving channels to the parent", async () => {
    const parent = await client.groups.create({ name: id.create(), scope: "channel" });
    const g = await client.groups.create({
      name: id.create(),
      parent: group.resourceID(parent.key),
    });
    const ch = await client.channels.create({
      name: id.create(),
      leaseholder: 1,
      virtual: true,
      dataType: DataType.FLOAT32,
      group: g.key,
    });
    const change: impact.Change = { type: "ungroup_groups", keys: [g.key] };
    const plan = await client.impact.plan(change);
    expect(plan).toEqual({
      entries: [],
      kept: 0,
      hidden: 0,
      token: plan.token,
      change,
    });
    expect(await client.impact.apply(change, { token: plan.token })).toEqual(plan);
    await expect
      .poll(async () => (await client.channels.retrieve(ch.key)).group)
      .toEqual(parent.key);
    await expect
      .poll(
        async () =>
          await isGone(async () => await client.groups.retrieve({ key: g.key })),
      )
      .toBe(true);
  });

  it("should count a record that the caller cannot see as hidden", async () => {
    const ch = await createChannel();
    await createReadTask(ch.key);
    const userClient = await createTestClientWithPolicy(client, {
      name: "impact",
      objects: [channel.resourceID(ch.key)],
      actions: ["retrieve", "delete"],
    });
    const plan = await userClient.impact.plan(deleteOf(ch.key));
    expect(plan.entries).toEqual([]);
    expect(plan.hidden).toEqual(1);
  });
});
