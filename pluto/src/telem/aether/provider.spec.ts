// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { MultiSeries } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";

const { mockUse } = vi.hoisted(() => ({ mockUse: vi.fn() }));

// Type assertions below follow existing vi.mock patterns (vitest doesn't expose module
// types from importOriginal without import() annotations, which lint forbids).
vi.mock("@/synnax/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { synnax: { ...(actual.synnax as object), use: mockUse } };
});

const MockSender = { send: vi.fn() };
const NOOP = alamos.Instrumentation.NOOP;

const update = (provider: aether.Component, key: string): void => {
  provider._updateState({
    path: [key],
    state: {},
    type: telem.PROVIDER_TYPE,
    create: () => {
      throw new Error("should not create a child");
    },
  });
};

const stubCore = (): telem.Client => ({
  telem: {
    read: async () => new MultiSeries([]),
    stream: () => telemTest.mockSubscription(() => {}),
  },
  channels: {
    retrieve: async () => {
      throw new Error("unused");
    },
  },
});

const makeProvider = (
  createFactory: (client: telem.Client | null) => telem.CompoundFactory,
  key: string,
): aether.Component => {
  const Provider = telem.createProvider(createFactory);
  const props: aether.ComponentConstructorProps = {
    path: [key],
    type: telem.PROVIDER_TYPE,
    sender: MockSender,
    instrumentation: NOOP,
    parent: null,
  };
  return new Provider(props);
};

describe("telem.Provider", () => {
  beforeEach(() => {
    mockUse.mockReset();
  });

  it("builds the telemetry context from the current core", () => {
    const core = stubCore();
    mockUse.mockReturnValue(core);
    const spy = vi.fn((client: telem.Client | null) => telem.createFactory(client));
    const provider = makeProvider(spy, "telem-provider");
    update(provider, "telem-provider");
    expect(spy).toHaveBeenCalledTimes(1);
    expect(spy).toHaveBeenCalledWith(core);
  });

  it("rebuilds the context when the core swaps", () => {
    const spy = vi.fn((client: telem.Client | null) => telem.createFactory(client));
    const provider = makeProvider(spy, "telem-provider-swap");
    mockUse.mockReturnValue(null);
    update(provider, "telem-provider-swap");
    expect(spy).toHaveBeenCalledTimes(1);
    expect(spy).toHaveBeenLastCalledWith(null);
    const core = stubCore();
    mockUse.mockReturnValue(core);
    update(provider, "telem-provider-swap");
    expect(spy).toHaveBeenCalledTimes(2);
    expect(spy).toHaveBeenLastCalledWith(core);
  });

  it("does not rebuild the context when the core is unchanged", () => {
    const core = stubCore();
    mockUse.mockReturnValue(core);
    const spy = vi.fn((client: telem.Client | null) => telem.createFactory(client));
    const provider = makeProvider(spy, "telem-provider-stable");
    update(provider, "telem-provider-stable");
    update(provider, "telem-provider-stable");
    expect(spy).toHaveBeenCalledTimes(1);
  });
});
