// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, rack, status, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Status } from "@/feature/status";
import { Nav } from "@/platform/nav";
import { Notifications } from "@/platform/notifications";
import { createRackKey, renderForward } from "@/platform/notifications/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const rackDetailsZ = z.object({ rack: z.number() });

/**
 * @returns rules that keep only the keys in mine in use, since the Core holds the
 * problems of every other spec.
 */
const createRules = (mine: Set<status.Key>): Notifications.Rules => {
  const createRule = (type: ontology.ResourceType): Notifications.Rule => ({
    kind: () => "state",
    inUse: async ({ key }) => mine.has(ontology.idToString({ type, key })),
    coveredBy: (stat) => {
      const r = rackDetailsZ.safeParse(status.detailsOf(stat)).data?.rack;
      return r == null ? null : rack.statusKey(r);
    },
  });
  return {
    channel: createRule("channel"),
    device: createRule("device"),
    rack: createRule("rack"),
    task: createRule("task"),
  };
};

interface Problems {
  mine: Set<status.Key>;
  /** @returns the key of a new task status that is in use. */
  createTaskKey: () => status.Key;
  /** @returns the key of a new rack status that is in use. */
  createRackStatusKey: (r: rack.Key) => status.Key;
}

const createProblems = (): Problems => {
  const mine = new Set<status.Key>();
  const add = (key: status.Key): status.Key => {
    mine.add(key);
    return key;
  };
  return {
    mine,
    createTaskKey: () => add(task.statusKey(uuid.create())),
    createRackStatusKey: (r) => add(rack.statusKey(r)),
  };
};

const TOOLBAR = (
  <>
    <Nav.Menu
      items={Status.TOOLBAR}
      onSelect={vi.fn()}
      onStartHover={vi.fn()}
      onStopHover={vi.fn()}
      onToggle={vi.fn()}
      onPin={vi.fn()}
    />
    {Status.TOOLBAR.content}
  </>
);

const renderProblems = async (mine: Set<status.Key>) =>
  await renderForward(client, createRules(mine), TOOLBAR);

/** @returns the message of the status. */
const write = async (
  key: status.Key,
  variant: status.Variant,
  details?: unknown,
): Promise<string> => {
  const { message } = await client.statuses.set({
    key,
    name: uniqueName("name"),
    variant,
    message: uniqueName("message"),
    time: TimeStamp.now(),
    details,
  });
  return message;
};

const panel = () => within(screen.getByRole("tabpanel"));

/** @returns the entry in the problems view that shows the text. */
const findProblem = async (text: string): Promise<HTMLElement> => {
  const found = (await panel().findByText(text)).closest<HTMLElement>(
    ".console-status-problem",
  );
  if (found == null) throw new Error(`no problem shows ${text}`);
  return found;
};

const follows = (a: HTMLElement, b: HTMLElement): boolean =>
  (b.compareDocumentPosition(a) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;

const findBadge = async (): Promise<HTMLElement | null> =>
  (
    await screen.findByRole("menuitem", { name: "Statuses" })
  ).querySelector<HTMLElement>(".console-main-nav__badge");

describe("Status.TOOLBAR problems", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should show that no problem is open", async () => {
    // The forwarder would list the problems that other specs leave on the Core.
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Notifications.Provider>{TOOLBAR}</Notifications.Provider>, { wrapper });
    expect(await panel().findByText("No problems")).toBeTruthy();
    expect(await findBadge()).toBeNull();
  });

  it("should list problems newest first, by when each opened", async () => {
    const { mine, createTaskKey } = createProblems();
    const [older, newer] = [createTaskKey(), createTaskKey()];
    await renderProblems(mine);
    const olderMessage = await write(older, "error");
    const newerMessage = await write(newer, "error");
    // A later write of a problem keeps the time the problem opened.
    const rewritten = await write(older, "error");
    await findProblem(rewritten);
    expect(follows(await findProblem(rewritten), await findProblem(newerMessage))).toBe(
      true,
    );
    expect(panel().queryByText(olderMessage)).toBeNull();
  });

  it("should show the count of a problem", async () => {
    const { mine, createTaskKey } = createProblems();
    const key = createTaskKey();
    await renderProblems(mine);
    await write(key, "error");
    await write(key, "error");
    const last = await write(key, "error");
    expect(within(await findProblem(last)).getByText("x3")).toBeTruthy();
  });

  it("should show how long a problem has been open", async () => {
    vi.useFakeTimers({
      toFake: ["Date", "setTimeout", "clearTimeout"],
      shouldAdvanceTime: true,
    });
    const { mine, createTaskKey } = createProblems();
    const key = createTaskKey();
    const { since } = await client.statuses.set({
      key,
      name: uniqueName("name"),
      variant: "error",
      message: uniqueName("message"),
      time: TimeStamp.now(),
    });
    if (since == null) throw new Error("the Core must stamp when the problem started");
    // The Core stamps since by its own clock, which the fake clock lags under load.
    vi.setSystemTime(since.add(TimeSpan.milliseconds(1500)).date());
    const last = await write(key, "error");
    await renderProblems(mine);
    // A start at the last write would show "0s" or "< 1s".
    expect(within(await findProblem(last)).getByText(/^[1-9]\d*s$/)).toBeTruthy();
  });

  it("should list a problem under the open problem that covers it", async () => {
    const { mine, createTaskKey, createRackStatusKey } = createProblems();
    const [down, up] = await Promise.all([
      createRackKey(client),
      createRackKey(client),
    ]);
    const [covered, uncovered] = [createTaskKey(), createTaskKey()];
    await renderProblems(mine);
    const rackMessage = await write(createRackStatusKey(down), "error");
    const coveredMessage = await write(covered, "error", { rack: down });
    const uncoveredMessage = await write(uncovered, "error", { rack: up });
    const rackProblem = await findProblem(rackMessage);
    const coveredProblem = await findProblem(coveredMessage);
    expect(rackProblem.nextElementSibling).toBe(coveredProblem);
    expect(coveredProblem.classList).toContain("console--nested");
    const uncoveredProblem = await findProblem(uncoveredMessage);
    expect(uncoveredProblem.classList).not.toContain("console--nested");
  });

  it("should drop a problem when its key turns good", async () => {
    const { mine, createTaskKey } = createProblems();
    const key = createTaskKey();
    await renderProblems(mine);
    const message = await write(key, "error");
    await findProblem(message);
    await write(key, "success");
    await waitFor(() => expect(panel().queryByText(message)).toBeNull());
  });

  it("should drop a problem when its status is deleted", async () => {
    const { mine, createTaskKey } = createProblems();
    const key = createTaskKey();
    await renderProblems(mine);
    const message = await write(key, "error");
    await findProblem(message);
    await client.statuses.delete(key);
    await waitFor(() => expect(panel().queryByText(message)).toBeNull());
  });

  it("should count open problems on the nav icon, in the worst variant", async () => {
    const { mine, createTaskKey } = createProblems();
    const [warned, failed] = [createTaskKey(), createTaskKey()];
    // Problems with no rule, which other specs leave on the Core, count too.
    const { result } = await renderProblems(mine);
    const expectBadge = async (): Promise<void> => {
      const badge = await findBadge();
      expect(badge?.textContent).toBe(String(result.current.length));
      const failing = result.current.some(({ status: s }) => s.variant === "error");
      expect(badge?.classList).toContain(
        failing ? "console--error" : "console--warning",
      );
    };
    const holds = (key: status.Key): boolean =>
      result.current.some(({ status: s }) => s.key === key);
    await write(warned, "warning");
    await waitFor(() => expect(holds(warned)).toBe(true));
    await waitFor(expectBadge);
    await write(failed, "error");
    await waitFor(() => expect(holds(failed)).toBe(true));
    await waitFor(expectBadge);
    expect((await findBadge())?.classList).toContain("console--error");
  });
});
