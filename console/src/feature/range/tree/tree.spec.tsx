// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { createRenderCounter } from "@synnaxlabs/lyra/testutil";
import { Flux } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type ComponentType, type ReactElement, useEffect } from "react";
import { describe, expect, it } from "vitest";

import { Tree } from "@/feature/range/tree";
import { countEditableText, createConsoleWrapper, renderSuspended } from "@/testutil";

const client = createTestClient();

const noop = () => {};

const { Counted, counts, reset } = createRenderCounter();

const counted =
  (Item: ComponentType<List.ItemProps<ranger.Key>>) =>
  ({ key, ...props }: List.ItemProps<ranger.Key>): ReactElement => (
    <Counted key={key} id={props.itemKey}>
      <Item {...props} />
    </Counted>
  );

type Layout = "tree" | "list" | "timeline";

const ITEMS = { tree: counted(Tree.Item), list: counted(Tree.FlatItem) };

interface HarnessProps {
  label: label.Key;
  layout: Layout;
}

/** The ranges with the label as roots, in name order, with their children. */
const Harness = ({ label, layout }: HarnessProps): ReactElement => {
  const list = Flux.useList((c) => c.ranges, {
    name: "ranges",
    query: { where: { labels: [label] }, order: { field: "name", descending: false } },
  });
  useEffect(() => list.more(), []);
  const [expanded, toggle] = Tree.useExpanded();
  const tree = Tree.use({ roots: list.data, expanded });
  return (
    <Select.Frame
      multiple
      data={tree.nodes.map((n) => n.key)}
      getItem={list.getItem}
      subscribe={list.subscribe}
      value={[]}
      onChange={noop}
    >
      {layout === "timeline" ? (
        <Tree.Layout
          layout="timeline"
          nodes={tree.nodes}
          expanded={expanded}
          onExpand={toggle}
          more={tree.more}
        />
      ) : (
        <Tree.NodeProvider
          nodes={tree.nodes}
          expanded={expanded}
          onExpand={toggle}
          more={tree.more}
        >
          <List.Scroll>
            <List.Items<ranger.Key>>{ITEMS[layout]}</List.Items>
          </List.Scroll>
        </Tree.NodeProvider>
      )}
    </Select.Frame>
  );
};

/** Creates the ranges a, b, and c with a new label. */
const createRanges = async (): Promise<{
  label: label.Key;
  ranges: ranger.Range[];
}> => {
  const { key } = await client.labels.create({ name: "tree", color: "#000000" });
  const end = TimeStamp.now().sub(TimeSpan.days(1));
  const ranges = await client.ranges.create(
    ["a", "b", "c"].map((name) => ({
      name,
      timeRange: new TimeRange(end.sub(TimeSpan.hours(1)), end),
      labels: [key],
    })),
  );
  return { label: key, ranges };
};

/** Shows the tree and waits for its items to settle, then resets the count. */
const renderTree = async (
  label: label.Key,
  ranges: ranger.Range[],
  layout: Layout = "tree",
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(<Harness label={label} layout={layout} />, { wrapper });
  await waitFor(() =>
    ranges.forEach((r) => expect(countEditableText(List.itemNameID(r.key))).toBe(1)),
  );
  await act(async () => {});
  reset();
};

describe("range tree", () => {
  it.each<Layout>(["tree", "list"])(
    "should render only the item of a renamed range in the %s layout",
    async (layout) => {
      const { label, ranges } = await createRanges();
      await renderTree(label, ranges, layout);
      const [, b] = ranges;
      await client.ranges.rename(b.key, "b2");
      await screen.findByText("b2");
      await act(async () => {});
      expect(Object.fromEntries(counts)).toEqual({ [b.key]: 1 });
    },
  );

  it("should mount each child once on expand and render no other item", async () => {
    const { label, ranges } = await createRanges();
    // The ranges below an expanded range move, so the last range keeps all still.
    const c = ranges[2];
    const end = TimeStamp.now().sub(TimeSpan.days(1));
    const children = await client.ranges.create(
      ["x", "y"].map((name) => ({
        name,
        timeRange: new TimeRange(end.sub(TimeSpan.hours(1)), end),
        parent: c.key,
      })),
    );
    await renderTree(label, ranges);
    const item = screen.getByRole("listitem", { name: "c" });
    fireEvent.click(within(item).getByRole("button", { name: "Expand" }));
    await waitFor(() =>
      children.forEach((c) =>
        expect(countEditableText(List.itemNameID(c.key))).toBe(1),
      ),
    );
    await act(async () => {});
    expect(Object.fromEntries(counts)).toEqual({ [c.key]: 1 });
  });

  it("should change no lane when the timeline zooms", async () => {
    const { label, ranges } = await createRanges();
    await renderTree(label, ranges, "timeline");
    const changes: MutationRecord[] = [];
    const observer = new MutationObserver((records) => changes.push(...records));
    screen.getAllByRole("listitem").forEach((lane) =>
      observer.observe(lane, {
        attributes: true,
        childList: true,
        characterData: true,
        subtree: true,
      }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    await screen.findByRole("button", { name: "Fit all" });
    await act(async () => {});
    observer.disconnect();
    expect(changes).toEqual([]);
  });

  it("should mount an added range once and render no other item", async () => {
    const { label, ranges } = await createRanges();
    await renderTree(label, ranges);
    const end = TimeStamp.now().sub(TimeSpan.days(1));
    await client.ranges.create({
      name: "d",
      timeRange: new TimeRange(end.sub(TimeSpan.hours(1)), end),
      labels: [label],
    });
    await screen.findByText("d");
    await act(async () => {});
    expect(Object.fromEntries(counts)).toEqual({});
  });
});
