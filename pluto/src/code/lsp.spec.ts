// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status, type Synnax } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import {
  type LSPMessage,
  type LSPStream,
  useLanguageServer,
  type UseLanguageServerArgs,
} from "@/code/lsp";

const { startMock, stopMock } = vi.hoisted(() => ({
  startMock: vi.fn(),
  stopMock: vi.fn(),
}));

// The real language client pulls in monaco; replace it with a class whose start/stop are
// the only behaviors useLanguageServer drives.
vi.mock("vscode-languageclient/browser", () => {
  class BaseLanguageClient {
    start = startMock;
    stop = stopMock;
  }
  return {
    BaseLanguageClient,
    CloseAction: { DoNotRestart: 1 },
    ErrorAction: { Continue: 1 },
  };
});

const CLIENT = {} as Synnax;
const MONACO = {} as unknown;

// makeStream returns an LSPStream whose receive never settles, so a "connected" client
// stays connected until the hook aborts it.
const makeStream = (): LSPStream & {
  closeSend: ReturnType<typeof vi.fn<() => void>>;
} => ({
  receive: () => new Promise<LSPMessage>(() => {}),
  send: vi.fn(),
  closeSend: vi.fn<() => void>(),
});

type StatusFn = (s: status.Status) => void;

describe("useLanguageServer", () => {
  let onStatus: ReturnType<typeof vi.fn<StatusFn>>;

  const statuses = (): status.Status[] => onStatus.mock.calls.map((c) => c[0]);
  const variants = (): string[] => statuses().map((s) => s.variant);
  const last = (): status.Status => statuses().at(-1) as status.Status;

  const renderLSP = (overrides: Partial<UseLanguageServerArgs> = {}) =>
    renderHook(() =>
      useLanguageServer({
        monaco: MONACO,
        client: CLIENT,
        languageID: "arc",
        open: vi.fn().mockResolvedValue(makeStream()),
        onStatus,
        ...overrides,
      }),
    );

  // The first client start cold-loads the monaco runtime, whose deep CSS imports cannot
  // be evaluated under Node's ESM loader. That failure is a test-env artifact (CSS loads
  // fine in the browser), so absorb it once here rather than letting it surface in the
  // first connection assertion.
  beforeAll(async () => {
    startMock.mockResolvedValue(undefined);
    const warmStatus = vi.fn<StatusFn>();
    renderHook(() =>
      useLanguageServer({
        monaco: MONACO,
        client: CLIENT,
        languageID: "arc",
        open: vi.fn().mockResolvedValue(makeStream()),
        onStatus: warmStatus,
      }),
    );
    await waitFor(() => expect(warmStatus.mock.calls.length).toBeGreaterThan(1));
  });

  beforeEach(() => {
    onStatus = vi.fn<StatusFn>();
    startMock.mockReset().mockResolvedValue(undefined);
    stopMock.mockReset().mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("inactive gating", () => {
    it.each([
      { name: "monaco is absent", monaco: null, client: CLIENT },
      { name: "the client is absent", monaco: MONACO, client: null },
    ])(
      "should report inactive and not open a stream when $name",
      async ({ monaco, client }) => {
        const open = vi.fn();
        renderLSP({ monaco, client, open });
        await waitFor(() => expect(last().variant).toEqual("disabled"));
        expect(last().message).toEqual("Language server inactive");
        expect(open).not.toHaveBeenCalled();
      },
    );
  });

  describe("connection lifecycle", () => {
    it("should open the stream against the client and transition to connected", async () => {
      const stream = makeStream();
      const open = vi.fn().mockResolvedValue(stream);
      renderLSP({ open });
      await waitFor(() => expect(last().variant).toEqual("success"));
      expect(open).toHaveBeenCalledWith(CLIENT);
      expect(startMock).toHaveBeenCalled();
      expect(variants()).toEqual(["loading", "success"]);
      expect(statuses()[0].message).toEqual("Connecting to language server");
      expect(last().message).toEqual("Language server connected");
    });

    it("should tear down the client and stream and report inactive on unmount", async () => {
      const stream = makeStream();
      const open = vi.fn().mockResolvedValue(stream);
      const { unmount } = renderLSP({ open });
      await waitFor(() => expect(last().variant).toEqual("success"));
      onStatus.mockClear();
      unmount();
      await waitFor(() => expect(stopMock).toHaveBeenCalled());
      expect(stream.closeSend).toHaveBeenCalled();
      expect(last().variant).toEqual("disabled");
    });

    it("should not start a client when aborted while the stream is opening", async () => {
      const stream = makeStream();
      let resolveOpen: (s: LSPStream) => void = () => {};
      const open = vi.fn(
        () =>
          new Promise<LSPStream>((resolve) => {
            resolveOpen = resolve;
          }),
      );
      const { unmount } = renderLSP({ open });
      await waitFor(() => expect(open).toHaveBeenCalled());
      unmount();
      await act(async () => {
        resolveOpen(stream);
        await Promise.resolve();
      });
      expect(startMock).not.toHaveBeenCalled();
      expect(stream.closeSend).toHaveBeenCalled();
    });
  });

  describe("reconnection", () => {
    it("should retry after a failed attempt and recover", async () => {
      vi.useFakeTimers();
      const stream = makeStream();
      const open = vi
        .fn()
        .mockRejectedValueOnce(new Error("server down"))
        .mockResolvedValue(stream);
      renderLSP({ open });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(50);
      });
      expect(open).toHaveBeenCalledTimes(1);
      expect(variants()).toContain("error");
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1500);
      });
      expect(open).toHaveBeenCalledTimes(2);
      expect(last().variant).toEqual("success");
    });
  });
});
