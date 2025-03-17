// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  List,
  Ranger,
  Text,
} from "@synnaxlabs/pluto";
import { type Keyed } from "@synnaxlabs/x";
import { type FC } from "react";

import { Layout } from "@/layout";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

type ListEntry = Keyed<ranger.Key>;

export const ChildRangeListItem = (props: List.ItemProps<ranger.Key, ListEntry>) => {
  const { entry } = props;
  const placeLayout = Layout.usePlacer();
  const { value: rng } = Ranger.useRetrieve(entry.key);
  if (rng == null) return null;
  return (
    <List.ItemFrame
      onClick={() =>
        placeLayout({ ...OVERVIEW_LAYOUT, name: rng.name, key: entry.key })
      }
      direction="x"
      size={0.5}
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.5rem" }}
      {...props}
    >
      <Text.WithIcon
        startIcon={<Icon.Range style={{ color: "var(--pluto-gray-l9)" }} />}
        level="p"
        weight={450}
        shade={9}
        size="small"
      >
        {rng.name}
      </Text.WithIcon>
      <Ranger.TimeRangeChip level="p" timeRange={rng.timeRange} showSpan />
    </List.ItemFrame>
  );
};

const childRangeListItem = componentRenderProp(ChildRangeListItem);

export interface ChildRangesProps {
  rangeKey: string;
}

export const ChildRanges: FC<ChildRangesProps> = ({ rangeKey }) => {
  const placeLayout = Layout.usePlacer();
  const childRanges = Ranger.useRetrieveChildren(rangeKey);
  const entries = childRanges?.map((k) => ({ key: k }) as Keyed<ranger.Key>);
  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={9} weight={450}>
        Child Ranges
      </Text.Text>
      <List.List data={entries}>
        <List.Core empty>{childRangeListItem}</List.Core>
      </List.List>
      <Button.Button
        size="medium"
        shade={8}
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
