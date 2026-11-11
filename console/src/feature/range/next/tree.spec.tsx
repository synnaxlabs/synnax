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
import { act, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useEffect } from "react";
import { describe, expect, it } from "vitest";

import { TreeItem } from "@/feature/range/next/Item";
import { NodeProvider, useExpanded, useTree } from "@/feature/range/next/tree";
import { countEditableText, createConsoleWrapper, renderSuspended } from "@/testutil";

const client = createTestClient();

const noop = () => {};

const { Counted, counts, reset } = createRenderCounter();

const item = ({ key, ...props }: List.ItemProps<ranger.Key>): ReactElement => (
  <Counted key={key} id={props.itemKey}>
    <TreeItem {...props} />
  </Counted>
);

interface TreeProps {
  label: label.Key;
}

/** The ranges with the label as roots, in name order, with their children. */
const Tree = ({ label }: TreeProps): ReactElement => {
  const list = Flux.useList((c) => c.ranges, {
    name: "ranges",
    query: { where: { labels: [label] }, order: { field: "name", descending: false } },
  });
  useEffect(() => list.more(), []);
  const [expanded, toggle] = useExpanded();
  const tree = useTree({ roots: list.data, expanded });
  return (
    <Select.Frame
      multiple
      data={tree.nodes.map((n) => n.key)}
      getItem={list.getItem}
      subscribe={list.subscribe}
      value={[]}
      onChange={noop}
    >
      <NodeProvider
        nodes={tree.nodes}
        expanded={expanded}
        onExpand={toggle}
        more={tree.more}
      >
        <List.Scroll>
          <List.Items<ranger.Key>>{item}</List.Items>
        </List.Scroll>
      </NodeProvider>
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
const renderTree = async (label: label.Key, ranges: ranger.Range[]): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  await renderSuspended(<Tree label={label} />, { wrapper });
  await waitFor(() =>
    ranges.forEach((r) => expect(countEditableText(List.itemNameID(r.key))).toBe(1)),
  );
  await act(async () => {});
  reset();
};

describe("range tree", () => {
  it("should render only the item of a renamed range", async () => {
    const { label, ranges } = await createRanges();
    await renderTree(label, ranges);
    const [, b] = ranges;
    await client.ranges.rename(b.key, "b2");
    await screen.findByText("b2");
    await act(async () => {});
    expect(Object.fromEntries(counts)).toEqual({ [b.key]: 1 });
  });
});
