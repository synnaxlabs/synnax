// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, Frame, type framer, newTestClient } from "@synnaxlabs/client";
import { EOF } from "@synnaxlabs/freighter";
import { Series } from "@synnaxlabs/x";
import { describe, expect, it, type Mock, vi } from "vitest";
import z from "zod";

import { flux } from "@/flux/aether";
import { type Status } from "@/status";

// ==================== Test Utilities ====================

class MockHardenedStreamer implements framer.Streamer {
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
    if (this.reads == null && this.nextFn == null) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      return { done: true, value: undefined };
    }
    if (this.nextFn != null) return await this.nextFn();
    const fr = this.reads?.shift();
    this.nextVi(fr);
    if (fr == null) return { done: true, value: undefined };

    return { done: false, value: fr };
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new EOF();
    return res.value;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    this.iteratorVi();
    return this;
  }
}

/**
 * Creates a mock error handler that executes async functions
 * and swallows errors to prevent test failures.
 */
const createMockErrorHandler = (): Mock =>
  vi.fn((excOrFunc: unknown) => {
    if (typeof excOrFunc === "function")
      void (async () => {
        try {
          await excOrFunc();
        } catch (_) {
          // Swallow error to prevent test failure
        }
      })();
  });

/**
 * Default mock error handler that re-throws for basic tests.
 */
const mockHandleError: Status.ErrorHandler = (excOrFunc: unknown, message?: string) => {
  if (typeof excOrFunc === "function")
    void (async () => {
      await excOrFunc();
    })();
  else mockHandleError(excOrFunc, message);
};

/**
 * Creates a streamer that yields specified frames in sequence.
 */
const createFrameStreamer = (frames: framer.Frame[]) => async () => {
  let i = 0;
  return new MockHardenedStreamer([], async () => {
    if (i >= frames.length) return { done: true, value: undefined };
    return { done: false, value: frames[i++] };
  });
};

/**
 * Creates a single frame with the given channel data.
 */
const createFrame = (data: Record<string, unknown[]>): framer.Frame => {
  const frameData: Record<string, Series> = {};
  for (const [channel, values] of Object.entries(data))
    frameData[channel] = new Series(values);

  return new Frame(frameData);
};

/**
 * Creates a basic store configuration with a single listener.
 */
const createStoreConfig = <T>(
  channel: string,
  schema: z.ZodType<T>,
  onChange: Mock,
): flux.StoreConfig<flux.Store> => ({
  labels: {
    listeners: [
      {
        channel,
        schema,
        onChange,
      },
    ],
  },
});

/**
 * Creates default streamer arguments for testing.
 */
const createStreamerArgs = (
  overrides?: Partial<flux.StreamerArgs<flux.Store>>,
): flux.StreamerArgs<flux.Store> => ({
  handleError: mockHandleError,
  storeConfig: { labels: { listeners: [] } },
  client: newTestClient(),
  store: {} as flux.Store,
  openStreamer: async () => new MockHardenedStreamer([]),
  ...overrides,
});

// ==================== Test Suites ====================

describe("openStreamer", () => {
  it("should open a streamer on a set of channels", async () => {
    const onChange = vi.fn();
    const schema = z.object({ name: z.string() });
    const frames = [createFrame({ test: [{ name: "test" }] })];

    const closeStreamer = await flux.openStreamer(
      createStreamerArgs({
        storeConfig: createStoreConfig("test", schema, onChange),
        openStreamer: createFrameStreamer(frames),
      }),
    );

    expect(closeStreamer).toBeDefined();
    await expect.poll(() => onChange.mock.calls.length).toBeGreaterThan(0);
    await closeStreamer();
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange.mock.calls[0][0].changed.name).toBe("test");
  });

  describe("Error Handling & Recovery", () => {
    it("should call handleError when schema validation fails", async () => {
      const onChange = vi.fn();
      const handleError = createMockErrorHandler();
      const schema = z.object({ name: z.string(), age: z.number() });
      // Invalid data - missing required 'age' field
      const frames = [createFrame({ test: [{ name: "test" }] })];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig: createStoreConfig("test", schema, onChange),
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => handleError.mock.calls.length).toBeGreaterThan(0);
      expect(onChange).not.toHaveBeenCalled();
      await closeStreamer();
    });

    it("should continue streaming after a listener throws an error", async () => {
      const listener1 = vi.fn().mockImplementation(() => {
        throw new Error("Listener 1 error");
      });
      const listener2 = vi.fn();
      const handleError = createMockErrorHandler();
      const schema = z.object({ value: z.number() });

      const frames = [
        createFrame({ test: [{ value: 1 }], test2: [{ value: 2 }] }),
        createFrame({ test2: [{ value: 3 }] }),
      ];

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "test", schema, onChange: listener1 },
            { channel: "test2", schema, onChange: listener2 },
          ],
        },
      };

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => listener2.mock.calls.length).toBe(2);
      expect(listener1).toHaveBeenCalledTimes(1);
      expect(handleError.mock.calls.length).toBeGreaterThan(0);
      await closeStreamer();
    });

    it("should handle EOF errors from the underlying streamer gracefully", async () => {
      const onChange = vi.fn();
      const handleError = createMockErrorHandler();
      const schema = z.object({ value: z.number() });

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig: createStoreConfig("test", schema, onChange),
          openStreamer: async () => {
            let i = 0;
            return new MockHardenedStreamer([], async () => {
              if (i === 0) {
                i++;
                return {
                  done: false,
                  value: createFrame({ test: [{ value: 1 }] }),
                };
              }
              if (i === 1) {
                i++;
                throw new EOF();
              }
              return { done: true, value: undefined };
            });
          },
        }),
      );

      await expect.poll(() => onChange.mock.calls.length).toBe(1);
      expect(onChange.mock.calls[0][0].changed.value).toBe(1);
      // Wait a bit to ensure the streamer handles the EOF
      await new Promise((resolve) => setTimeout(resolve, 100));
      await closeStreamer();
    });

    it("should recover when an onChange handler rejects", async () => {
      let callCount = 0;
      const onChange = vi.fn().mockImplementation(async () => {
        callCount++;
        if (callCount === 1) throw new Error("Async rejection");
      });
      const handleError = createMockErrorHandler();
      const schema = z.object({ value: z.number() });

      const frames = [
        createFrame({ test: [{ value: 1 }] }),
        createFrame({ test: [{ value: 2 }] }),
        createFrame({ test: [{ value: 3 }] }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig: createStoreConfig("test", schema, onChange),
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => onChange.mock.calls.length).toBe(3);
      expect(handleError.mock.calls.length).toBeGreaterThan(0);
      // First call should have errored, but subsequent calls should succeed
      expect(onChange.mock.calls[1][0].changed.value).toBe(2);
      expect(onChange.mock.calls[2][0].changed.value).toBe(3);
      await closeStreamer();
    });
  });

  describe("Multiple Listeners on Same Channel", () => {
    it("should invoke all listeners for a channel when data arrives", async () => {
      const listener1 = vi.fn();
      const listener2 = vi.fn();
      const listener3 = vi.fn();
      const schema = z.object({ value: z.number() });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "test", schema, onChange: listener1 },
            { channel: "test", schema, onChange: listener2 },
            { channel: "test", schema, onChange: listener3 },
          ],
        },
      };

      const frames = [
        createFrame({ test: [{ value: 1 }] }),
        createFrame({ test: [{ value: 2 }] }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => listener1.mock.calls.length).toBe(2);
      await expect.poll(() => listener2.mock.calls.length).toBe(2);
      await expect.poll(() => listener3.mock.calls.length).toBe(2);

      // Verify all listeners received the same data
      for (const listener of [listener1, listener2, listener3]) {
        expect(listener.mock.calls[0][0].changed.value).toBe(1);
        expect(listener.mock.calls[1][0].changed.value).toBe(2);
      }

      await closeStreamer();
    });

    it("should continue invoking other listeners when one fails", async () => {
      const listener1 = vi.fn();
      const listener2 = vi.fn().mockImplementation(() => {
        throw new Error("Listener 2 fails");
      });
      const listener3 = vi.fn();
      const handleError = createMockErrorHandler();
      const schema = z.object({ value: z.number() });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "test", schema, onChange: listener1 },
            { channel: "test", schema, onChange: listener2 },
            { channel: "test", schema, onChange: listener3 },
          ],
        },
      };

      const frames = [
        createFrame({ test: [{ value: 1 }] }),
        createFrame({ test: [{ value: 2 }] }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => listener1.mock.calls.length).toBe(2);
      await expect.poll(() => listener3.mock.calls.length).toBe(2);

      // Listener2 should have been called even though it throws
      expect(listener2).toHaveBeenCalledTimes(2);
      
      // Error handler should have been invoked for each failure
      expect(handleError.mock.calls.length).toBeGreaterThanOrEqual(2);

      // Verify other listeners still received correct data
      expect(listener1.mock.calls[0][0].changed.value).toBe(1);
      expect(listener1.mock.calls[1][0].changed.value).toBe(2);
      expect(listener3.mock.calls[0][0].changed.value).toBe(1);
      expect(listener3.mock.calls[1][0].changed.value).toBe(2);

      await closeStreamer();
    });

    it("should maintain consistent execution order for listeners", async () => {
      const executionOrder: string[] = [];
      const listener1 = vi.fn().mockImplementation(() => {
        executionOrder.push("listener1");
      });
      const listener2 = vi.fn().mockImplementation(() => {
        executionOrder.push("listener2");
      });
      const listener3 = vi.fn().mockImplementation(() => {
        executionOrder.push("listener3");
      });
      const schema = z.object({ value: z.number() });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "test", schema, onChange: listener1 },
            { channel: "test", schema, onChange: listener2 },
            { channel: "test", schema, onChange: listener3 },
          ],
        },
      };

      const frames = [
        createFrame({ test: [{ value: 1 }] }),
        createFrame({ test: [{ value: 2 }] }),
        createFrame({ test: [{ value: 3 }] }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => executionOrder.length).toBe(9);

      // Verify order is consistent across all frames
      expect(executionOrder[0]).toBe("listener1");
      expect(executionOrder[1]).toBe("listener2");
      expect(executionOrder[2]).toBe("listener3");
      expect(executionOrder[3]).toBe("listener1");
      expect(executionOrder[4]).toBe("listener2");
      expect(executionOrder[5]).toBe("listener3");
      expect(executionOrder[6]).toBe("listener1");
      expect(executionOrder[7]).toBe("listener2");
      expect(executionOrder[8]).toBe("listener3");

      await closeStreamer();
    });

    it("should handle mixed success and failure across multiple listeners", async () => {
      let callCount = 0;
      const listener1 = vi.fn();
      const listener2 = vi.fn().mockImplementation(() => {
        callCount++;
        if (callCount % 2 === 0) throw new Error("Even calls fail");
      });
      const listener3 = vi.fn();
      const handleError = createMockErrorHandler();
      const schema = z.object({ value: z.number() });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "test", schema, onChange: listener1 },
            { channel: "test", schema, onChange: listener2 },
            { channel: "test", schema, onChange: listener3 },
          ],
        },
      };

      const frames = [
        createFrame({ test: [{ value: 1 }] }),
        createFrame({ test: [{ value: 2 }] }),
        createFrame({ test: [{ value: 3 }] }),
        createFrame({ test: [{ value: 4 }] }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => listener1.mock.calls.length).toBe(4);
      await expect.poll(() => listener3.mock.calls.length).toBe(4);

      // Listener2 should have been called all times despite throwing on even calls
      expect(listener2).toHaveBeenCalledTimes(4);
      
      // Error handler should have been called for even numbered calls (2nd and 4th)
      expect(handleError.mock.calls.length).toBeGreaterThanOrEqual(2);

      // Verify all listeners received correct data
      for (let i = 0; i < 4; i++) {
        expect(listener1.mock.calls[i][0].changed.value).toBe(i + 1);
        expect(listener3.mock.calls[i][0].changed.value).toBe(i + 1);
      }

      await closeStreamer();
    });

    it("should handle multiple listeners with different schemas on same channel", async () => {
      const listener1 = vi.fn();
      const listener2 = vi.fn();
      const handleError = createMockErrorHandler();
      
      // Different schemas for same channel
      const schema1 = z.object({ value: z.number() });
      const schema2 = z.object({ value: z.string() });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "test", schema: schema1, onChange: listener1 },
            { channel: "test", schema: schema2, onChange: listener2 },
          ],
        },
      };

      // Data that satisfies schema1 but not schema2
      const frames = [createFrame({ test: [{ value: 123 }] })];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          handleError,
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => listener1.mock.calls.length).toBe(1);
      
      // Listener1 should succeed with number schema
      expect(listener1.mock.calls[0][0].changed.value).toBe(123);
      
      // Listener2 should fail validation (expecting string, got number)
      expect(listener2).not.toHaveBeenCalled();
      
      // Error handler should have been called for schema2 validation failure
      expect(handleError.mock.calls.length).toBeGreaterThan(0);

      await closeStreamer();
    });
  });

  describe("Channel Name Sorting (Delete Operations)", () => {
    it("should process channels with 'delete' in the name first", async () => {
      const executionOrder: string[] = [];
      const deleteListener = vi.fn().mockImplementation(() => {
        executionOrder.push("delete");
      });
      const createListener = vi.fn().mockImplementation(() => {
        executionOrder.push("create");
      });
      const updateListener = vi.fn().mockImplementation(() => {
        executionOrder.push("update");
      });
      const schema = z.object({ id: z.number() });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            { channel: "user_create", schema, onChange: createListener },
            { channel: "user_delete", schema, onChange: deleteListener },
            { channel: "user_update", schema, onChange: updateListener },
          ],
        },
      };

      // Frame contains all three channels in non-sorted order
      const frames = [
        createFrame({
          user_create: [{ id: 1 }],
          user_update: [{ id: 2 }],
          user_delete: [{ id: 3 }],
        }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => executionOrder.length).toBe(3);

      // Delete should be processed first due to sorting
      expect(executionOrder[0]).toBe("delete");
      // Then create and update in their original order
      expect(executionOrder[1]).toBe("create");
      expect(executionOrder[2]).toBe("update");

      await closeStreamer();
    });

    it("should handle multiple delete channels before non-delete channels", async () => {
      const executionOrder: string[] = [];
      const listeners: Record<string, Mock> = {};
      const channels = [
        "relationship_create",
        "relationship_delete",
        "user_delete",
        "user_update",
        "permission_delete",
        "permission_grant",
      ];

      const schema = z.object({ id: z.number() });
      const storeListeners: flux.StoreConfig<flux.Store>["labels"]["listeners"] = [];

      channels.forEach((channel) => {
        const listener = vi.fn().mockImplementation(() => {
          executionOrder.push(channel);
        });
        listeners[channel] = listener;
        storeListeners.push({ channel, schema, onChange: listener });
      });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: { listeners: storeListeners },
      };

      // Create frame with all channels
      const frameData: Record<string, unknown[]> = {};
      channels.forEach((channel, index) => {
        frameData[channel] = [{ id: index }];
      });

      const frames = [createFrame(frameData)];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => executionOrder.length).toBe(6);

      // All delete channels should come first
      const deleteChannels = executionOrder.slice(0, 3);
      const nonDeleteChannels = executionOrder.slice(3);

      expect(deleteChannels).toContain("relationship_delete");
      expect(deleteChannels).toContain("user_delete");
      expect(deleteChannels).toContain("permission_delete");

      expect(nonDeleteChannels).toContain("relationship_create");
      expect(nonDeleteChannels).toContain("user_update");
      expect(nonDeleteChannels).toContain("permission_grant");

      await closeStreamer();
    });

    it("should correctly handle relationship changes (delete then create)", async () => {
      const operations: Array<{ channel: string; data: unknown }> = [];
      const deleteListener = vi.fn().mockImplementation(({ changed }) => {
        operations.push({ channel: "relationship_delete", data: changed });
      });
      const createListener = vi.fn().mockImplementation(({ changed }) => {
        operations.push({ channel: "relationship_create", data: changed });
      });

      const relationshipSchema = z.object({
        parentId: z.number(),
        childId: z.number(),
        type: z.string(),
      });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: {
          listeners: [
            {
              channel: "relationship_create",
              schema: relationshipSchema,
              onChange: createListener,
            },
            {
              channel: "relationship_delete",
              schema: relationshipSchema,
              onChange: deleteListener,
            },
          ],
        },
      };

      // Simulate updating a relationship (delete old, create new)
      const frames = [
        createFrame({
          relationship_create: [{ parentId: 1, childId: 2, type: "updated" }],
          relationship_delete: [{ parentId: 1, childId: 2, type: "original" }],
        }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => operations.length).toBe(2);

      // Delete should happen first
      expect(operations[0].channel).toBe("relationship_delete");
      expect(operations[0].data).toEqual({
        parentId: 1,
        childId: 2,
        type: "original",
      });

      // Then create
      expect(operations[1].channel).toBe("relationship_create");
      expect(operations[1].data).toEqual({
        parentId: 1,
        childId: 2,
        type: "updated",
      });

      await closeStreamer();
    });

    it("should handle frames with only delete channels", async () => {
      const executionOrder: string[] = [];
      const schema = z.object({ id: z.number() });
      const listeners: flux.StoreConfig<flux.Store>["labels"]["listeners"] = [];

      ["user_delete", "role_delete", "permission_delete"].forEach((channel) => {
        const listener = vi.fn().mockImplementation(() => {
          executionOrder.push(channel);
        });
        listeners.push({ channel, schema, onChange: listener });
      });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: { listeners },
      };

      const frames = [
        createFrame({
          user_delete: [{ id: 1 }],
          role_delete: [{ id: 2 }],
          permission_delete: [{ id: 3 }],
        }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => executionOrder.length).toBe(3);

      // All are delete operations, so sorting won't change their order
      // They should all be present
      expect(executionOrder).toContain("user_delete");
      expect(executionOrder).toContain("role_delete");
      expect(executionOrder).toContain("permission_delete");

      await closeStreamer();
    });

    it("should handle frames with no delete channels", async () => {
      const executionOrder: string[] = [];
      const schema = z.object({ id: z.number() });
      const listeners: flux.StoreConfig<flux.Store>["labels"]["listeners"] = [];

      ["user_create", "role_update", "permission_grant"].forEach((channel) => {
        const listener = vi.fn().mockImplementation(() => {
          executionOrder.push(channel);
        });
        listeners.push({ channel, schema, onChange: listener });
      });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: { listeners },
      };

      const frames = [
        createFrame({
          user_create: [{ id: 1 }],
          role_update: [{ id: 2 }],
          permission_grant: [{ id: 3 }],
        }),
      ];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => executionOrder.length).toBe(3);

      // No delete operations, all should be present
      expect(executionOrder).toContain("user_create");
      expect(executionOrder).toContain("role_update");
      expect(executionOrder).toContain("permission_grant");

      await closeStreamer();
    });

    it("should handle 'delete' appearing in different positions of channel name", async () => {
      const executionOrder: string[] = [];
      const schema = z.object({ id: z.number() });
      const channels = [
        "create_user",
        "delete_user", // 'delete' at beginning
        "user_delete", // 'delete' at end
        "user_soft_delete", // 'delete' at end after underscore
        "undelete_user", // contains 'delete' but not a delete operation
        "update_user",
      ];

      const listeners: flux.StoreConfig<flux.Store>["labels"]["listeners"] = [];
      channels.forEach((channel) => {
        const listener = vi.fn().mockImplementation(() => {
          executionOrder.push(channel);
        });
        listeners.push({ channel, schema, onChange: listener });
      });

      const storeConfig: flux.StoreConfig<flux.Store> = {
        labels: { listeners },
      };

      const frameData: Record<string, unknown[]> = {};
      channels.forEach((channel, index) => {
        frameData[channel] = [{ id: index }];
      });

      const frames = [createFrame(frameData)];

      const closeStreamer = await flux.openStreamer(
        createStreamerArgs({
          storeConfig,
          openStreamer: createFrameStreamer(frames),
        }),
      );

      await expect.poll(() => executionOrder.length).toBe(6);

      // Channels with 'delete' anywhere should come first
      const firstFour = executionOrder.slice(0, 4);
      expect(firstFour).toContain("delete_user");
      expect(firstFour).toContain("user_delete");
      expect(firstFour).toContain("user_soft_delete");
      expect(firstFour).toContain("undelete_user");

      // Non-delete channels should come last
      const lastTwo = executionOrder.slice(4);
      expect(lastTwo).toContain("create_user");
      expect(lastTwo).toContain("update_user");

      await closeStreamer();
    });
  });
});
