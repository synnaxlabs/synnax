// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot, type ranger, type Synnax } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import {
  dynamicWindow,
  scheduler,
  staticWindow,
  TimeRange,
  TimeSpan,
  uuid,
} from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, selectRange } from "@/testutil";

const client = createTestClient();

const createRange = async (): Promise<ranger.Range> =>
  await createTestRange(client, { name: uniqueName("range") });

const preloadedFor = (keys: string[]) => ({
  [Session.Range.SLICE_NAME]: Session.Range.sliceStateZ.parse({
    ranges: keys.map((key) => ({ variant: "persisted", key })),
  }),
});

/** A range that Desktop 1.0 kept only in the Console, as version 0 stored it. */
interface Local {
  variant: "static";
  key: string;
  name: string;
  timeRange: { start: number; end: number };
}

const createLocal = (): Local => ({
  variant: "static",
  key: uuid.create(),
  name: uniqueName("local"),
  timeRange: { start: 1, end: 2 },
});

const windowOf = (local: Local): ranger.Ref => ({
  variant: "window",
  window: staticWindow(new TimeRange(local.timeRange)),
});

const createPlot = async (ranges: ranger.Ref[]): Promise<lineplot.LinePlot> => {
  const project = await client.projects.create({ name: uniqueName("project") });
  return await client.lineplots.create({
    project: project.key,
    name: uniqueName("plot"),
    ranges,
  });
};

describe("Range.SYNCHRONIZERS", () => {
  it("should remove a favorited range from the slice when it is deleted remotely", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor([range.key]),
    });
    renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
      wrapper,
    });
    expect(selectRange(store.getState(), range.key)).toBeDefined();
    await client.ranges.delete(range.key);
    await waitFor(() => {
      expect(selectRange(store.getState(), range.key)).toBeUndefined();
    });
  });

  it("should remove a favorited range the Core no longer holds on connect", async () => {
    const range = await createRange();
    const gone = uuid.create();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor([range.key, gone]),
    });
    renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
      wrapper,
    });
    await waitFor(() => {
      expect(selectRange(store.getState(), gone)).toBeUndefined();
    });
    expect(selectRange(store.getState(), range.key)).toBeDefined();
  });

  describe("static range keys", () => {
    it("should replace keys of static ranges with their windows", async () => {
      const local = createLocal();
      const saved = await createRange();
      const savedRef: ranger.Ref = { variant: "key", key: saved.key };
      const rolling: ranger.Ref = {
        variant: "window",
        window: dynamicWindow(TimeSpan.minutes(1)),
      };
      const localRef: ranger.Ref = { variant: "key", key: local.key };
      const mixed = await createPlot([savedRef, localRef, rolling]);
      const shown = await createPlot([windowOf(local), localRef]);
      const untouched = await createPlot([savedRef]);
      const send = vi.spyOn(client.lineplots, "send");
      const { wrapper, store } = await createConsoleWrapper({
        client,
        preloadedState: {
          [Session.Range.SLICE_NAME]: Session.Range.sliceStateZ.parse({
            ranges: [local],
          }),
        },
      });
      renderHook(() => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS), {
        wrapper,
      });
      await waitFor(() =>
        expect(store.getState()[Session.Range.SLICE_NAME].staticWindows).toEqual({}),
      );
      const reader = createTestClient();
      expect((await reader.lineplots.retrieve(mixed.key)).ranges).toEqual([
        savedRef,
        rolling,
        windowOf(local),
      ]);
      expect((await reader.lineplots.retrieve(shown.key)).ranges).toEqual([
        windowOf(local),
      ]);
      expect((await reader.lineplots.retrieve(untouched.key)).ranges).toEqual([
        savedRef,
      ]);
      expect(send).not.toHaveBeenCalledWith(untouched.key, expect.anything());
      expect(client.lineplots.hasUndo(mixed.key)).toBe(false);
      send.mockRestore();
    });

    it("should leave line plots alone once the session rewrote them", async () => {
      const local = createLocal();
      const plot = await createPlot([{ variant: "key", key: local.key }]);
      const { wrapper } = await createConsoleWrapper({
        client,
        preloadedState: {
          [Session.Range.SLICE_NAME]: Session.Range.sliceStateZ.parse({
            ranges: [local],
            staticKeysRewritten: true,
          }),
        },
      });
      const { result } = renderHook(
        () => Session.Synchronizer.use(Session.Range.SYNCHRONIZERS),
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBe(true));
      const { ranges } = await createTestClient().lineplots.retrieve(plot.key);
      expect(ranges).toEqual([{ variant: "key", key: local.key }]);
    });

    const mount = async (local: Local, as: Synnax = client) => {
      const { wrapper, store } = await createConsoleWrapper({
        client: as,
        preloadedState: {
          [Session.Range.SLICE_NAME]: Session.Range.sliceStateZ.parse({
            ranges: [local],
          }),
        },
      });
      const hook = renderHook(
        () => ({
          verified: Session.Synchronizer.use(Session.Range.SYNCHRONIZERS),
          statuses: Status.useNotifications().statuses,
        }),
        { wrapper },
      );
      const rewritten = () =>
        Object.keys(store.getState()[Session.Range.SLICE_NAME].staticWindows).length ===
        0;
      return { ...hook, rewritten };
    };

    const mountWithLocal = async (as: Synnax = client) => {
      const local = createLocal();
      const plot = await createPlot([{ variant: "key", key: local.key }]);
      return { ...(await mount(local, as)), plot, local };
    };

    it("should report a failed send and try the rewrite again", async () => {
      const error = vi.spyOn(console, "error").mockImplementation(() => {});
      const send = vi
        .spyOn(client.lineplots, "send")
        .mockRejectedValue(new Error("send failed"));
      try {
        const local = createLocal();
        await createPlot([{ variant: "key", key: local.key }]);
        await createPlot([{ variant: "key", key: local.key }]);
        const { result, rewritten } = await mount(local);
        await waitFor(() => expect(result.current.verified).toBe(true));
        await waitFor(() => expect(send).toHaveBeenCalledTimes(2));
        expect(result.current.statuses).toEqual([
          expect.objectContaining({
            variant: "error",
            name: "Line plot",
            message: "Failed to update static ranges",
            description: "send failed",
          }),
        ]);
        expect(rewritten()).toBe(false);
      } finally {
        send.mockRestore();
        error.mockRestore();
      }
    });

    it("should skip line plots the user cannot update and try again later", async () => {
      const viewer = await createTestClientWithRole(client, "Viewer");
      const { result, plot, local, rewritten } = await mountWithLocal(viewer);
      await waitFor(() => expect(result.current.verified).toBe(true));
      expect(rewritten()).toBe(false);
      expect(result.current.statuses).toEqual([]);
      const { ranges } = await createTestClient().lineplots.retrieve(plot.key);
      expect(ranges).toEqual([{ variant: "key", key: local.key }]);
    });

    it("should mark the rewrite done when a line plot is deleted first", async () => {
      const local = createLocal();
      const plot = await createPlot([{ variant: "key", key: local.key }]);
      const retrieve = client.lineplots.retrieve.bind(client.lineplots);
      const spy = vi
        .spyOn(client.lineplots, "retrieve")
        .mockImplementationOnce(async (params) => {
          const plots = await retrieve(params);
          await createTestClient().lineplots.delete(plot.key);
          return plots;
        });
      try {
        const { result, rewritten } = await mount(local);
        await waitFor(() => expect(rewritten()).toBe(true));
        expect(result.current.statuses).toEqual([]);
      } finally {
        spy.mockRestore();
      }
    });

    it("should not mark the rewrite done when its pass ends", async () => {
      const { promise: gate, resolve: release } = Promise.withResolvers<void>();
      const send = vi.spyOn(client.lineplots, "send").mockReturnValue(gate);
      try {
        const { unmount, rewritten } = await mountWithLocal();
        await waitFor(() => expect(send).toHaveBeenCalled());
        unmount();
        release();
        await scheduler.nextTask();
        expect(rewritten()).toBe(false);
      } finally {
        send.mockRestore();
      }
    });
  });
});
