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
import { renderAether } from "@/testutil/renderAether";
import { theming } from "@/theming/aether";

const client = createTestClient();

const THEME = theming.themeZ.parse(theming.SYNNAX_LIGHT);
const PALETTE = THEME.colors.visualization.palettes.default;
const DEFAULT_COLOR = THEME.colors.gray.l9;
const POLL = { timeout: 5000 };
// Each test creates channels, connects a client, and takes control, so the default
// per-test budget is too small.
const SUITE = { timeout: 30_000 };

const openWriters: framer.Writer[] = [];

afterEach(async () => {
  await Promise.all(openWriters.splice(0).map(async (w) => await w.close()));
});

const createChannels = async (count: number): Promise<channel.Channel[]> =>
  await client.channels.create(
    Array.from({ length: count }, () => ({
      name: `control_${id.create()}`,
      dataType: DataType.FLOAT64,
      virtual: true,
    })),
  );

interface Holder {
  subject: string;
  release: () => Promise<void>;
}

/** Takes control of the channel under a fresh subject. Any writer left open at the end
 * of the test is closed. */
const hold = async (ch: channel.Channel): Promise<Holder> => {
  const subject = id.create();
  const w = await client.openWriter({
    start: TimeStamp.now(),
    channels: [ch.key],
    controlSubject: { key: subject, name: subject },
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

/** Mounts Colors against the test cluster. Reading the given channels opens the control
 * stream, so no transfer after this point is missed. */
const setup = async (channels: channel.Channel[]): Promise<Colors> => {
  const h = renderAether(Colors, {
    state: {},
    synnax: { props: TEST_CLIENT_PARAMS },
    // Colors reads no telemetry, and the telem provider opens a frame feed this spec
    // does not use.
    telem: false,
  });
  const provider = h.providers.synnax;
  assert(provider != null);
  const { client: mounted } = provider.internal;
  assert(mounted != null);
  if (channels.length > 0)
    await mounted.control.retrieve(channels.map(({ key }) => key));
  return h.component;
};

/** Waits until the subject holds a color, and returns it. */
const assigned = async (colors: Colors, subject: string): Promise<color.Color> => {
  await expect
    .poll(() => color.equals(colors.get(subject), DEFAULT_COLOR), POLL)
    .toBe(false);
  return colors.get(subject);
};

describe("control/aether/Colors", SUITE, () => {
  it("should assign a palette color to a subject holding control", async () => {
    const [ch] = await createChannels(1);
    const colors = await setup([ch]);
    const { subject } = await hold(ch);
    const assignedColor = await assigned(colors, subject);
    expect(PALETTE.some((c) => color.equals(c, assignedColor))).toBe(true);
  });

  it("should assign a different color to each subject", async () => {
    const channels = await createChannels(2);
    const colors = await setup(channels);
    const [first, second] = await Promise.all(channels.map(hold));
    const firstColor = await assigned(colors, first.subject);
    const secondColor = await assigned(colors, second.subject);
    expect(color.equals(firstColor, secondColor)).toBe(false);
  });

  it("should keep a subject's color when an unrelated subject appears", async () => {
    const [first, second] = await createChannels(2);
    const colors = await setup([first, second]);
    const { subject } = await hold(first);
    const before = await assigned(colors, subject);
    const other = await hold(second);
    await assigned(colors, other.subject);
    expect(color.equals(colors.get(subject), before)).toBe(true);
  });

  it("should give a subject holding nothing the default color", async () => {
    const colors = await setup([]);
    expect(color.equals(colors.get("nobody"), DEFAULT_COLOR)).toBe(true);
  });

  it("should return a released subject to the default color", async () => {
    const [ch] = await createChannels(1);
    const colors = await setup([ch]);
    const { subject, release } = await hold(ch);
    await assigned(colors, subject);
    await release();
    await expect
      .poll(() => color.equals(colors.get(subject), DEFAULT_COLOR), POLL)
      .toBe(true);
  });

  it("should not hand a released color to a subject that still holds one", async () => {
    const [first, second, third] = await createChannels(3);
    const colors = await setup([first, second, third]);
    const released = await hold(first);
    const kept = await hold(second);
    await assigned(colors, released.subject);
    const keptColor = await assigned(colors, kept.subject);
    await released.release();
    const next = await hold(third);
    const nextColor = await assigned(colors, next.subject);
    expect(color.equals(nextColor, keptColor)).toBe(false);
  });

  it("should share colors once the palette runs out", async () => {
    const channels = await createChannels(PALETTE.length + 1);
    const colors = await setup(channels);
    const holders = await Promise.all(channels.map(hold));
    // One poll over the whole fan-in: the transfers arrive in any order, so a
    // per-subject window would starve whichever lands last.
    await expect
      .poll(
        () =>
          holders.every(
            ({ subject }) => !color.equals(colors.get(subject), DEFAULT_COLOR),
          ),
        { timeout: 20_000 },
      )
      .toBe(true);
    const distinct = new Set(
      holders.map(({ subject }) => color.hex(colors.get(subject))),
    );
    expect(distinct.size).toBeLessThan(holders.length);
  });

  it("should notify subscribers when the assignment changes", async () => {
    const [ch] = await createChannels(1);
    const colors = await setup([ch]);
    const handler = vi.fn();
    colors.onChange(handler);
    await hold(ch);
    await expect.poll(() => handler.mock.calls.length > 0, POLL).toBe(true);
  });

  describe("setOverrides", SUITE, () => {
    it("should prefer an override over the assigned color", async () => {
      const [ch] = await createChannels(1);
      const colors = await setup([ch]);
      const { subject } = await hold(ch);
      await assigned(colors, subject);
      const override = color.construct("#123456");
      colors.setOverrides({ [subject]: override });
      expect(color.equals(colors.get(subject), override)).toBe(true);
    });

    it("should notify subscribers when the overrides change", async () => {
      const colors = await setup([]);
      const handler = vi.fn();
      colors.onChange(handler);
      colors.setOverrides({ valve: color.construct("#123456") });
      expect(handler).toHaveBeenCalledTimes(1);
    });

    it("should not notify subscribers when the overrides are unchanged", async () => {
      const colors = await setup([]);
      colors.setOverrides({ valve: color.construct("#123456") });
      const handler = vi.fn();
      colors.onChange(handler);
      colors.setOverrides({ valve: color.construct("#123456") });
      expect(handler).not.toHaveBeenCalled();
    });
  });
});
