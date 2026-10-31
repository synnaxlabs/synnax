// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, status, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { type Notifications } from "@/platform/notifications";
import {
  close,
  createRackKey,
  findNotification,
  renderForward,
  settleWrites,
} from "@/platform/notifications/testutil";
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

const fakeTimers = () =>
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true });

const advance = (span: TimeSpan): void =>
  act(() => void vi.advanceTimersByTime(span.milliseconds));

const REPLY: Notifications.Rule = { kind: () => "reply" };
const usedOnly = (used: status.Key): Notifications.Rule => ({
  kind: () => "state",
  inUse: async ({ key }) => task.statusKey(key) === used,
});
const keysOf = (problems: Notifications.Problem[]): status.Key[] =>
  problems.map(({ status: { key } }) => key);
const STATE: Notifications.Rule = { kind: () => "state" };
const DELAYED: Notifications.Rule = { kind: () => "state", warningsDelayed: true };
const commanded = (rule: Partial<Notifications.Rule> = {}): Notifications.Rule => ({
  kind: (stat) => (status.detailsOf(stat)?.cmd == null ? "state" : "reply"),
  warningsDelayed: true,
  ...rule,
});

/** @returns the message of an error that answers a command on the key. */
const writeReply = async (key: status.Key): Promise<string> =>
  (
    await client.statuses.set({
      key,
      name: "",
      variant: "error",
      message: uniqueName("reply"),
      time: TimeStamp.now(),
      details: { cmd: "start" },
    })
  ).message;

describe("Notifications.Provider", () => {
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
      const inUse = vi.fn(async (_: Notifications.InUseParams) => false);
      await renderForward(client, { task: { ...REPLY, inUse, warningsDelayed: true } });
      const [reply] = await write({ variant: "warning" });
      expect(await screen.findByText(reply.message)).toBeTruthy();
      // The forwarder checks the replies that are open at load.
      const calls = inUse.mock.calls.filter(
        ([args]) => task.statusKey(args.key) === reply.key,
      );
      expect(calls).toHaveLength(0);
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

    it("should check use only when a new problem starts on its key", async () => {
      const key = createTaskKey();
      const inUse = vi.fn(async (_: Notifications.InUseParams) => true);
      await renderForward(client, { task: { ...STATE, inUse } });
      // The Core keeps the since of a problem that comes back within 30 s.
      await write(
        { key, variant: "error" },
        { key, variant: "error" },
        { key, variant: "success" },
        { key, variant: "error" },
      );
      // A delete reaches the forwarder at once, before the writes the Core streams.
      await settleWrites(client);
      await client.statuses.delete(key);
      const [next] = await write({ key, variant: "error" });
      await screen.findByText(next.message);
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

  describe("use changes", () => {
    interface Controlled {
      rule: Notifications.Rule;
      /** Reports a change of use to the forwarder. */
      change: () => void;
    }

    const createControlled = (inUse: Notifications.Rule["inUse"]): Controlled => {
      let handler: (() => void) | undefined;
      return {
        rule: {
          ...STATE,
          inUse,
          onUseChange: (_, h) => {
            handler = h;
            return () => (handler = undefined);
          },
        },
        change: () => {
          if (handler == null) throw new Error("the forwarder must listen for changes");
          handler();
        },
      };
    };

    /** @returns a use check that ends by a resolver on the key, and false elsewhere. */
    const createPendingInUse =
      (
        key: status.Key,
        resolvers: Array<(inUse: boolean) => void>,
      ): Notifications.Rule["inUse"] =>
      async ({ key: k }) => {
        if (task.statusKey(k) !== key) return false;
        return await new Promise<boolean>((r) => resolvers.push(r));
      };

    it("should list, not notify, a problem whose thing comes into use", async () => {
      const key = createTaskKey();
      const resolvers: Array<(inUse: boolean) => void> = [];
      const { rule, change } = createControlled(createPendingInUse(key, resolvers));
      const { result } = await renderForward(client, { task: rule });
      const [error] = await write({ key, variant: "error" });
      await vi.waitFor(() => expect(resolvers).toHaveLength(1));
      change();
      change();
      await vi.waitFor(() => expect(resolvers).toHaveLength(2));
      // The first check ends last, but it still decides the notification.
      act(() => resolvers[1](true));
      act(() => resolvers[0](false));
      await waitFor(() => expect(keysOf(result.current)).toContain(key));
      await settleWrites(client);
      expect(screen.queryByText(error.message)).toBeNull();
    });

    it("should not check the use of a closed problem until it reopens", async () => {
      const key = createTaskKey();
      const checked = vi.fn();
      const { rule, change } = createControlled(async ({ key: k }) => {
        if (task.statusKey(k) !== key) return false;
        checked();
        return true;
      });
      await renderForward(client, { task: rule });
      const local: status.Status = {
        key,
        name: "",
        variant: "error",
        message: uniqueName("message"),
        description: "",
        time: TimeStamp.now(),
      };
      client.statuses.setLocal(local);
      await vi.waitFor(() => expect(checked).toHaveBeenCalledTimes(1));
      client.statuses.setLocal({ ...local, variant: "success", time: TimeStamp.now() });
      change();
      await settleWrites(client);
      expect(checked).toHaveBeenCalledTimes(1);
      client.statuses.setLocal({ ...local, time: TimeStamp.now() });
      await vi.waitFor(() => expect(checked).toHaveBeenCalledTimes(2));
    });

    it("should keep only the result of the latest check", async () => {
      const key = createTaskKey();
      const resolvers: Array<(inUse: boolean) => void> = [];
      const { rule, change } = createControlled(createPendingInUse(key, resolvers));
      const { result } = await renderForward(client, { task: rule });
      await write({ key, variant: "error" });
      await vi.waitFor(() => expect(resolvers).toHaveLength(1));
      act(() => resolvers[0](true));
      await waitFor(() => expect(keysOf(result.current)).toContain(key));
      change();
      await vi.waitFor(() => expect(resolvers).toHaveLength(2));
      change();
      await vi.waitFor(() => expect(resolvers).toHaveLength(3));
      act(() => resolvers[2](false));
      await waitFor(() => expect(keysOf(result.current)).not.toContain(key));
      act(() => resolvers[1](true));
      await settleWrites(client);
      expect(keysOf(result.current)).not.toContain(key);
    });

    it("should notify a held warning when an older check ends last", async () => {
      vi.useFakeTimers({
        toFake: ["setTimeout", "clearTimeout"],
        shouldAdvanceTime: true,
      });
      const key = createTaskKey();
      const resolvers: Array<(inUse: boolean) => void> = [];
      const { rule, change } = createControlled(createPendingInUse(key, resolvers));
      await renderForward(client, { task: { ...rule, warningsDelayed: true } });
      const [warning] = await write({ key, variant: "warning" });
      await vi.waitFor(() => expect(resolvers).toHaveLength(1));
      act(() => resolvers[0](true));
      change();
      await vi.waitFor(() => expect(resolvers).toHaveLength(2));
      change();
      await vi.waitFor(() => expect(resolvers).toHaveLength(3));
      act(() => resolvers[2](true));
      act(() => resolvers[1](false));
      await settleWrites(client);
      act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
      expect(screen.getByText(warning.message)).toBeTruthy();
    });

    it("should drop a problem whose thing leaves use", async () => {
      const key = createTaskKey();
      const used = new Set([key]);
      const { rule, change } = createControlled(async ({ key: k }) =>
        used.has(task.statusKey(k)),
      );
      // Rules for every type keep the problems of other specs out of the list.
      const { result } = await renderForward(client, {
        channel: rule,
        device: rule,
        rack: rule,
        task: rule,
      });
      await write({ key, variant: "error" });
      await waitFor(() => expect(keysOf(result.current)).toContain(key));
      used.delete(key);
      change();
      await waitFor(() => expect(keysOf(result.current)).not.toContain(key));
    });
  });

  describe("problems", () => {
    it("should notify a problem once and count its later writes", async () => {
      await renderForward(client, { task: STATE });
      const key = createTaskKey();
      const message = uniqueName("message");
      await write(
        { key, variant: "error", message },
        { key, variant: "error", message },
      );
      await settleWrites(client);
      expect(within(await findNotification(message)).getByText("x2")).toBeTruthy();
    });

    it("should keep a closed problem closed", async () => {
      await renderForward(client, { task: STATE });
      const key = createTaskKey();
      const [first] = await write({ key, variant: "error" });
      await close(first.message);
      const [repeat] = await write({ key, variant: "error" });
      await settleWrites(client);
      expect(screen.queryByText(repeat.message)).toBeNull();
    });

    it("should notify the next problem on the key of a closed one", async () => {
      await renderForward(client, { task: STATE });
      const key = createTaskKey();
      const [first] = await write({ key, variant: "error" });
      await close(first.message);
      await client.statuses.delete(key);
      const [next] = await write({ key, variant: "error" });
      expect(await screen.findByText(next.message)).toBeTruthy();
    });

    it("should take the problems on a key with no since as one", async () => {
      await renderForward(client, {});
      const local: status.Status = {
        key: uniqueName("alarm"),
        name: "",
        variant: "error",
        message: uniqueName("message"),
        description: "",
        time: TimeStamp.now(),
      };
      client.statuses.setLocal(local);
      client.statuses.setLocal(local);
      expect(
        within(await findNotification(local.message)).getByText("x2"),
      ).toBeTruthy();
    });

    it("should not notify a problem that is open at load", async () => {
      const key = createTaskKey();
      await write({ key, variant: "error" });
      await renderForward(client, { task: STATE });
      const [repeat, other] = await write(
        { key, variant: "error" },
        { variant: "error" },
      );
      await screen.findByText(other.message);
      expect(screen.queryByText(repeat.message)).toBeNull();
    });

    it("should count a write that the load returns once", async () => {
      const key = createTaskKey();
      const retrieve = client.statuses.retrieve.bind(client.statuses);
      const loading = Promise.withResolvers<void>();
      const written = Promise.withResolvers<void>();
      // Holds the load of the open problems until the problem is written.
      const spy = vi
        .spyOn(client.statuses, "retrieve")
        .mockImplementation(async (params) => {
          if (params.variants != null) {
            loading.resolve();
            await written.promise;
          }
          return await retrieve(params);
        });
      const rendering = renderForward(client, { task: STATE });
      await loading.promise;
      await write({ key, variant: "error" });
      written.resolve();
      const { result } = await rendering;
      spy.mockRestore();
      await settleWrites(client);
      expect(result.current.find(({ status: s }) => s.key === key)?.count).toBe(1);
    });

    it("should close a problem open at load on an earlier recovery", async () => {
      const key = createTaskKey();
      await write({ key, variant: "error" });
      const { result } = await renderForward(client, { task: STATE });
      expect(keysOf(result.current)).toContain(key);
      // The Driver can stamp a recovery on a clock that is behind the Core.
      await write({
        key,
        variant: "success",
        time: TimeStamp.now().sub(TimeSpan.minutes(1)),
      });
      await waitFor(() => expect(keysOf(result.current)).not.toContain(key));
    });

    it("should notify a reply to a problem that is open at load", async () => {
      const key = createTaskKey();
      await write({ key, variant: "error" });
      await renderForward(client, { task: REPLY });
      const [reply] = await write({ key, variant: "error" });
      expect(await screen.findByText(reply.message)).toBeTruthy();
    });
  });

  describe("cover", () => {
    it("should not notify a problem inside the covering problem", async () => {
      const covering = rack.statusKey(await createRackKey(client));
      await renderForward(client, {
        rack: STATE,
        task: { ...STATE, coveredBy: () => covering },
      });
      const [rackProblem, taskProblem] = await write(
        { key: covering, variant: "error" },
        { variant: "error" },
      );
      await screen.findByText(rackProblem.message);
      await settleWrites(client);
      expect(screen.queryByText(taskProblem.message)).toBeNull();
    });

    it("should notify a problem that starts before the covering problem", async () => {
      fakeTimers();
      const covering = rack.statusKey(await createRackKey(client));
      const key = createTaskKey();
      await renderForward(client, {
        rack: STATE,
        task: { ...DELAYED, coveredBy: () => covering },
      });
      const [, , again] = await write(
        { key, variant: "warning" },
        { key: covering, variant: "error" },
        { key, variant: "warning" },
      );
      await settleWrites(client);
      advance(TimeSpan.seconds(5));
      expect(screen.getByText(again.message)).toBeTruthy();
    });

    it("should notify a problem after the covering problem closes", async () => {
      const covering = rack.statusKey(await createRackKey(client));
      await renderForward(client, {
        rack: STATE,
        task: { ...STATE, coveredBy: () => covering },
      });
      const [, , taskProblem] = await write(
        { key: covering, variant: "error" },
        { key: covering, variant: "success" },
        { variant: "error" },
      );
      expect(await screen.findByText(taskProblem.message)).toBeTruthy();
    });

    it("should not notify a covered warning after the covering problem closes", async () => {
      fakeTimers();
      const covering = rack.statusKey(await createRackKey(client));
      await renderForward(client, {
        rack: STATE,
        task: { ...DELAYED, coveredBy: () => covering },
      });
      const [, taskProblem] = await write(
        { key: covering, variant: "error" },
        { variant: "warning" },
        { key: covering, variant: "success" },
      );
      await settleWrites(client);
      advance(TimeSpan.seconds(5));
      expect(screen.queryByText(taskProblem.message)).toBeNull();
    });

    it("should notify a covered problem that is still bad after the covering problem closes", async () => {
      const covering = rack.statusKey(await createRackKey(client));
      const key = createTaskKey();
      await renderForward(client, {
        rack: STATE,
        task: { ...STATE, coveredBy: () => covering },
      });
      const [, covered, , stillBad] = await write(
        { key: covering, variant: "error" },
        { key, variant: "error" },
        { key: covering, variant: "success" },
        { key, variant: "error" },
      );
      expect(await screen.findByText(stillBad.message)).toBeTruthy();
      expect(screen.queryByText(covered.message)).toBeNull();
    });

    it("should wait out the delay of a warning that leaves its cover", async () => {
      fakeTimers();
      const covering = rack.statusKey(await createRackKey(client));
      const key = createTaskKey();
      await renderForward(client, {
        rack: STATE,
        task: { ...DELAYED, coveredBy: () => covering },
      });
      await write({ key: covering, variant: "error" }, { key, variant: "warning" });
      await settleWrites(client);
      advance(TimeSpan.seconds(5));
      const [, uncovered] = await write(
        { key: covering, variant: "success" },
        { key, variant: "warning" },
      );
      await settleWrites(client);
      expect(screen.queryByText(uncovered.message)).toBeNull();
      advance(TimeSpan.seconds(5));
      expect(screen.getByText(uncovered.message)).toBeTruthy();
    });
  });

  describe("delayed warnings", () => {
    it("should notify a warning only when its problem is open after 5 s", async () => {
      fakeTimers();
      await renderForward(client, { task: DELAYED });
      const [warning] = await write({ variant: "warning" });
      await settleWrites(client);
      expect(screen.queryByText(warning.message)).toBeNull();
      advance(TimeSpan.seconds(5));
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
      advance(TimeSpan.seconds(5));
      expect(screen.queryByText(warning.message)).toBeNull();
    });

    it("should wait out the delay again when a closed warning comes back", async () => {
      fakeTimers();
      await renderForward(client, { task: DELAYED });
      const key = createTaskKey();
      const [, , back] = await write(
        { key, variant: "warning" },
        { key, variant: "success" },
        { key, variant: "warning" },
      );
      await settleWrites(client);
      expect(screen.queryByText(back.message)).toBeNull();
      advance(TimeSpan.seconds(5));
      expect(screen.getByText(back.message)).toBeTruthy();
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
      advance(TimeSpan.seconds(5));
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
      advance(TimeSpan.seconds(5));
      expect(screen.queryByText(warning.message)).toBeNull();
    });
  });

  describe("open problems", () => {
    it("should list a problem only when its thing is in use", async () => {
      const [unused, used] = [createTaskKey(), createTaskKey()];
      const { result } = await renderForward(client, { task: usedOnly(used) });
      await write({ key: unused, variant: "error" }, { key: used, variant: "error" });
      await waitFor(() => expect(keysOf(result.current)).toContain(used));
      expect(keysOf(result.current)).not.toContain(unused);
    });

    it("should list a status that no rule owns, but not a reply", async () => {
      const { result } = await renderForward(client, { task: REPLY });
      const [reply, alarm] = await write(
        { variant: "error" },
        { ...createAlarm(), variant: "error" },
      );
      await waitFor(() => expect(keysOf(result.current)).toContain(alarm.key));
      await screen.findByText(reply.message);
      expect(keysOf(result.current)).not.toContain(reply.key);
      // The problems view of other specs lists an alarm that stays open.
      await client.statuses.delete(alarm.key);
    });

    it("should list a problem open at load when its thing is in use", async () => {
      const [unused, used] = [createTaskKey(), createTaskKey()];
      await write({ key: unused, variant: "error" }, { key: used, variant: "error" });
      const { result } = await renderForward(client, { task: usedOnly(used) });
      await waitFor(() => expect(keysOf(result.current)).toContain(used));
      expect(keysOf(result.current)).not.toContain(unused);
    });

    it("should list, not notify, a problem open at load whose check fails", async () => {
      const key = createTaskKey();
      const exc = new Error("The Core is unreachable.");
      const [problem] = await write({ key, variant: "error" });
      const inUse = vi.fn(async ({ key: k }: Notifications.InUseParams) => {
        if (task.statusKey(k) === key) throw exc;
        return false;
      });
      const printError = vi.spyOn(console, "error").mockImplementation(() => {});
      const { result } = await renderForward(client, { task: { ...STATE, inUse } });
      await waitFor(() => expect(keysOf(result.current)).toContain(key));
      expect(screen.queryByText(problem.message)).toBeNull();
      expect(printError).toHaveBeenCalledWith(
        `failed to check if the thing of ${key} is in use`,
        exc,
      );
      printError.mockRestore();
    });

    it("should list a state that joins a reply open at load", async () => {
      const key = createTaskKey();
      await writeReply(key);
      const { result } = await renderForward(client, { task: commanded() });
      expect(keysOf(result.current)).not.toContain(key);
      const [state] = await write({ key, variant: "error" });
      await waitFor(() =>
        expect(result.current.map(({ status: s }) => s.message)).toContain(
          state.message,
        ),
      );
    });

    it("should list a problem whose notification a newer reply retires", async () => {
      const key = createTaskKey();
      const { result } = await renderForward(client, { task: commanded() });
      await write({ key, variant: "warning" });
      await screen.findByText(await writeReply(key));
      expect(keysOf(result.current)).toContain(key);
    });
  });

  describe("retired problems", () => {
    it("should notify the next write of a problem after a newer reply", async () => {
      const key = createTaskKey();
      await renderForward(client, { task: commanded() });
      const [warning] = await write({ key, variant: "warning" });
      await screen.findByText(await writeReply(key));
      const [error] = await write({ key, variant: "error" });
      expect(await screen.findByText(error.message)).toBeTruthy();
      expect(screen.queryByText(warning.message)).toBeNull();
    });

    it("should not notify a problem whose check ends after a newer reply", async () => {
      const key = createTaskKey();
      let resolve: ((inUse: boolean) => void) | undefined;
      const inUse = vi.fn(async ({ key: k }: Notifications.InUseParams) => {
        if (task.statusKey(k) !== key) return false;
        return await new Promise<boolean>((r) => (resolve = r));
      });
      await renderForward(client, { task: commanded({ inUse }) });
      const [error] = await write({ key, variant: "error" });
      await vi.waitFor(() => expect(resolve).toBeDefined());
      await screen.findByText(await writeReply(key));
      act(() => resolve?.(true));
      await settleWrites(client);
      expect(screen.queryByText(error.message)).toBeNull();
    });
  });
});
