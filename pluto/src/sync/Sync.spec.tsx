// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, useEffect, useRef, useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Status } from "@/status";
import { Sync } from "@/sync";
import { SynnaxProvider, useConnectToClient } from "@/testutil/Synnax";

let shouldMockStreamer = false;

vi.mock("@synnaxlabs/client", async () => {
  const actual = await vi.importActual<any>("@synnaxlabs/client");
  return {
    ...actual,
    framer: {
      ...actual.framer,
      HardenedStreamer: {
        ...actual.framer.HardenedStreamer,
        open: vi.fn((...args) => {
          if (shouldMockStreamer) throw new Error("opening streamer error");

          return actual.framer.HardenedStreamer.open(...args);
        }),
      },
    },
  };
});

const TestProvider = (props: PropsWithChildren) => (
  <SynnaxProvider defaultConnected={false}>
    <Sync.Provider {...props} />
  </SynnaxProvider>
);

describe("sync", () => {
  beforeEach(() => {
    vi.doUnmock("@synnaxlabs/client");
    shouldMockStreamer = false;
  });

  it("should add a basic listener", async () => {
    const test_channel_name = "sync_test_channel";
    const useSync = () => {
      const [data, setData] = useState<string[]>([]);
      const isStreamerOpen = Sync.useStreamerIsOpen();
      const addListener = Sync.useAddListener();
      const connectToClient = useConnectToClient();
      useEffect(
        () =>
          addListener({
            channels: test_channel_name,
            handler: (frame) => {
              setData((state) => [
                ...state,
                ...frame.series.flatMap((s) => s.toStrings()).flat(),
              ]);
            },
          }),
        [addListener],
      );
      return { isStreamerOpen, data, connectToClient };
    };
    const client = newTestClient();
    await client.channels.create(
      { name: test_channel_name, dataType: "string", virtual: true },
      { retrieveIfNameExists: true },
    );
    const writer = await client.openWriter(test_channel_name);
    const { result } = renderHook(useSync, { wrapper: TestProvider });

    await act(async () => result.current.connectToClient(true));
    await waitFor(async () => expect(result.current.isStreamerOpen).toBe(true));
    await act(async () => writer.write({ [test_channel_name]: "write number 1" }));
    await waitFor(async () => expect(result.current.data).toEqual(["write number 1"]));
    await act(async () => writer.write({ [test_channel_name]: "write number 2" }));
    await waitFor(async () =>
      expect(result.current.data).toEqual(["write number 1", "write number 2"]),
    );
  });

  it("should handle updates for other listeners when one throws an error", async () => {
    const error_channel_name = "sync_test_channel_error";
    const success_channel_name = "sync_test_channel_success";

    const useErrorListener = () => {
      const timesReceivedRef = useRef(0);
      const addListener = Sync.useAddListener();
      useEffect(() =>
        addListener({
          channels: error_channel_name,
          handler: () => {
            timesReceivedRef.current++;
            if (timesReceivedRef.current > 1) throw new Error("test error");
          },
        }),
      );
    };

    const useSuccessListener = () => {
      const [data, setData] = useState<string[]>([]);
      Sync.useStringListener(
        success_channel_name,
        (s) => s,
        (data) => setData((prev) => [...prev, data]),
      );
      return data;
    };
    const useBothListeners = () => {
      const connectToClient = useConnectToClient();
      useEffect(() => connectToClient(true), [connectToClient]);
      useErrorListener();
      const statuses = Status.useNotifications().statuses;
      const successFrames = useSuccessListener();
      const isStreamerOpen = Sync.useStreamerIsOpen();
      return { successFrames, isStreamerOpen, statuses };
    };
    const client = newTestClient();
    await client.channels.create(
      [
        { name: error_channel_name, dataType: "string", virtual: true },
        { name: success_channel_name, dataType: "string", virtual: true },
      ],
      { retrieveIfNameExists: true },
    );
    const writer = await client.openWriter([error_channel_name, success_channel_name]);
    const { result } = renderHook(useBothListeners, { wrapper: TestProvider });
    await waitFor(async () => expect(result.current.isStreamerOpen).toBe(true));
    await act(async () => writer.write({ [success_channel_name]: "write number 1" }));
    await waitFor(async () =>
      expect(result.current.successFrames).toEqual(["write number 1"]),
    );
    await act(async () => writer.write({ [error_channel_name]: "error 1" }));
    await act(async () => writer.write({ [error_channel_name]: "error 2" }));
    await waitFor(async () => {
      expect(
        result.current.statuses.some((s) => s.description?.includes("test error")),
      ).toBeTruthy();
    });
    await act(async () => writer.write({ [success_channel_name]: "write number 2" }));
    await waitFor(async () =>
      expect(result.current.successFrames).toEqual([
        "write number 1",
        "write number 2",
      ]),
    );
  });

  it("should handle when the streamer throws an error", async () => {
    // Enable the mock for this test
    shouldMockStreamer = true;

    const use = () => {
      const connectToClient = useConnectToClient();
      useEffect(() => connectToClient(true), [connectToClient]);
      const addListener = Sync.useAddListener();
      useEffect(
        () =>
          addListener({
            channels: "sync_test_channel",
            handler: () => {},
          }),
        [],
      );
      return Status.useNotifications().statuses;
    };
    const { result } = renderHook(use, { wrapper: TestProvider });
    await waitFor(async () => {
      expect(
        result.current.some(
          (s) =>
            s.description?.includes("opening streamer error") && s.variant === "error",
        ),
      ).toBe(true);
    });
  });
});
