// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer } from "@synnaxlabs/client";
import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { color, DataType, id, TimeStamp } from "@synnaxlabs/x";
import { afterEach, assert, describe, expect, it, vi } from "vitest";

import { Colors } from "@/telem/control/aether/colors";
import {
  type AuthoritySource,
  authoritySource,
  Controller,
  controllerStateZ,
  setChannelValue,
} from "@/telem/control/aether/controller";
import { renderAether } from "@/testutil/renderAether";

const client = createTestClient();

const CONTROLLER_KEY = "controller-1";
const CONTROLLER_NAME = "Valve Controller";
const AUTHORITY = 200;
const POLL = { timeout: 5000 };
// Each test creates channels, connects a client, and takes control, so the default
// per-test budget is too small.
const SUITE = { timeout: 30_000 };

const openWriters: framer.Writer[] = [];

afterEach(async () => {
  await Promise.all(openWriters.splice(0).map(async (w) => await w.close()));
});

const createVirtual = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: `control_${id.create()}`,
    dataType: DataType.FLOAT64,
    virtual: true,
  });

/** Creates an index and a data channel, which is what a controller writes through. */
const createIndexed = async (): Promise<channel.Channel> => {
  const index = await client.channels.create({
    name: `control_index_${id.create()}`,
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  return await client.channels.create({
    name: `control_data_${id.create()}`,
    dataType: DataType.FLOAT32,
    index: index.key,
  });
};

interface Holder {
  subject: string;
  release: () => Promise<void>;
}

/** Takes control of the channel under the given subject. Any writer left open at the
 * end of the test is closed. */
const hold = async (key: channel.Key, subject: string): Promise<Holder> => {
  const w = await client.openWriter({
    start: TimeStamp.now(),
    channels: [key],
    controlSubject: { key: subject, name: subject },
    authorities: AUTHORITY,
  });
  openWriters.push(w);
  return {
    subject,
    release: async () => {
      openWriters.splice(openWriters.indexOf(w), 1);
      await w.close();
    },
  };
};

/** Mounts a controller against the test cluster with a status source bound to the
 * channel. Reading the channel opens the control stream, so no transfer after this
 * point is missed. */
const setup = async (key: channel.Key) => {
  const h = renderAether(Colors, {
    state: {},
    synnax: { props: TEST_CLIENT_PARAMS },
    registry: { [Controller.TYPE]: Controller },
    children: {
      [CONTROLLER_KEY]: {
        type: Controller.TYPE,
        state: controllerStateZ.parse({ name: CONTROLLER_NAME }),
      },
    },
  });
  const controller = h.child<Controller>(CONTROLLER_KEY);
  const { client: mounted } = controller;
  assert(mounted != null);
  if (key !== 0) await mounted.control.retrieve([key]);
  const source = controller.create<AuthoritySource>(authoritySource({ channel: key }));
  assert(source != null);
  return { colors: h.component, controller, source };
};

describe("control/aether/AuthoritySource", SUITE, () => {
  it("should report no channel when the source is bound to key zero", async () => {
    const { source, controller } = await setup(0);
    const status = source.value();
    expect(status.variant).toEqual("disabled");
    expect(status.message).toEqual("No channel");
    expect(status.key).toEqual(controller.key);
    expect(status.details).toEqual({ valid: false, authority: 0 });
  });

  it("should report uncontrolled when no subject holds the channel", async () => {
    const ch = await createVirtual();
    const { source } = await setup(ch.key);
    const status = source.value();
    expect(status.variant).toEqual("disabled");
    expect(status.message).toEqual("Uncontrolled");
    expect(status.details).toEqual({ valid: true, color: undefined, authority: 0 });
  });

  it("should succeed when the controller itself holds the channel", async () => {
    const ch = await createIndexed();
    const { source, controller } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    expect(source.value().message).toEqual(`Controlled by ${CONTROLLER_NAME}`);
    expect(source.value().key).toEqual(controller.key);
  });

  it("should error when another subject holds the channel", async () => {
    const ch = await createVirtual();
    const { source } = await setup(ch.key);
    const { subject } = await hold(ch.key, "another_operator");
    await expect.poll(() => source.value().variant, POLL).toEqual("error");
    expect(source.value().message).toEqual(`Controlled by ${subject}`);
    expect(source.value().key).toEqual(subject);
  });

  it("should carry the holder's authority and assigned color", async () => {
    const ch = await createVirtual();
    const { source, colors } = await setup(ch.key);
    const { subject } = await hold(ch.key, "another_operator");
    await expect.poll(() => source.value().variant, POLL).toEqual("error");
    const { details } = source.value();
    expect(details.authority).toEqual(AUTHORITY);
    assert(details.color != null);
    expect(color.equals(details.color, colors.get(subject))).toBe(true);
  });

  it("should follow a channel from uncontrolled to held to released", async () => {
    const ch = await createVirtual();
    const { source } = await setup(ch.key);
    expect(source.value().message).toEqual("Uncontrolled");
    const holder = await hold(ch.key, "another_operator");
    await expect
      .poll(() => source.value().message, POLL)
      .toEqual(`Controlled by ${holder.subject}`);
    await holder.release();
    await expect.poll(() => source.value().message, POLL).toEqual("Uncontrolled");
  });

  it("should notify subscribers when control transfers", async () => {
    const ch = await createVirtual();
    const { source } = await setup(ch.key);
    const handler = vi.fn();
    source.onChange(handler);
    source.value();
    await hold(ch.key, "another_operator");
    await expect.poll(() => handler.mock.calls.length > 0, POLL).toBe(true);
  });

  it("should stop notifying subscribers after cleanup", async () => {
    const ch = await createVirtual();
    const { source, colors } = await setup(ch.key);
    // The first read is what subscribes the source to the channel's control state.
    source.value();
    const handler = vi.fn();
    source.onChange(handler);
    source.cleanup();
    // Colors sees every transfer this client observes, so it witnesses the acquire the
    // cleaned-up source must not report.
    const witness = vi.fn();
    colors.onChange(witness);
    await hold(ch.key, "another_operator");
    await expect.poll(() => witness.mock.calls.length > 0, POLL).toBe(true);
    expect(handler).not.toHaveBeenCalled();
  });
});
