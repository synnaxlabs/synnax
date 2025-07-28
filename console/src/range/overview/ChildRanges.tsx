// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import {
  Align,
  Button,
  componentRenderProp,
  Icon,
  List,
  Ranger,
  Text,
} from "@synnaxlabs/pluto";
import { type FC } from "react";

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
      gap="tiny"
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
        gap="small"
      >
        {entry.name}
      </Text.WithIcon>
      <Align.Space x gap="small">
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
  const res = Ranger.useChildren(rangeKey);
  let childRanges: ranger.Payload[] = [];
  if (res.variant === "success") childRanges = res.data.map(({ payload }) => payload);
  const placeLayout = Layout.usePlacer();
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
        gap="small"
        variant="text"
        onClick={() => placeLayout(createCreateLayout({ parent: rangeKey }))}
      >
        Add Child Range
      </Button.Button>
    </Align.Space>
  );
};
