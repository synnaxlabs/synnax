// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MultiSeries } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";

const { mockUse } = vi.hoisted(() => ({ mockUse: vi.fn() }));

// Type assertions below follow existing vi.mock patterns (vitest doesn't expose module
// types from importOriginal without import() annotations, which lint forbids).
vi.mock("@/synnax/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { synnax: { ...(actual.synnax as object), use: mockUse } };
});

const ALAMOS_KEY = "alamos";
const STATUS_KEY = "status";
const TELEM_KEY = "telem";

interface StubCore {
  openFeed: Mock;
  channels: telem.Client["channels"];
}

const stubCore = (): StubCore => ({
  openFeed: vi.fn(() => ({
    read: async () => new MultiSeries([]),
    stream: () => telemTest.mockSubscription(() => {}),
    close: async () => {},
  })),
  channels: {
    retrieve: async () => {
      throw new Error("unused");
    },
  },
});

interface MountedProvider {
  /** Re-runs the provider's lifecycle. */
  update: () => void;
  /** The status aggregator the provider reports failures to. */
  aggregator: status.Aggregator;
}

/** Mounts the provider under the alamos and status providers it reads context from,
 * in production's nesting order. */
const mountProvider = (
  createFactory: (client: telem.Client | null) => telem.CompoundFactory,
): MountedProvider => {
  const registry: aether.ComponentRegistry = {
    ...alamos.REGISTRY,
    ...status.REGISTRY,
    [telem.PROVIDER_TYPE]: telem.createProvider(createFactory),
  };
  const driver = aetherTest.createDriver(registry);
  const path = [aetherTest.ROOT_KEY, ALAMOS_KEY];
  driver.update(path, alamos.Provider.TYPE, alamos.providerStateZ.parse({}));
  path.push(STATUS_KEY);
  driver.update(
    path,
    status.Aggregator.TYPE,
    status.aggregatorStateZ.parse({ statuses: [] }),
  );
  const aggregator = driver.find<status.Aggregator>([...path]);
  path.push(TELEM_KEY);
  const update = (): void => driver.update([...path], telem.PROVIDER_TYPE, {});
  update();
  return { update, aggregator };
};

describe("telem.Provider", () => {
  beforeEach(() => {
    mockUse.mockReset();
  });

  it("builds the telemetry context from the current core", () => {
    const core = stubCore();
    mockUse.mockReturnValue(core);
    const spy = vi.fn((client: telem.Client | null) => telem.createFactory(client));
    mountProvider(spy);
    expect(spy).toHaveBeenCalledTimes(1);
    expect(core.openFeed).toHaveBeenCalledTimes(1);
    expect(spy).toHaveBeenCalledWith({
      feed: core.openFeed.mock.results[0].value,
      channels: core.channels,
    });
  });

  it("rebuilds the context when the core swaps", () => {
    const spy = vi.fn((client: telem.Client | null) => telem.createFactory(client));
    mockUse.mockReturnValue(null);
    const { update } = mountProvider(spy);
    expect(spy).toHaveBeenCalledTimes(1);
    expect(spy).toHaveBeenLastCalledWith(null);
    const core = stubCore();
    mockUse.mockReturnValue(core);
    update();
    expect(spy).toHaveBeenCalledTimes(2);
    expect(spy).toHaveBeenLastCalledWith({
      feed: core.openFeed.mock.results[0].value,
      channels: core.channels,
    });
  });

  it("does not rebuild the context when the core is unchanged", () => {
    const core = stubCore();
    mockUse.mockReturnValue(core);
    const spy = vi.fn((client: telem.Client | null) => telem.createFactory(client));
    const { update } = mountProvider(spy);
    update();
    expect(spy).toHaveBeenCalledTimes(1);
  });

  it("hands the feed instrumentation so cache anomalies are not silenced", () => {
    const core = stubCore();
    mockUse.mockReturnValue(core);
    mountProvider((client) => telem.createFactory(client));
    const [props] = core.openFeed.mock.calls[0] as [{ instrumentation?: unknown }];
    expect(props.instrumentation).toBeDefined();
  });

  it("reports a failed feed close to the status aggregator", async () => {
    const core = stubCore();
    const err = new Error("close failed");
    core.openFeed.mockReturnValue({
      read: async () => new MultiSeries([]),
      stream: () => telemTest.mockSubscription(() => {}),
      close: async () => {
        throw err;
      },
    });
    mockUse.mockReturnValue(core);
    const { update, aggregator } = mountProvider((client) =>
      telem.createFactory(client),
    );
    mockUse.mockReturnValue(null);
    update();
    await expect.poll(() => aggregator.state.statuses).toHaveLength(1);
    expect(aggregator.state.statuses[0].message).toEqual(
      "failed to close telemetry feed",
    );
  });
});
