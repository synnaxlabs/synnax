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
import {
  type AuthoritySource,
  authoritySource,
  Controller,
  controllerStateZ,
} from "@/telem/control/aether/controller";
import { renderAether } from "@/testutil/renderAether";

type CacheConstructor = new (params: query.CacheParams) => query.Cache;
type ControlClientConstructor = new (cfg: {
  unary: UnaryClient;
  cache: query.Cache;
}) => Synnax["control"];
type ErrorConstructor = new (message: string) => Error;

// The aether provider builds its own client from connection parameters, so the client
// is replaced at the module boundary. Its control table is the real one, detached from
// any Core, and the specs write to it directly. A read that misses the table raises
// not-found, which is what the Core answers for an uncontrolled channel. Type
// assertions below follow existing vi.mock patterns (vitest doesn't expose module types
// from importOriginal without import() annotations, which lint forbids).
vi.mock("@synnaxlabs/client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const { Cache } = actual.query as { Cache: CacheConstructor };
  const { Client } = actual.control as { Client: ControlClientConstructor };
  const { NotFoundError } = actual as { NotFoundError: ErrorConstructor };
  const unary: UnaryClient = {
    send: async () => {
      throw new NotFoundError("no control state");
    },
    use: () => {},
  };
  const unused = (): never => {
    throw new Error("the control status specs read no telemetry");
  };
  class MockSynnax {
    readonly control = new Client({ unary, cache: new Cache({ openStreamer: null }) });
    readonly channels = { retrieve: unused };
    openFeed = () => ({ read: unused, stream: unused, close: async () => {} });
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

const CONTROLLER_KEY = "controller-1";
const CHANNEL = 65537;

const held = (subject: string, resource: number): control.KeyedState => ({
  key: resource,
  resource,
  subject: { key: subject, name: subject, group: 0 },
  authority: 200,
});

const setup = (channel: number = CHANNEL) => {
  const h = renderAether(Colors, {
    state: {},
    synnax: { props: PARAMS },
    registry: { [Controller.TYPE]: Controller },
    children: {
      [CONTROLLER_KEY]: {
        type: Controller.TYPE,
        state: controllerStateZ.parse({ name: "Valve Controller" }),
      },
    },
  });
  const controller = h.child<Controller>(CONTROLLER_KEY);
  const { client } = controller;
  assert(client != null);
  const source = controller.create<AuthoritySource>(authoritySource({ channel }));
  assert(source != null);
  return { colors: h.component, controller, source, store: client.control.store };
};

describe("control/aether/AuthoritySource", () => {
  it("should report no channel when the source is bound to key zero", () => {
    const { source, controller } = setup(0);
    const status = source.value();
    expect(status.variant).toEqual("disabled");
    expect(status.message).toEqual("No Channel");
    expect(status.key).toEqual(controller.key);
    expect(status.details).toEqual({ valid: false, authority: 0 });
  });

  it("should report uncontrolled when no subject holds the channel", () => {
    const { source } = setup();
    const status = source.value();
    expect(status.variant).toEqual("disabled");
    expect(status.message).toEqual("Uncontrolled");
    expect(status.details).toEqual({ valid: true, color: undefined, authority: 0 });
  });

  it("should succeed when the controller itself holds the channel", () => {
    const { source, store, controller } = setup();
    store.set([held(controller.key, CHANNEL)]);
    const status = source.value();
    expect(status.variant).toEqual("success");
    expect(status.message).toEqual(`Controlled by ${controller.key}`);
  });

  it("should error when another subject holds the channel", () => {
    const { source, store } = setup();
    store.set([held("another_operator", CHANNEL)]);
    const status = source.value();
    expect(status.variant).toEqual("error");
    expect(status.message).toEqual("Controlled by another_operator");
    expect(status.key).toEqual("another_operator");
  });

  it("should carry the holder's authority and assigned color", () => {
    const { source, store, colors } = setup();
    store.set([held("another_operator", CHANNEL)]);
    const { details } = source.value();
    expect(details.authority).toEqual(200);
    assert(details.color != null);
    expect(color.equals(details.color, colors.get("another_operator"))).toBe(true);
  });

  it("should follow a channel from uncontrolled to held to released", () => {
    const { source, store } = setup();
    expect(source.value().message).toEqual("Uncontrolled");
    store.set([held("another_operator", CHANNEL)]);
    expect(source.value().message).toEqual("Controlled by another_operator");
    store.delete(CHANNEL);
    expect(source.value().message).toEqual("Uncontrolled");
  });

  it("should notify subscribers when control transfers", () => {
    const { source, store } = setup();
    const handler = vi.fn();
    source.onChange(handler);
    source.value();
    store.set([held("another_operator", CHANNEL)]);
    expect(handler).toHaveBeenCalled();
  });

  it("should stop notifying subscribers after cleanup", () => {
    const { source, store } = setup();
    // The first read is what subscribes the source to the channel's control state.
    source.value();
    const handler = vi.fn();
    source.onChange(handler);
    source.cleanup();
    store.set([held("another_operator", CHANNEL)]);
    expect(handler).not.toHaveBeenCalled();
  });
});
