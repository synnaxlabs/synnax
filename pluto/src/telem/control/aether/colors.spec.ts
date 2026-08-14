// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type control, type query, type Synnax } from "@synnaxlabs/client";
import { type UnaryClient } from "@synnaxlabs/freighter";
import { color } from "@synnaxlabs/x";
import { assert, describe, expect, it, vi } from "vitest";

import { Colors } from "@/telem/control/aether/colors";
import { renderAether } from "@/testutil/renderAether";
import { theming } from "@/theming/aether";

type CacheConstructor = new (params: query.CacheParams) => query.Cache;
type ControlClientConstructor = new (cfg: {
  unary: UnaryClient;
  cache: query.Cache;
}) => Synnax["control"];

// The aether provider builds its own client from connection parameters, so the client
// is replaced at the module boundary. Its control table is the real one, detached from
// any Core, and the specs write to it directly. Type assertions below follow existing
// vi.mock patterns (vitest doesn't expose module types from importOriginal without
// import() annotations, which lint forbids).
vi.mock("@synnaxlabs/client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const { Cache } = actual.query as { Cache: CacheConstructor };
  const { Client } = actual.control as { Client: ControlClientConstructor };
  const unary: UnaryClient = {
    send: async () => {
      throw new Error("the control colors specs must not reach a Core");
    },
    use: () => {},
  };
  class MockSynnax {
    readonly control = new Client({ unary, cache: new Cache({ openStreamer: null }) });
    async close(): Promise<void> {}
  }
  return { ...actual, Synnax: MockSynnax };
});

const PARAMS = {
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
};

const THEME = theming.themeZ.parse(theming.SYNNAX_LIGHT);
const PALETTE = THEME.colors.visualization.palettes.default;

const held = (subject: string, resource: number): control.KeyedState => ({
  key: resource,
  resource,
  subject: { key: subject, name: subject, group: 0 },
  authority: 200,
});

const setup = () => {
  // Colors reads no telemetry, and the telem provider opens a frame feed the replaced
  // client does not serve.
  const h = renderAether(Colors, {
    state: {},
    synnax: { props: PARAMS },
    telem: false,
  });
  const provider = h.providers.synnax;
  assert(provider != null);
  const { client } = provider.internal;
  assert(client != null);
  return { colors: h.component, store: client.control.store };
};

describe("control/aether/Colors", () => {
  it("should assign the first palette color to the first subject", () => {
    const { colors, store } = setup();
    store.set([held("valve", 1)]);
    expect(color.equals(colors.get("valve"), PALETTE[0])).toBe(true);
  });

  it("should assign a different color to each subject", () => {
    const { colors, store } = setup();
    store.set([held("valve", 1), held("pump", 2)]);
    expect(color.equals(colors.get("valve"), colors.get("pump"))).toBe(false);
  });

  it("should keep a subject's color when an unrelated subject appears", () => {
    const { colors, store } = setup();
    store.set([held("valve", 1)]);
    const before = colors.get("valve");
    store.set([held("pump", 2)]);
    expect(color.equals(colors.get("valve"), before)).toBe(true);
  });

  it("should return a released subject's color to the pool", () => {
    const { colors, store } = setup();
    store.set([held("valve", 1), held("pump", 2)]);
    const released = colors.get("valve");
    store.delete(1);
    store.set([held("heater", 3)]);
    expect(color.equals(colors.get("heater"), released)).toBe(true);
  });

  it("should fall back to the first palette color once the palette runs out", () => {
    const { colors, store } = setup();
    store.set(PALETTE.map((_, i) => held(`subject_${i}`, i + 1)));
    store.set([held("overflow", PALETTE.length + 1)]);
    expect(color.equals(colors.get("overflow"), PALETTE[0])).toBe(true);
  });

  it("should give a subject holding nothing the default color", () => {
    const { colors } = setup();
    expect(color.equals(colors.get("nobody"), THEME.colors.gray.l9)).toBe(true);
  });

  it("should notify subscribers when the assignment changes", () => {
    const { colors, store } = setup();
    const handler = vi.fn();
    colors.onChange(handler);
    store.set([held("valve", 1)]);
    expect(handler).toHaveBeenCalled();
  });

  describe("setOverrides", () => {
    it("should prefer an override over the assigned color", () => {
      const { colors, store } = setup();
      store.set([held("valve", 1)]);
      const override = color.construct("#123456");
      colors.setOverrides({ valve: override });
      expect(color.equals(colors.get("valve"), override)).toBe(true);
    });

    it("should notify subscribers when the overrides change", () => {
      const { colors } = setup();
      const handler = vi.fn();
      colors.onChange(handler);
      colors.setOverrides({ valve: color.construct("#123456") });
      expect(handler).toHaveBeenCalledTimes(1);
    });

    it("should not notify subscribers when the overrides are unchanged", () => {
      const { colors } = setup();
      colors.setOverrides({ valve: color.construct("#123456") });
      const handler = vi.fn();
      colors.onChange(handler);
      colors.setOverrides({ valve: color.construct("#123456") });
      expect(handler).not.toHaveBeenCalled();
    });
  });
});
