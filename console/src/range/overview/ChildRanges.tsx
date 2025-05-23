// Copyright 2025 Synnax Labs, Inc.
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
import { type FC, useState } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export const ChildRangeListItem = (props: List.ItemProps<string, ranger.Payload>) => {
  const { entry } = props;
  const placeLayout = Layout.usePlacer();
  return (
    <List.ItemFrame
      onClick={() =>
        placeLayout({ ...OVERVIEW_LAYOUT, name: entry.name, key: entry.key })
      }
      x
      size="tiny"
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.5rem" }}
      {...props}
    >
      <Text.WithIcon
        startIcon={<Icon.Range style={{ color: "var(--pluto-gray-l11)" }} />}
        level="p"
        weight={450}
        shade={11}
        size="small"
      >
        {entry.name}
      </Text.WithIcon>
      <Align.Space x size="small">
        <Ranger.TimeRangeChip level="p" timeRange={entry.timeRange} showSpan />
      </Align.Space>
    </List.ItemFrame>
  );
};

const childRangeListItem = componentRenderProp(ChildRangeListItem);

export interface ChildRangesProps {
  rangeKey: string;
}

export const ChildRanges: FC<ChildRangesProps> = ({ rangeKey }) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const [childRanges, setChildRanges] = useState<ranger.Range[]>([]);
  const handleError = Status.useErrorHandler();

  useAsyncEffect(async () => {
    try {
      if (client == null) throw NULL_CLIENT_ERROR;
      const rng = await client.ranges.retrieve(rangeKey);
      const childRanges = await rng.retrieveChildren();
      childRanges.sort(ranger.sort);
      setChildRanges(childRanges);
      const tracker = await rng.openChildRangeTracker();
      tracker.onChange((ranges) => setChildRanges(ranges));
      return async () => await tracker.close();
    } catch (e) {
      handleError(e, `Failed to retrieve child ranges`);
      return undefined;
    }
  }, [rangeKey, client?.key]);

  return (
    <Align.Space y>
      <Text.Text level="h4" shade={11} weight={450}>
        Child Ranges
      </Text.Text>
      <List.List data={childRanges}>
        <List.Core empty>{childRangeListItem}</List.Core>
      </List.List>
      <Button.Button
        size="medium"
        shade={10}
        weight={500}
        startIcon={<Icon.Add />}
        style={{ width: "fit-content" }}
        iconSpacing="small"
        variant="text"
        onClick={() => placeLayout(createCreateLayout({ parent: rangeKey }))}
      >
        Add Child Range
      </Button.Button>
    </Align.Space>
  );
};
