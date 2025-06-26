// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act, type PropsWithChildren, useEffect, useState } from "react";
import { describe, expect, it } from "vitest";

import { Sync } from "@/query/sync";

describe("sync", () => {
  it("should add a basic listener", async () => {
    const client = newTestClient();
    const Wrapper = (props: PropsWithChildren) => (
      <Sync.Provider useClient={() => client} {...props} />
    );
    const testChannelName = `sync_test_channel_${id.create()}`;
    await client.channels.create({
      name: testChannelName,
      dataType: "string",
      virtual: true,
    });
    const writer = await client.openWriter([testChannelName]);
    const { result, unmount } = renderHook(
      () => {
        const [data, setData] = useState<string[]>([]);
        const [open, setOpen] = useState(false);
        const addListener = Sync.useAddListener();
        useEffect(
          () =>
            addListener({
              channels: testChannelName,
              handler: (frame) => {
                setData((state) => [
                  ...state,
                  ...frame.series.flatMap((s) => s.toStrings()).flat(),
                ]);
              },
              onOpen: () => setOpen(true),
            }),
          [addListener],
        );
        return { data, open };
      },
      { wrapper: Wrapper },
    );
    await waitFor(async () => expect(result.current.open).toEqual(true));
    await act(
      async () => await writer.write({ [testChannelName]: "write number one" }),
    );
    await waitFor(async () =>
      expect(result.current.data).toEqual(["write number one"]),
    );
    await writer.close();
    unmount();
  });

  // it("should handle updates for other listeners when one throws an error", async () => {
  //   const errChannelName = "sync_test_channel_error";
  //   const successChannelName = "sync_test_channel_success";

  //   const useErrorListener = () => {
  //     const timesReceivedRef = useRef(0);
  //     const addListener = Sync.useAddListener();
  //     useEffect(() =>
  //       addListener({
  //         channels: errChannelName,
  //         handler: () => {
  //           timesReceivedRef.current++;
  //           if (timesReceivedRef.current > 1) throw new Error("test error");
  //         },
  //       }),
  //     );
  //   };

  //   const useSuccessListener = () => {
  //     const [data, setData] = useState<string[]>([]);
  //     Sync.useListener({
  //       channel: successChannelName,
  //       onChange: Sync.stringHandler(async ({ changed }) =>
  //         setData((prev) => [...prev, changed]),
  //       ),
  //     });
  //     return data;
  //   };
  //   const useBothListeners = () => {
  //     const connectToClient = useConnectToClient();
  //     useEffect(() => connectToClient(true), [connectToClient]);
  //     useErrorListener();
  //     const statuses = Status.useNotifications().statuses;
  //     const successFrames = useSuccessListener();
  //     const isStreamerOpen = Query.useStreamerIsOpen();
  //     return { successFrames, isStreamerOpen, statuses };
  //   };
  //   const client = newTestClient();
  //   await client.channels.create(
  //     [
  //       { name: errChannelName, dataType: "string", virtual: true },
  //       { name: successChannelName, dataType: "string", virtual: true },
  //     ],
  //     { retrieveIfNameExists: true },
  //   );
  //   const writer = await client.openWriter([errChannelName, successChannelName]);
  //   const { result } = renderHook(useBothListeners, { wrapper: TestProvider });
  //   await waitFor(async () => expect(result.current.isStreamerOpen).toBe(true));
  //   await act(async () => writer.write({ [successChannelName]: "write number 1" }));
  //   await waitFor(async () =>
  //     expect(result.current.successFrames).toEqual(["write number 1"]),
  //   );
  //   await act(async () => writer.write({ [errChannelName]: "error 1" }));
  //   await act(async () => writer.write({ [errChannelName]: "error 2" }));
  //   await waitFor(async () => {
  //     expect(
  //       result.current.statuses.some((s) => s.description?.includes("test error")),
  //     ).toBeTruthy();
  //   });
  //   await act(async () => writer.write({ [successChannelName]: "write number 2" }));
  //   await waitFor(async () =>
  //     expect(result.current.successFrames).toEqual([
  //       "write number 1",
  //       "write number 2",
  //     ]),
  //   );
  // });

  // it("should handle when the streamer throws an error", async () => {
  //   // Enable the mock for this test
  //   shouldMockStreamer = true;

  //   const use = () => {
  //     const connectToClient = useConnectToClient();
  //     useEffect(() => connectToClient(true), [connectToClient]);
  //     const addListener = Query.useAddListener();
  //     useEffect(
  //       () =>
  //         addListener({
  //           channels: "sync_test_channel",
  //           handler: () => {},
  //         }),
  //       [],
  //     );
  //     return Status.useNotifications().statuses;
  //   };
  //   const { result } = renderHook(use, { wrapper: TestProvider });
  //   await waitFor(async () => {
  //     expect(
  //       result.current.some(
  //         (s) =>
  //           s.description?.includes("opening streamer error") && s.variant === "error",
  //       ),
  //     ).toBe(true);
  //   });
  // });
});
