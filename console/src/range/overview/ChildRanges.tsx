// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  List,
  Ranger,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { FC, useState } from "react";

import { Layout } from "@/layout";
import { createEditLayout, overviewLayout } from "@/range/external";

export const ChildRangeListItem = (props: List.ItemProps<string, ranger.Payload>) => {
  const { entry } = props;
  const placer = Layout.usePlacer();
  return (
    <List.ItemFrame
      onClick={() => placer({ ...overviewLayout, name: entry.name, key: entry.key })}
      direction="x"
      size={0.5}
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.5rem" }}
      {...props}
    >
      <Text.WithIcon
        startIcon={
          <Icon.Range
            style={{ transform: "scale(0.9)", color: "var(--pluto-gray-l9)" }}
          />
        }
        level="p"
        weight={450}
        shade={9}
        size="small"
      >
        {entry.name}{" "}
      </Text.WithIcon>
      <Ranger.TimeRangeChip level="small" timeRange={entry.timeRange} />
    </List.ItemFrame>
  );
};

const childRangeListItem = componentRenderProp(ChildRangeListItem);

export interface ChildRangesProps {
  rangeKey: string;
}

export const ChildRanges: FC<ChildRangesProps> = ({ rangeKey }) => {
  const client = Synnax.use();
  const placer = Layout.usePlacer();
  const [childRanges, setChildRanges] = useState<ranger.Range[]>([]);
  const addStatus = Status.useAggregator();

  useAsyncEffect(async () => {
    try {
      if (client == null) return;
      const rng = await client.ranges.retrieve(rangeKey);
      const childRanges = await rng.retrieveChildren();
      setChildRanges(childRanges);
      const tracker = await rng.openChildRangeTracker();
      tracker.onChange((ranges) => setChildRanges(ranges));
      return async () => await tracker.close();
    } catch (e) {
      addStatus({
        variant: "error",
        message: `Failed to retrieve child ranges`,
        description: (e as Error).message,
      });
      return undefined;
    }
  }, [rangeKey, client?.key]);

  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Child Ranges
      </Text.Text>
      <List.List data={childRanges}>
        <List.Core empty>{childRangeListItem}</List.Core>
      </List.List>
      <Button.Button
        size="medium"
        shade={8}
        weight={500}
        startIcon={<Icon.Add />}
        style={{ width: "fit-content" }}
        variant="text"
        onClick={() => placer(createEditLayout({ initial: { parent: rangeKey } }))}
      >
        Add Child Range
      </Button.Button>
    </Align.Space>
  );
};
