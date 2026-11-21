// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Card } from "@/feature/range/toolbar/Card";
import { createTestLabel, dotColorBeside } from "@/platform/label/testutil";
import { createTestRange, rangeColorBeside } from "@/platform/range/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const renderCard = async (range: ranger.Range): Promise<HTMLElement> => {
  const { wrapper } = await createConsoleWrapper({ client });
  return render(<Card range={range} />, { wrapper }).container;
};

/** Creates a range from `start` ago to `end` ago, or with no end when `end` is null. */
const createAgo = async (
  name: string,
  start: TimeSpan,
  end: TimeSpan | null,
  parent?: ranger.Key,
): Promise<ranger.Range> => {
  const now = TimeStamp.now();
  return await createTestRange(client, {
    name,
    parent,
    timeRange: new TimeRange(
      now.sub(start),
      end == null ? TimeStamp.MAX : now.sub(end),
    ),
  });
};

/** Creates `count` children of `parent`, each one minute after the one before. */
const createChildren = async (
  parent: ranger.Range,
  count: number,
): Promise<ranger.Range[]> => {
  const start = parent.timeRange.start;
  return await client.ranges.create(
    Array.from({ length: count }, (_, i) => ({
      name: uniqueName(`child-${i}`),
      parent: parent.key,
      timeRange: new TimeRange(
        start.add(TimeSpan.minutes(i)),
        start.add(TimeSpan.minutes(i + 1)),
      ),
    })),
  );
};

describe("range/toolbar/Card", () => {
  it("should show the path from the root to the parent", async () => {
    const root = await createAgo(uniqueName("root"), TimeSpan.hours(3), null);
    const mid = await createAgo(uniqueName("mid"), TimeSpan.hours(2), null, root.key);
    const rng = await createAgo(uniqueName("leaf"), TimeSpan.hours(1), null, mid.key);
    await renderCard(rng);
    const midName = await screen.findByText(mid.name);
    const rootName = await screen.findByText(root.name);
    expect(
      rootName.compareDocumentPosition(midName) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("should show the total of a finished range", async () => {
    const rng = await createAgo(
      uniqueName("done"),
      TimeSpan.minutes(10),
      TimeSpan.minutes(5),
    );
    await renderCard(rng);
    expect(await screen.findByText("5m 00s")).toBeTruthy();
    expect(screen.getByText("Completed")).toBeTruthy();
  });

  it("should show a running range as running to now", async () => {
    const rng = await createAgo(uniqueName("live"), TimeSpan.minutes(1), null);
    const container = await renderCard(rng);
    expect(await screen.findByText("Running")).toBeTruthy();
    expect(container.textContent).toContain("Now");
  });

  it("should show no end or total for a planned range with no end", async () => {
    const rng = await createTestRange(client, {
      name: uniqueName("planned"),
      timeRange: new TimeRange(TimeStamp.now().add(TimeSpan.HOUR), TimeStamp.MAX),
    });
    const container = await renderCard(rng);
    expect(await screen.findByText("To do")).toBeTruthy();
    // The random suffix of the name can hold digits followed by a "d".
    const text = container.textContent.replace(rng.name, "");
    expect(text).not.toContain("2262");
    expect(text).not.toMatch(/\d+d/);
  });

  it("should list the first six children and count the rest", async () => {
    const rng = await createAgo(uniqueName("run"), TimeSpan.hours(1), null);
    const children = await createChildren(rng, 8);
    await renderCard(rng);
    for (const c of children.slice(0, 6))
      expect(await screen.findByText(c.name)).toBeTruthy();
    for (const c of children.slice(6)) expect(screen.queryByText(c.name)).toBeNull();
    expect(screen.getByText("2 more")).toBeTruthy();
  });

  it("should mark the count as partial past the first page of children", async () => {
    const rng = await createAgo(uniqueName("campaign"), TimeSpan.hours(2), null);
    await createChildren(rng, 52);
    await renderCard(rng);
    expect(await screen.findByText("44+ more")).toBeTruthy();
  });

  it("should show the labels of the range", async () => {
    const rng = await createAgo(uniqueName("labeled"), TimeSpan.minutes(1), null);
    const lbl = await createTestLabel(client, {
      name: uniqueName("label"),
      color: { type: "fixed", value: "#FF0000" },
    });
    await client.labels.label(rng.resourceID, [lbl.key]);
    await renderCard(rng);
    expect(await screen.findByText(lbl.name)).toBeTruthy();
  });

  it("should paint the named colors of the range, children, and labels", async () => {
    const [own, child, tag] = await client.colors.create(
      ["#111111", "#222222", "#333333"].map((light) => ({
        name: uniqueName("color"),
        light,
      })),
    );
    const rng = await createTestRange(client, {
      color: { type: "named", key: own.key },
    });
    const sub = await createTestRange(client, {
      name: uniqueName("child"),
      parent: rng.key,
      color: { type: "named", key: child.key },
    });
    const lbl = await createTestLabel(client, {
      name: uniqueName("label"),
      color: { type: "named", key: tag.key },
    });
    await client.labels.label(rng.resourceID, [lbl.key]);
    await renderCard(rng);
    const name = await screen.findByText(rng.name);
    await waitFor(() => expect(rangeColorBeside(name)).toBe("#111111"));
    const childName = await screen.findByText(sub.name);
    await waitFor(() => expect(rangeColorBeside(childName)).toBe("#222222"));
    const labelName = await screen.findByText(lbl.name);
    await waitFor(() => expect(dotColorBeside(labelName)).toBe("#333333"));
  });
});
