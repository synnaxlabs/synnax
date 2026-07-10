// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { type client } from "@/telem/client";

const { mockUse, mockClose } = vi.hoisted(() => ({
  mockUse: vi.fn(),
  mockClose: vi.fn(async () => {}),
}));

// Type assertions below follow existing vi.mock patterns (vitest doesn't expose module
// types from importOriginal without import() annotations, which lint forbids).
vi.mock("@/synnax/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { synnax: { ...(actual.synnax as object), use: mockUse } };
});

vi.mock("@/telem/client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  class MockCore implements client.Client {
    async retrieveChannel(): ReturnType<client.ChannelClient["retrieveChannel"]> {
      throw new Error("not implemented in MockCore");
    }

    async read(): ReturnType<client.ReadClient["read"]> {
      throw new Error("not implemented in MockCore");
    }

    async stream(): ReturnType<client.StreamClient["stream"]> {
      return () => {};
    }

    close = mockClose;
  }
  return { client: { ...(actual.client as object), Core: MockCore } };
});

// BaseProvider.afterUpdate/afterDelete route client-close errors through
// status.useErrorHandler(ctx), which normally reads the status aggregator's context
// value. Bypass the context dependency but keep the real fire-and-forget error-handling
// semantics by wiring the real createErrorHandler to a no-op adder.
vi.mock("@/status/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const realStatus = actual.status as { createErrorHandler: (add: unknown) => unknown };
  const noopAdd = (): void => {};
  return {
    status: {
      ...(actual.status as object),
      useErrorHandler: () => realStatus.createErrorHandler(noopAdd),
    },
  };
});

// afterUpdate also pulls instrumentation from the alamos context, which this isolated
// tree never provisions. Bypass it with a no-op instrumentation instance.
vi.mock("@/alamos/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const { alamos: pkg } = await import("@synnaxlabs/alamos");
  return {
    alamos: {
      ...(actual.alamos as object),
      useInstrumentation: (_ctx: unknown, name?: string) =>
        name == null ? pkg.Instrumentation.NOOP : pkg.Instrumentation.NOOP.child(name),
    },
  };
});

const MockSender = { send: vi.fn() };
const NOOP = alamos.Instrumentation.NOOP;

const makeProvider = (key: string): aether.Component => {
  const props: aether.ComponentConstructorProps = {
    path: [key],
    type: telem.PROVIDER_TYPE,
    sender: MockSender,
    instrumentation: NOOP,
    parent: null,
  };
  return new telem.Provider(props);
};

const shouldNotCreate = (): never => {
  throw new Error("should not create a child");
};

describe("telem.Provider", () => {
  beforeEach(() => {
    mockUse.mockReset();
    mockClose.mockClear();
  });

  it("closes the underlying client when the node is deleted", async () => {
    mockUse.mockReturnValue({});
    const key = "telem-provider";
    const provider = makeProvider(key);

    provider._updateState({
      path: [key],
      state: {},
      type: telem.PROVIDER_TYPE,
      create: shouldNotCreate,
    });
    expect(mockClose).not.toHaveBeenCalled();

    provider._delete([key]);
    await expect.poll(() => mockClose.mock.calls.length).toBe(1);
  });

  it("closes the previous client when the underlying core swaps", async () => {
    const key = "telem-provider-swap";
    const provider = makeProvider(key);

    mockUse.mockReturnValue({});
    provider._updateState({
      path: [key],
      state: {},
      type: telem.PROVIDER_TYPE,
      create: shouldNotCreate,
    });
    expect(mockClose).not.toHaveBeenCalled();

    mockUse.mockReturnValue({});
    provider._updateState({
      path: [key],
      state: {},
      type: telem.PROVIDER_TYPE,
      create: shouldNotCreate,
    });
    await expect.poll(() => mockClose.mock.calls.length).toBe(1);
  });
});
