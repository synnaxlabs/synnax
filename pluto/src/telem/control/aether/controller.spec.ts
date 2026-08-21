// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer, ValidationError } from "@synnaxlabs/client";
import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { StreamClosed } from "@synnaxlabs/freighter";
import { color, DataType, id, TimeStamp } from "@synnaxlabs/x";
import { afterEach, assert, describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { type aether } from "@/aether/aether";
import { type status } from "@/status/aether";
import { Colors } from "@/telem/control/aether/colors";
import {
  type AuthoritySource,
  authoritySource,
  Controller,
  controllerStateZ,
  type OpenWriter,
  type SetChannelValue,
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
  vi.restoreAllMocks();
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
 * channel, and waits for the client's change stream. */
const setup = async (key: channel.Key, openWriter?: OpenWriter) => {
  const h = renderAether(Colors, {
    state: {},
    synnax: { props: TEST_CLIENT_PARAMS },
    registry:
      openWriter == null ? { [Controller.TYPE]: Controller } : registryWith(openWriter),
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
  // With the cache enabled, connect() resolves only once the change stream is
  // live, so no transfer published after this point is missed.
  await mounted.connect();
  if (key !== 0) await mounted.control.retrieve([key]);
  const source = controller.create<AuthoritySource>(authoritySource({ channel: key }));
  assert(source != null);
  // Merges, as the DOM Controller does, so the pushed state keeps the status the
  // controller reports back.
  const setState = (next: Partial<z.input<typeof controllerStateZ>>): void =>
    h.setChildState(CONTROLLER_KEY, Controller.TYPE, { ...controller.state, ...next });
  return { colors: h.component, controller, source, setState, handle: h };
};

/** Mounts a controller with no Core behind it. */
const setupDisconnected = () => {
  const h = renderAether(Colors, {
    state: {},
    synnax: { props: null },
    registry: { [Controller.TYPE]: Controller },
    children: {
      [CONTROLLER_KEY]: {
        type: Controller.TYPE,
        state: controllerStateZ.parse({ name: CONTROLLER_NAME }),
      },
    },
  });
  return { controller: h.child<Controller>(CONTROLLER_KEY), handle: h };
};

const statusesOf = (h: { providers: { status: status.Aggregator | null } }): string[] =>
  h.providers.status?.state.statuses.map(({ message }) => message) ?? [];

interface Intercept {
  openWriter: OpenWriter;
  openCount: () => number;
  writeCount: () => number;
}

/**
 * A writer factory that counts what the controller opens and writes, and fails the
 * writes `onWrite` picks. Each writer delegates to a real one, so everything the test
 * does not fail still reaches the Core.
 */
const interceptWrites = (
  onWrite: (attempt: number) => Error | null = () => null,
): Intercept => {
  let opens = 0;
  let writes = 0;
  const openWriter: OpenWriter = async (client, config) => {
    opens += 1;
    const w = await client.openWriter(config);
    const write = w.write.bind(w);
    return Object.assign(Object.create(w) as framer.Writer, {
      write: async (...args: Parameters<typeof write>) => {
        writes += 1;
        const err = onWrite(writes);
        if (err != null) throw err;
        return await write(...args);
      },
    });
  };
  return { openWriter, openCount: () => opens, writeCount: () => writes };
};

/**
 * A writer factory whose open blocks until `finishOpen` is called, counting the writes
 * that reach the Core. Lets a test change state while an acquisition is in flight.
 */
const gateWriter = () => {
  let finishOpen!: () => void;
  const gate = new Promise<void>((resolve) => (finishOpen = resolve));
  let writes = 0;
  const openWriter: OpenWriter = async (client, config) => {
    await gate;
    const w = await client.openWriter(config);
    const write = w.write.bind(w);
    return Object.assign(Object.create(w) as framer.Writer, {
      write: async (...args: Parameters<typeof write>) => {
        writes += 1;
        return await write(...args);
      },
    });
  };
  return { openWriter, finishOpen: () => finishOpen(), writeCount: () => writes };
};

/** Binds every Controller the harness mounts to `openWriter`. */
const registryWith = (openWriter: OpenWriter): aether.ComponentRegistry => ({
  [Controller.TYPE]: class extends Controller {
    constructor(props: aether.ComponentConstructorProps) {
      super(props, openWriter);
    }
  },
});

describe("control/aether/Controller", SUITE, () => {
  it("should warn and stay released when no Core is connected", async () => {
    const { controller, handle } = setupDisconnected();
    controller.acquire();
    await expect
      .poll(() => statusesOf(handle), POLL)
      .toContain(
        `Failed to acquire control on ${CONTROLLER_NAME}: no Core is connected.`,
      );
    expect(controller.state.status).toBeUndefined();
  });

  it("should warn when there are no channels to control", async () => {
    const { controller, handle } = await setup(0);
    controller.acquire();
    await expect
      .poll(() => statusesOf(handle), POLL)
      .toContain(
        `Cannot acquire control on ${CONTROLLER_NAME} because there are no channels to control.`,
      );
    expect(controller.state.status).toBeUndefined();
  });

  it("should open a single writer when acquire is called twice", async () => {
    const ch = await createIndexed();
    const { openWriter, openCount } = interceptWrites();
    const { controller, source } = await setup(ch.key, openWriter);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    expect(openCount()).toEqual(1);
  });

  it("should return the channel when control is released", async () => {
    const ch = await createIndexed();
    const { controller, source } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    controller.release();
    await expect.poll(() => controller.state.status, POLL).toEqual("released");
    await expect.poll(() => source.value().message, POLL).toEqual("Uncontrolled");
  });

  it("should release the control it holds when it is deleted", async () => {
    const ch = await createIndexed();
    const { controller, source, handle } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    handle.deleteChild(CONTROLLER_KEY);
    await expect.poll(() => source.value().message, POLL).toEqual("Uncontrolled");
  });

  it("should need control of the channel a sink writes and its index", async () => {
    const ch = await createIndexed();
    const { controller } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    await expect
      .poll(() => [...controller.state.needsControlOf].sort(), POLL)
      .toEqual([ch.key, ch.index].sort());
  });

  it("should count a channel once when two sinks write it", async () => {
    const ch = await createIndexed();
    const { controller } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.create(setChannelValue({ channel: ch.key }));
    await expect.poll(() => controller.state.needsControlOf.length, POLL).toEqual(2);
  });

  it("should stop needing a channel once its sink is cleaned up", async () => {
    const ch = await createIndexed();
    const { controller } = await setup(ch.key);
    const sink = controller.create<SetChannelValue>(
      setChannelValue({ channel: ch.key }),
    );
    assert(sink != null);
    await expect.poll(() => controller.state.needsControlOf.length, POLL).toEqual(2);
    sink.cleanup();
    await expect.poll(() => controller.state.needsControlOf, POLL).toEqual([]);
  });

  it("should reopen the writer and replay the write on a closed stream", async () => {
    const ch = await createIndexed();
    const { openWriter, openCount, writeCount } = interceptWrites((attempt) =>
      attempt === 1 ? new StreamClosed() : null,
    );
    const { controller, source } = await setup(ch.key, openWriter);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    await controller.set({ [ch.key]: [1] });
    expect(writeCount()).toEqual(2);
    expect(openCount()).toEqual(2);
  });

  it("should propagate a write error that is not retryable", async () => {
    const ch = await createIndexed();
    const { openWriter, openCount, writeCount } = interceptWrites(
      () => new ValidationError("bad frame"),
    );
    const { controller, source } = await setup(ch.key, openWriter);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    await expect(controller.set({ [ch.key]: [1] })).rejects.toThrow("bad frame");
    expect(writeCount()).toEqual(1);
    expect(openCount()).toEqual(1);
  });

  it("should give up the control it holds once it is disabled", async () => {
    const ch = await createIndexed();
    const { controller, source, setState } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    setState({ disabled: true });
    await expect.poll(() => controller.state.status, POLL).toEqual("released");
    await expect.poll(() => source.value().message, POLL).toEqual("Uncontrolled");
  });

  it("should take control when asked explicitly while disabled", async () => {
    const ch = await createIndexed();
    const { controller, source, setState } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    setState({ disabled: true });
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    expect(controller.state.status).toEqual("acquired");
  });

  it("should refuse to open a writer for a write while disabled", async () => {
    const ch = await createIndexed();
    const { openWriter, openCount } = interceptWrites();
    const { controller, setState } = await setup(ch.key, openWriter);
    controller.create(setChannelValue({ channel: ch.key }));
    await expect.poll(() => controller.state.needsControlOf, POLL).toContain(ch.key);
    setState({ disabled: true });
    await controller.set({ [ch.key]: [1] });
    expect(openCount()).toEqual(0);
    expect(controller.state.status).not.toEqual("acquired");
  });

  it("should stand down when a write's writer opens after it was disabled", async () => {
    const ch = await createIndexed();
    const { openWriter, finishOpen, writeCount } = gateWriter();
    const { controller, setState } = await setup(ch.key, openWriter);
    controller.create(setChannelValue({ channel: ch.key }));
    await expect.poll(() => controller.state.needsControlOf, POLL).toContain(ch.key);
    const write = controller.set({ [ch.key]: [1] });
    setState({ disabled: true });
    finishOpen();
    await write;
    expect(writeCount()).toEqual(0);
    expect(controller.state.status).toEqual("released");
  });

  it("should keep a writer the user asks for while a write is acquiring", async () => {
    const ch = await createIndexed();
    const { openWriter, finishOpen } = gateWriter();
    const { controller, setState } = await setup(ch.key, openWriter);
    controller.create(setChannelValue({ channel: ch.key }));
    await expect.poll(() => controller.state.needsControlOf, POLL).toContain(ch.key);
    const write = controller.set({ [ch.key]: [1] });
    setState({ disabled: true });
    controller.acquire();
    finishOpen();
    await write;
    await expect.poll(() => controller.state.status, POLL).toEqual("acquired");
  });

  it("should keep control while it is not disabled", async () => {
    const ch = await createIndexed();
    const { controller, source, setState } = await setup(ch.key);
    controller.create(setChannelValue({ channel: ch.key }));
    controller.acquire();
    await expect.poll(() => source.value().variant, POLL).toEqual("success");
    setState({ disabled: false });
    expect(controller.state.status).toEqual("acquired");
  });
});

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
