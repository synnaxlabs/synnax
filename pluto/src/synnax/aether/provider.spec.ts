// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type SynnaxParams } from "@synnaxlabs/client";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";

const { mockClose, constructed } = vi.hoisted(() => ({
  mockClose: vi.fn(async () => undefined),
  constructed: [] as SynnaxParams[],
}));

vi.mock("@synnaxlabs/client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  class MockSynnax {
    close = mockClose;
    constructor(params: SynnaxParams) {
      constructed.push(params);
    }
  }
  return { ...actual, Synnax: MockSynnax };
});

const MockSender = { send: vi.fn() };
const NOOP = alamos.Instrumentation.NOOP;

const PARAMS: SynnaxParams = {
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
};

const makeAggregator = (key: string): status.Aggregator => {
  const agg = new status.Aggregator({
    path: [`${key}-status`],
    type: status.Aggregator.TYPE,
    sender: MockSender,
    instrumentation: NOOP,
    parent: null,
  });
  agg._updateState({
    path: [`${key}-status`],
    state: { statuses: [] },
    type: status.Aggregator.TYPE,
    create: shouldNotCreate,
  });
  return agg;
};

const makeProvider = (
  key: string,
  agg: status.Aggregator = makeAggregator(key),
): aether.Component => {
  const props: aether.ComponentConstructorProps = {
    path: [key],
    type: synnax.Provider.TYPE,
    sender: MockSender,
    instrumentation: NOOP,
    parent: agg,
  };
  return new synnax.Provider(props);
};

const shouldNotCreate = (): never => {
  throw new Error("should not create a child");
};

const update = (provider: aether.Component, key: string, props: SynnaxParams | null) =>
  provider._updateState({
    path: [key],
    state: { props },
    type: synnax.Provider.TYPE,
    create: shouldNotCreate,
  });

describe("synnax.aether.Provider", () => {
  beforeEach(() => {
    mockClose.mockClear();
  });

  it("closes the underlying client when the node is deleted", () => {
    const key = "synnax-provider";
    const provider = makeProvider(key);

    update(provider, key, PARAMS);
    expect(mockClose).not.toHaveBeenCalled();

    provider._delete([key]);
    expect(mockClose).toHaveBeenCalledTimes(1);
  });

  it("logs a close failure instead of throwing through the lifecycle", async () => {
    const key = "synnax-provider-fail";
    const provider = makeProvider(key);
    update(provider, key, PARAMS);
    mockClose.mockRejectedValueOnce(new Error("close boom"));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      provider._delete([key]);
      await vi.waitFor(() => expect(consoleError).toHaveBeenCalled());
      expect(consoleError.mock.calls[0][0]).toBe("failed to close client");
    } finally {
      consoleError.mockRestore();
    }
  });

  it("closes the previous client when the connection params swap", () => {
    const key = "synnax-provider-swap";
    const provider = makeProvider(key);

    update(provider, key, PARAMS);
    expect(mockClose).not.toHaveBeenCalled();

    update(provider, key, { ...PARAMS, host: "other-host" });
    expect(mockClose).toHaveBeenCalledTimes(1);
  });

  it("reports a client's internal error under the Core's name", () => {
    const key = "synnax-provider-internal";
    const agg = makeAggregator(key);
    const provider = makeProvider(key, agg);
    update(provider, key, { ...PARAMS, name: "Lab Core" });
    constructed.at(-1)?.onInternalError?.(new Error("stream broke"));
    expect(agg.state.statuses).toHaveLength(1);
    const [stat] = agg.state.statuses;
    expect(stat.name).toBe("Lab Core");
    expect(stat.message).toBe("Failed to sync with Core");
    expect(stat.description).toBe("stream broke");
  });

  it("closes the client when props swap to null", () => {
    const key = "synnax-provider-null";
    const provider = makeProvider(key);

    update(provider, key, PARAMS);
    expect(mockClose).not.toHaveBeenCalled();

    update(provider, key, null);
    expect(mockClose).toHaveBeenCalledTimes(1);
  });
});
