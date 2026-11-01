// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { type Notifications } from "@/platform/notifications";
import { renderForward, settleWrites } from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createTaskKey = (): status.Key => task.statusKey(uuid.create());

interface Written {
  key: status.Key;
  message: string;
}

// One write per status: the Core keeps only the last status of a key in one write.
const write = async (
  ...statuses: Array<Partial<status.New> & Pick<status.New, "variant">>
): Promise<Written[]> => {
  const written: Written[] = [];
  for (const s of statuses) {
    const { key, message } = await client.statuses.set({
      key: createTaskKey(),
      name: "",
      message: uniqueName("message"),
      time: TimeStamp.now(),
      ...s,
    });
    written.push({ key, message });
  }
  return written;
};

const createAlarm = (): Partial<status.New> & Pick<status.New, "variant"> => ({
  key: uniqueName("alarm"),
  variant: "info",
});

const REPLY: Notifications.Rule = { kind: () => "reply" };
const STATE: Notifications.Rule = { kind: () => "state" };
const DELAYED: Notifications.Rule = { kind: () => "state", warningsDelayed: true };

describe("Notifications.useForward", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should notify a status that no rule owns at once", async () => {
    await renderForward(client, {});
    const [alarm] = await write(createAlarm());
    expect(await screen.findByText(alarm.message)).toBeTruthy();
  });

  it("should notify a status with an old time", async () => {
    await renderForward(client, {});
    const [alarm] = await write({
      ...createAlarm(),
      time: TimeStamp.now().sub(TimeSpan.minutes(1)),
    });
    expect(await screen.findByText(alarm.message)).toBeTruthy();
  });

  describe("reply", () => {
    it("should notify a reply at once, without checking use", async () => {
      const inUse = vi.fn(async () => false);
      await renderForward(client, { task: { ...REPLY, inUse, warningsDelayed: true } });
      const [reply] = await write({ variant: "warning" });
      expect(await screen.findByText(reply.message)).toBeTruthy();
      expect(inUse).not.toHaveBeenCalled();
    });

    it("should not notify a reply that reports success", async () => {
      await renderForward(client, { task: REPLY });
      const [success] = await write({ variant: "success" });
      await settleWrites(client);
      expect(screen.queryByText(success.message)).toBeNull();
    });
  });

  describe("state", () => {
    it("should notify a problem only when its thing is in use", async () => {
      const used = createTaskKey();
      const inUse = vi.fn(
        async ({ key }: Notifications.InUseParams) => task.statusKey(key) === used,
      );
      await renderForward(client, { task: { ...STATE, inUse } });
      const [unused, inUseError] = await write(
        { variant: "error" },
        { key: used, variant: "error" },
      );
      await screen.findByText(inUseError.message);
      expect(screen.queryByText(unused.message)).toBeNull();
    });

    it("should not notify a state that is not a warning or an error", async () => {
      await renderForward(client, { task: STATE });
      const [info] = await write({ variant: "info" });
      await settleWrites(client);
      expect(screen.queryByText(info.message)).toBeNull();
    });

    it("should check use only when a problem starts on its key", async () => {
      const key = createTaskKey();
      const inUse = vi.fn(async (_: Notifications.InUseParams) => true);
      await renderForward(client, { task: { ...STATE, inUse } });
      const written = await write(
        { key, variant: "error" },
        { key, variant: "error" },
        { key, variant: "success" },
        { key, variant: "success" },
        { key, variant: "error" },
      );
      await screen.findByText(written[4].message);
      // Specs that share the Core write task statuses too.
      const calls = inUse.mock.calls.filter(
        ([args]) => task.statusKey(args.key) === key,
      );
      expect(calls).toHaveLength(2);
    });

    it("should drop the pending check of a deleted status", async () => {
      const key = createTaskKey();
      let resolve: ((inUse: boolean) => void) | undefined;
      const inUse = vi.fn(async ({ key: k }: Notifications.InUseParams) => {
        if (task.statusKey(k) !== key) return false;
        return await new Promise<boolean>((r) => (resolve = r));
      });
      await renderForward(client, { task: { ...STATE, inUse } });
      const [error] = await write({ key, variant: "error" });
      await vi.waitFor(() => expect(resolve).toBeDefined());
      await client.statuses.delete(key);
      act(() => resolve?.(true));
      await settleWrites(client);
      expect(screen.queryByText(error.message)).toBeNull();
    });

    it("should not notify a deleted status whose check fails", async () => {
      const key = createTaskKey();
      let reject: ((exc: Error) => void) | undefined;
      const inUse = vi.fn(async ({ key: k }: Notifications.InUseParams) => {
        if (task.statusKey(k) !== key) return false;
        return await new Promise<boolean>((_, r) => (reject = r));
      });
      await renderForward(client, { task: { ...STATE, inUse } });
      const name = uniqueName("task");
      await write({ key, variant: "error", name });
      await vi.waitFor(() => expect(reject).toBeDefined());
      await client.statuses.delete(key);
      act(() => reject?.(new Error("The Core is unreachable.")));
      await settleWrites(client);
      expect(screen.queryByText(name)).toBeNull();
    });

    it("should notify a status whose check fails and print the error", async () => {
      const key = createTaskKey();
      const exc = new Error("The Core is unreachable.");
      const inUse = vi.fn(async ({ key: k }: Notifications.InUseParams) => {
        if (task.statusKey(k) === key) throw exc;
        return false;
      });
      const printError = vi.spyOn(console, "error").mockImplementation(() => {});
      await renderForward(client, { task: { ...STATE, inUse } });
      const name = uniqueName("task");
      await write({ key, variant: "error", name, message: "Failed to start task" });
      expect(await screen.findByText("Failed to start task")).toBeTruthy();
      expect(printError).toHaveBeenCalledWith(
        `failed to check if the thing of ${key} is in use`,
        exc,
      );
      printError.mockRestore();
    });
  });

  describe("delayed warnings", () => {
    const fakeTimers = () =>
      vi.useFakeTimers({
        toFake: ["setTimeout", "clearTimeout"],
        shouldAdvanceTime: true,
      });

    it("should notify a warning only when its problem is open after 5 s", async () => {
      fakeTimers();
      await renderForward(client, { task: DELAYED });
      const [warning] = await write({ variant: "warning" });
      await settleWrites(client);
      expect(screen.queryByText(warning.message)).toBeNull();
      act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
      expect(screen.getByText(warning.message)).toBeTruthy();
    });

    it("should not notify a warning whose problem closes within 5 s", async () => {
      fakeTimers();
      await renderForward(client, { task: DELAYED });
      const key = createTaskKey();
      const [warning] = await write(
        { key, variant: "warning" },
        { key, variant: "success" },
      );
      await settleWrites(client);
      act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
      expect(screen.queryByText(warning.message)).toBeNull();
    });

    it("should notify an error at once", async () => {
      fakeTimers();
      await renderForward(client, { task: DELAYED });
      const [error] = await write({ variant: "error" });
      await settleWrites(client);
      expect(screen.getByText(error.message)).toBeTruthy();
    });

    it("should not notify a held warning after a newer reply", async () => {
      fakeTimers();
      const rule: Notifications.Rule = {
        kind: (stat) => (status.detailsOf(stat)?.cmd == null ? "state" : "reply"),
        warningsDelayed: true,
      };
      await renderForward(client, { task: rule });
      const key = createTaskKey();
      const [warning] = await write({ key, variant: "warning" });
      const { message: reply } = await client.statuses.set({
        key,
        name: "",
        variant: "error",
        message: uniqueName("reply"),
        time: TimeStamp.now(),
        details: { cmd: "start" },
      });
      await screen.findByText(reply);
      act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
      expect(screen.queryByText(warning.message)).toBeNull();
      expect(screen.getByText(reply)).toBeTruthy();
    });

    it("should not notify a held warning whose status is deleted", async () => {
      fakeTimers();
      await renderForward(client, { task: DELAYED });
      const key = createTaskKey();
      const [warning] = await write({ key, variant: "warning" });
      await settleWrites(client);
      await client.statuses.delete(key);
      act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
      expect(screen.queryByText(warning.message)).toBeNull();
    });
  });
});
