// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useAsyncEffect } from "@/hooks/useAsyncEffect";

describe("useAsyncEffect", () => {
  it("should call the effect function", async () => {
    let state = 0;
    const effect = vi.fn(async (signal: AbortSignal) => {
      state++;
      expect(signal.aborted).toBe(false);
    });
    const { rerender, unmount } = renderHook(() => useAsyncEffect(effect, [state]));
    await scheduler.flushTaskQueue();
    expect(state).toBe(1);
    rerender();
    await scheduler.flushTaskQueue();
    expect(state).toBe(2);
    unmount();
    await scheduler.flushTaskQueue();
    expect(state).toBe(2);
  });
  it("should cleanup", async () => {
    let connectedNumber = 0;
    const effect = vi.fn(async (signal: AbortSignal) => {
      expect(signal.aborted).toBe(false);
      connectedNumber++;
      return () => {
        connectedNumber--;
      };
    });

    const { rerender, unmount } = renderHook(() => useAsyncEffect(effect));
    await scheduler.flushTaskQueue();
    expect(connectedNumber).toBe(1);

    rerender();
    await scheduler.flushTaskQueue();
    expect(connectedNumber).toBe(1);

    unmount();
    await scheduler.flushTaskQueue();
    expect(connectedNumber).toBe(0);
  });
  it("should still cleanup previous effects even if the callback has not finished by the time the hook rerenders", async () => {
    class Server {
      private connections: Set<string> = new Set();
      private firstConnectionPromise: Promise<void> | null = null;
      private pendingFirstConnection: string | null = null;

      connect(id: string): Promise<void> {
        if (this.firstConnectionPromise == null) {
          this.pendingFirstConnection = id;
          this.firstConnectionPromise = new Promise((resolve) => {
            const originalAdd = this.connections.add.bind(this.connections);
            this.connections.add = (nextId: string): Set<string> => {
              originalAdd(nextId);
              // After the second connection is added, add the first connection
              if (this.pendingFirstConnection != null) {
                originalAdd(this.pendingFirstConnection);
                this.pendingFirstConnection = null;
              }
              this.connections.add = originalAdd;
              resolve();
              return this.connections;
            };
          });
          return this.firstConnectionPromise;
        }
        this.connections.add(id);
        return Promise.resolve();
      }

      disconnect(id: string): void {
        this.connections.delete(id);
      }

      isConnected(id: string): boolean {
        return this.connections.has(id);
      }

      get connectionsCount(): number {
        return this.connections.size;
      }
    }

    const server = new Server();
    let count = 0;

    let cleanupsCalled = 0;

    const effect = vi.fn(async (signal: AbortSignal) => {
      expect(signal.aborted).toBe(false);
      const currID = `id-${count++}`;
      await server.connect(currID);
      return () => {
        server.disconnect(currID);
        cleanupsCalled++;
      };
    });
    const { rerender, unmount } = renderHook(() => useAsyncEffect(effect));
    await scheduler.flushTaskQueue();
    expect(server.isConnected("id-0")).toBe(false);
    expect(server.isConnected("id-1")).toBe(false);
    expect(server.connectionsCount).toBe(0);
    expect(cleanupsCalled).toBe(0);

    rerender();
    await scheduler.flushTaskQueue();
    expect(server.isConnected("id-0")).toBe(false);
    expect(server.isConnected("id-1")).toBeTruthy();
    expect(server.connectionsCount).toBe(1);
    expect(cleanupsCalled).toBe(1);

    unmount();
    await scheduler.flushTaskQueue();
    expect(server.isConnected("id-0")).toBe(false);
    expect(server.isConnected("id-1")).toBe(false);
    expect(server.connectionsCount).toBe(0);
    expect(cleanupsCalled).toBe(2);
  });

  it("should not finish running the effect if the hook is unmounted before the effect is finished", async () => {
    let state = 0;
    const effect = vi.fn(async (signal: AbortSignal) => {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      if (signal.aborted) return;
      state++;
    });

    vi.useFakeTimers();

    const { unmount } = renderHook(() => useAsyncEffect(effect));
    unmount();
    vi.advanceTimersByTime(2000);
    vi.useRealTimers();
    await scheduler.flushTaskQueue();
    expect(state).toBe(0);
  });

  it("should have an error if the effect throws an error", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const effect = vi.fn(async () => {
      throw new Error("test");
    });
    renderHook(() => useAsyncEffect(effect, undefined));
    await scheduler.flushTaskQueue();
    expect(consoleSpy).toHaveBeenCalledWith(new Error("test"));
  });
});
