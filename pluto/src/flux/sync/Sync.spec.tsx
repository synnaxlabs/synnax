// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer, newTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { render, renderHook, waitFor } from "@testing-library/react";
import { act, useEffect, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Sync } from "@/flux/sync";
import { newSynnaxWrapper } from "@/testutil/Synnax";

describe("sync", () => {
  describe("nominal path", () => {
    it("should add a basic listener", async () => {
      const client = newTestClient();
      const testChannelName = `sync_test_channel_${id.create()}`;
      await client.channels.create({
        name: testChannelName,
        dataType: "string",
        virtual: true,
      });
      const writer = await client.openWriter([testChannelName]);
      const { result } = renderHook(
        () => {
          const [data, setData] = useState<string[]>([]);
          const [open, setOpen] = useState(false);
          const addListener = Sync.useAddListener();
          useEffect(
            () =>
              addListener({
                channel: testChannelName,
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
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(async () => expect(result.current.open).toEqual(true));
      await act(
        async () => await writer.write({ [testChannelName]: "write number one" }),
      );
      await waitFor(async () =>
        expect(result.current.data).toEqual(["write number one"]),
      );
      await writer.close();
    });
  });

  describe("streamer lifecycle", () => {
    class MockStreamer implements framer.Streamer {
      private keysI: channel.Params[];
      readonly updateVi = vi.fn();
      readonly closeVi = vi.fn();
      readonly iteratorVi = vi.fn();
      readonly nextVi = vi.fn();
      readonly reads?: framer.Frame[];
      readonly nextFn?: () => Promise<IteratorResult<framer.Frame>>;

      constructor(
        keys: channel.Keys,
        nextFn?: () => Promise<IteratorResult<framer.Frame>>,
        reads?: framer.Frame[],
      ) {
        this.keysI = [keys];
        this.reads = reads;
        this.nextFn = nextFn;
      }

      get keys(): channel.Keys {
        return this.keysI.at(-1) as channel.Keys;
      }

      update(keys: channel.Params): Promise<void> {
        this.keysI.push(keys);
        this.updateVi(keys);
        return Promise.resolve();
      }

      close(): void {
        this.closeVi();
      }

      async next(): Promise<IteratorResult<framer.Frame>> {
        if (this.reads == null && this.nextFn == null)
          throw new Error("no next function or reads provided");
        if (this.nextFn != null) return await this.nextFn();
        const fr = this.reads?.shift();
        this.nextVi(fr);
        if (fr == null) return { done: true, value: undefined };
        return { done: false, value: fr };
      }

      async read(): Promise<framer.Frame> {
        const res = await this.next();
        if (res.done) throw new Error("no more frames");
        return res.value;
      }

      [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
        this.iteratorVi();
        return this;
      }
    }

    it("should not open the streamer if no listeners are added", () => {
      const openStreamer = vi.fn();
      render(
        <Sync.Provider
          openStreamer={async () => {
            openStreamer();
            throw new Error("should not be called");
          }}
        />,
      );
      expect(openStreamer).not.toHaveBeenCalled();
    });

    it("should open the streamer when a listener is added", async () => {
      const openStreamer = vi.fn();
      const streamer = new MockStreamer([], async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return { done: false, value: new framer.Frame([]) };
      });
      const { result } = renderHook(() => Sync.useAddListener(), {
        wrapper: ({ children }) => (
          <Sync.Provider
            openStreamer={async () => {
              openStreamer();
              return streamer;
            }}
          >
            {children}
          </Sync.Provider>
        ),
      });
      expect(openStreamer).toHaveBeenCalledTimes(0);
      act(() => {
        result.current({
          channel: "test_channel",
          handler: vi.fn(),
          onOpen: vi.fn(),
        });
      });
      await waitFor(async () => {
        expect(openStreamer).toHaveBeenCalled();
      });
    });

    it("should only open the streamer once even if multiple listeners are added", async () => {
      const openStreamer = vi.fn();
      const streamer = new MockStreamer([], async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return { done: false, value: new framer.Frame([]) };
      });
      const { result } = renderHook(() => Sync.useAddListener(), {
        wrapper: ({ children }) => (
          <Sync.Provider
            openStreamer={async () => {
              openStreamer();
              return streamer;
            }}
          >
            {children}
          </Sync.Provider>
        ),
      });
      expect(openStreamer).toHaveBeenCalledTimes(0);
      act(() => {
        result.current({
          channel: "test_channel_1",
          handler: vi.fn(),
          onOpen: vi.fn(),
        });
        result.current({
          channel: "test_channel_2",
          handler: vi.fn(),
          onOpen: vi.fn(),
        });
      });
      await waitFor(async () => {
        expect(openStreamer).toHaveBeenCalledTimes(1);
      });
    });

    it("should call update if multiple listeners are added", async () => {
      const mockStreamer = new MockStreamer([], async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return { done: false, value: new framer.Frame([]) };
      });
      const openStreamer = vi.fn().mockResolvedValue(mockStreamer);
      const { result } = renderHook(() => Sync.useAddListener(), {
        wrapper: ({ children }) => (
          <Sync.Provider openStreamer={openStreamer}>{children}</Sync.Provider>
        ),
      });
      const addListener = result.current;
      act(() => {
        addListener({
          channel: "test_channel_1",
          handler: vi.fn(),
          onOpen: vi.fn(),
        });
      });
      expect(mockStreamer.updateVi).toHaveBeenCalledTimes(0);
      act(() => {
        addListener({
          channel: "test_channel_2",
          handler: vi.fn(),
          onOpen: vi.fn(),
        });
      });
      await waitFor(async () => {
        expect(mockStreamer.updateVi).toHaveBeenLastCalledWith([
          "test_channel_1",
          "test_channel_2",
        ]);
      });
    });

    it("should close the streamer when all listeners are removed", async () => {
      const mockStreamer = new MockStreamer([], async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return { done: false, value: new framer.Frame([]) };
      });
      const openStreamer = vi.fn().mockResolvedValue(mockStreamer);
      const { result } = renderHook(() => Sync.useAddListener(), {
        wrapper: ({ children }) => (
          <Sync.Provider openStreamer={openStreamer}>{children}</Sync.Provider>
        ),
      });
      const addListener = result.current;
      const [destroyFirst, destroySecond] = await act(() => [
        addListener({
          channel: "test_channel_1",
          handler: vi.fn(),
          onOpen: vi.fn(),
        }),
        addListener({
          channel: "test_channel_2",
          handler: vi.fn(),
          onOpen: vi.fn(),
        }),
      ]);
      await waitFor(async () => {
        expect(openStreamer).toHaveBeenCalledTimes(1);
        expect(mockStreamer.closeVi).toHaveBeenCalledTimes(0);
      });
      destroyFirst();
      expect(mockStreamer.closeVi).not.toHaveBeenCalled();
      destroySecond();
      await waitFor(async () => {
        expect(mockStreamer.closeVi).toHaveBeenCalledTimes(1);
      });
    });

    it("should close the streamer when the provider is unmounted", async () => {
      const mockStreamer = new MockStreamer([], async () => {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return { done: false, value: new framer.Frame([]) };
      });
      const openStreamer = vi.fn().mockResolvedValue(mockStreamer);
      const { result, unmount } = renderHook(() => Sync.useAddListener(), {
        wrapper: ({ children }) => (
          <Sync.Provider openStreamer={openStreamer}>{children}</Sync.Provider>
        ),
      });
      const addListener = result.current;
      act(() => {
        addListener({
          channel: "test_channel_1",
          handler: vi.fn(),
          onOpen: vi.fn(),
        });
      });
      await waitFor(async () => {
        expect(openStreamer).toHaveBeenCalledTimes(1);
        expect(mockStreamer.closeVi).toHaveBeenCalledTimes(0);
      });
      unmount();
      await waitFor(async () => {
        expect(mockStreamer.closeVi).toHaveBeenCalledTimes(1);
      });
    });
  });
});
