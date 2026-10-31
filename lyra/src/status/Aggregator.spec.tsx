// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Status } from "@/status";

const quiet: Status.Log = () => {};

const wrapper = ({ children }: PropsWithChildren) => (
  <Status.Aggregator log={quiet}>{children}</Status.Aggregator>
);

const renderAggregator = () =>
  renderHook(
    () => ({ add: Status.useAdder(), notifications: Status.useNotifications() }),
    { wrapper },
  );

const problem = (name: string = "p1"): Status.Crude => ({
  key: "task:1",
  variant: "error",
  message: "Failed to read",
  problem: name,
});

describe("Aggregator", () => {
  it("should correctly render an aggregator", () => {
    const { getByText } = render(
      <Status.Aggregator>
        <p>Test</p>
      </Status.Aggregator>,
    );
    expect(getByText("Test")).toBeTruthy();
  });

  describe("add", () => {
    it("should allow the caller to add a status", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].message).toEqual("Test");
    });
    it("should allow the caller to silence a notification", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const { id } = result.current.statuses.statuses[0];
      act(() => {
        result.current.statuses.silence(id);
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });
    it("should remove notifications after an expiration threshold", async () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications({
            expiration: TimeSpan.milliseconds(1),
            poll: TimeSpan.milliseconds(1),
          }),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      await waitFor(() => expect(result.current.statuses.statuses).toHaveLength(0));
    });

    describe("expiration", () => {
      afterEach(() => {
        vi.useRealTimers();
      });

      it("should count from when it gets a status, not from the status time", () => {
        vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
        const { result } = renderHook(
          () => ({
            add: Status.useAdder(),
            statuses: Status.useNotifications(),
          }),
          { wrapper },
        );
        act(() => {
          result.current.add({
            variant: "warning",
            message: "Driver not running",
            time: TimeStamp.now().sub(TimeSpan.minutes(1)),
          });
        });
        expect(result.current.statuses.statuses).toHaveLength(1);
        act(() => void vi.advanceTimersByTime(TimeSpan.seconds(6).milliseconds));
        expect(result.current.statuses.statuses).toHaveLength(1);
        act(() => void vi.advanceTimersByTime(TimeSpan.seconds(2).milliseconds));
        expect(result.current.statuses.statuses).toHaveLength(0);
      });
    });
  });

  describe("problems", () => {
    afterEach(() => {
      vi.useRealTimers();
    });

    it("should join a status of the same problem to its notification", () => {
      const { result } = renderAggregator();
      act(() => result.current.add(problem()));
      act(() => result.current.add({ ...problem(), message: "Still failing" }));
      expect(result.current.notifications.statuses).toMatchObject([
        { message: "Still failing", count: 2 },
      ]);
    });

    it("should not show a notification again when a status joins it", () => {
      vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
      const { result } = renderAggregator();
      act(() => result.current.add(problem()));
      act(() => void vi.advanceTimersByTime(TimeSpan.seconds(8).milliseconds));
      act(() => result.current.add(problem()));
      expect(result.current.notifications.statuses).toHaveLength(0);
    });

    it("should not join a status that names no problem", () => {
      const { result } = renderAggregator();
      const reply: Status.Crude = { ...problem(), problem: undefined };
      act(() => result.current.add(reply));
      act(() => result.current.add(reply));
      expect(result.current.notifications.statuses).toMatchObject([{ count: 1 }]);
    });

    it("should show the next problem on a key", () => {
      const { result } = renderAggregator();
      act(() => result.current.add(problem()));
      act(() => result.current.add(problem("p2")));
      expect(result.current.notifications.statuses).toMatchObject([{ count: 1 }]);
    });

    it("should keep the notification of a problem beside another status", () => {
      const { result } = renderAggregator();
      act(() => result.current.add(problem()));
      act(() =>
        result.current.add({ ...problem(), problem: undefined, message: "No reply" }),
      );
      act(() => result.current.add(problem()));
      expect(result.current.notifications.statuses).toMatchObject([
        { message: "No reply", count: 1 },
        { message: "Failed to read", count: 2 },
      ]);
    });

    it("should add the counts of identical notifications on different keys", () => {
      const { result } = renderAggregator();
      act(() => result.current.add(problem()));
      act(() => result.current.add(problem()));
      act(() => result.current.add({ ...problem(), key: "task:2" }));
      expect(result.current.notifications.statuses).toMatchObject([{ count: 3 }]);
    });

    describe("history", () => {
      const renderSmall = () =>
        renderHook(
          () => ({ add: Status.useAdder(), notifications: Status.useNotifications() }),
          {
            wrapper: ({ children }: PropsWithChildren) => (
              <Status.Aggregator log={quiet} maxHistory={3}>
                {children}
              </Status.Aggregator>
            ),
          },
        );
      const expire = () =>
        act(() => void vi.advanceTimersByTime(TimeSpan.seconds(8).milliseconds));

      it("should keep a problem when other statuses fill the history", () => {
        vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
        const { result } = renderSmall();
        act(() => result.current.add(problem()));
        expire();
        for (let i = 0; i < 4; i++)
          act(() => result.current.add({ variant: "info", message: `Saved ${i}` }));
        expire();
        act(() => result.current.add(problem()));
        expect(result.current.notifications.statuses).toHaveLength(0);
      });

      it("should drop the oldest statuses that are not problems", () => {
        const { result } = renderSmall();
        for (let i = 0; i < 5; i++)
          act(() => result.current.add({ variant: "info", message: `Saved ${i}` }));
        expect(result.current.notifications.statuses.map((s) => s.message)).toEqual([
          "Saved 4",
          "Saved 3",
          "Saved 2",
        ]);
      });

      it("should keep every problem and a new status when problems fill it", () => {
        const { result } = renderSmall();
        for (let i = 0; i < 5; i++)
          act(() =>
            result.current.add({ ...problem(), key: `task:${i}`, message: `${i}` }),
          );
        act(() => result.current.add({ variant: "error", message: "Failed to start" }));
        expect(result.current.notifications.statuses.map((s) => s.message)).toEqual([
          "Failed to start",
          "4",
          "3",
          "2",
          "1",
          "0",
        ]);
      });
    });
  });

  describe("grouping", () => {
    it("should fold identical statuses into one", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({ variant: "error", message: "Test" });
      });
      act(() => {
        result.current.add({ variant: "error", message: "Test" });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].count).toBe(2);
    });

    it("should keep statuses with distinct names separate", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({ variant: "error", message: "Discrepancy", name: "1" });
      });
      act(() => {
        result.current.add({ variant: "error", message: "Discrepancy", name: "2" });
      });
      act(() => {
        result.current.add({ variant: "error", message: "Discrepancy", name: "3" });
      });
      const { statuses } = result.current.statuses;
      expect(statuses).toHaveLength(3);
      expect(statuses.map(({ name }) => name)).toEqual(["3", "2", "1"]);
      statuses.forEach(({ count }) => expect(count).toBe(1));
    });

    it("should keep statuses with distinct descriptions separate", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "error",
          message: "Failed to start task",
          description: "The device is unreachable.",
        });
      });
      act(() => {
        result.current.add({
          variant: "error",
          message: "Failed to start task",
          description: "The channel no longer exists.",
        });
      });
      const { statuses } = result.current.statuses;
      expect(statuses.map(({ description }) => description)).toEqual([
        "The channel no longer exists.",
        "The device is unreachable.",
      ]);
      statuses.forEach(({ count }) => expect(count).toBe(1));
    });
  });

  describe("silence", () => {
    it("should silence a notification", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const { id } = result.current.statuses.statuses[0];
      act(() => {
        result.current.statuses.silence(id);
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });

    it("should silence all identical notifications at once", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Info",
        });
      });
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Info",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].count).toBe(2);

      const { id } = result.current.statuses.statuses[0];
      act(() => {
        result.current.statuses.silence(id);
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });

    it("should not silence a notification with a different name", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({ variant: "error", message: "Discrepancy", name: "1" });
      });
      act(() => {
        result.current.add({ variant: "error", message: "Discrepancy", name: "2" });
      });
      expect(result.current.statuses.statuses).toHaveLength(2);
      const { id } = result.current.statuses.statuses[0];
      act(() => {
        result.current.statuses.silence(id);
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].name).toEqual("1");
    });

    it("should not silence a notification with a different description", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "error",
          message: "Failed to start task",
          description: "The device is unreachable.",
        });
      });
      act(() => {
        result.current.add({
          variant: "error",
          message: "Failed to start task",
          description: "The channel no longer exists.",
        });
      });
      const { id } = result.current.statuses.statuses[0];
      act(() => {
        result.current.statuses.silence(id);
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].description).toEqual(
        "The device is unreachable.",
      );
    });
    describe("problems", () => {
      const firstID = (result: {
        current: { notifications: Status.UseNotificationsReturn };
      }): string => result.current.notifications.statuses[0].id;

      it("should keep a closed problem closed", () => {
        const { result } = renderAggregator();
        act(() => result.current.add(problem("p1")));
        act(() => result.current.notifications.silence(firstID(result)));
        act(() => result.current.add(problem("p1")));
        expect(result.current.notifications.statuses).toHaveLength(0);
      });

      it("should show the next problem on the key of a closed one", () => {
        const { result } = renderAggregator();
        act(() => result.current.add(problem("p1")));
        act(() => result.current.notifications.silence(firstID(result)));
        act(() => result.current.add(problem("p2")));
        expect(result.current.notifications.statuses).toHaveLength(1);
      });

      it("should show a status that is not a problem after a closed one", () => {
        const { result } = renderAggregator();
        const good: Status.Crude = { key: "rack:1", variant: "info", message: "Ready" };
        act(() => result.current.add(good));
        act(() => result.current.notifications.silence(firstID(result)));
        act(() => result.current.add(good));
        expect(result.current.notifications.statuses).toHaveLength(1);
      });
    });
  });

  describe("silenceAll", () => {
    it("should silence every active notification", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({ variant: "success", message: "First" });
      });
      act(() => {
        result.current.add({ variant: "error", message: "Second" });
      });
      expect(result.current.statuses.statuses).toHaveLength(2);
      act(() => {
        result.current.statuses.silenceAll();
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });

    it("should not silence a notification added afterward", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({ variant: "success", message: "First" });
      });
      act(() => {
        result.current.statuses.silenceAll();
      });
      act(() => {
        result.current.add({ variant: "info", message: "Second" });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].message).toEqual("Second");
    });
  });

  describe("handleError", () => {
    it("should put the heading in the message and the error in the description", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add(new Error("Test"), "Failed to save range");
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const [stat] = result.current.statuses.statuses;
      expect(stat.name).toEqual("");
      expect(stat.message).toEqual("Failed to save range");
      expect(stat.description).toEqual("Test");
    });
    it("should allow the caller to name what failed", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add(new Error("Disk is full"), {
          message: "Failed to export",
          name: "Main schematic",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const [stat] = result.current.statuses.statuses;
      expect(stat.name).toEqual("Main schematic");
      expect(stat.message).toEqual("Failed to export");
      expect(stat.description).toEqual("Disk is full");
    });
    it("should allow the caller to pass in an async function", async () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      await act(async () => {
        await new Promise<void>((resolve) => {
          result.current.add(async () => {
            resolve();
            throw new Error("Test");
          }, "Failed to save range");
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const [stat] = result.current.statuses.statuses;
      expect(stat.message).toEqual("Failed to save range");
      expect(stat.description).toEqual("Test");
    });
    it("should print the error through the aggregator's log", () => {
      const log = vi.fn<Status.Log>();
      const { result } = renderHook(Status.useErrorHandler, {
        wrapper: ({ children }) => (
          <Status.Aggregator log={log}>{children}</Status.Aggregator>
        ),
      });
      act(() => result.current(new Error("Test"), "Failed to save range"));
      expect(log).toHaveBeenCalledExactlyOnceWith(
        expect.stringContaining("ERROR: Failed to save range"),
      );
    });
    describe("console.error", () => {
      afterEach(() => {
        vi.restoreAllMocks();
      });
      it("should print through the current console.error by default", () => {
        const { result } = renderHook(Status.useErrorHandler, {
          wrapper: Status.Aggregator,
        });
        const error = vi.spyOn(console, "error").mockImplementation(() => {});
        act(() => result.current(new Error("Test"), "Failed to save range"));
        expect(error).toHaveBeenCalledExactlyOnceWith(
          expect.stringContaining("ERROR: Failed to save range"),
        );
      });
      it("should keep the handler when console.error changes", () => {
        const { result, rerender } = renderHook(Status.useErrorHandler, {
          wrapper: Status.Aggregator,
        });
        const before = result.current;
        vi.spyOn(console, "error").mockImplementation(() => {});
        rerender();
        expect(result.current).toBe(before);
      });
    });
    it("should neither add nor print a skipped error", () => {
      const log = vi.fn<Status.Log>();
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        {
          wrapper: ({ children }) => (
            <Status.Aggregator log={log}>{children}</Status.Aggregator>
          ),
        },
      );
      const error = new Error("Test");
      act(() =>
        result.current.add(error, "Failed to save range", {
          matches: (e) => e === error,
        }),
      );
      expect(result.current.statuses.statuses).toHaveLength(0);
      expect(log).not.toHaveBeenCalled();
    });
    it("should not add a status if no error is thrown", async () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      await act(async () => {
        await new Promise<void>((resolve) => {
          result.current.add(async () => {
            resolve();
          }, "Failed to save range");
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });
  });
});
