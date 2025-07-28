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
  Component,
  Header,
  Icon,
  List,
  Ranger,
} from "@synnaxlabs/pluto";
import { type FC } from "react";

import { Layout } from "@/layout";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export const ChildRangeListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const entry = List.useItem<string, ranger.Range>(itemKey);
  const placeLayout = Layout.usePlacer();
  if (entry == null) return null;
  return (
    <Ranger.ListItem
      onClick={() =>
        placeLayout({ ...OVERVIEW_LAYOUT, name: entry.name, key: entry.key })
      }
      x
      showParent={false}
      gap="tiny"
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.5rem" }}
      {...props}
    />
  );
};

const childRangeListItem = Component.renderProp(ChildRangeListItem);

export interface ChildRangesProps {
  rangeKey: string;
}

export const ChildRanges: FC<ChildRangesProps> = ({ rangeKey }) => {
  const { getItem, subscribe, data, retrieve } = Ranger.useChildren();
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Space y>
      <Header.Header level="h4" bordered={false} borderShade={5}>
        <Header.Title shade={11} weight={450}>
          Child Ranges
        </Header.Title>
        <Header.Actions>
          <Button.Icon
            size="medium"
            shade={0}
            onClick={() => placeLayout(createCreateLayout({ parent: rangeKey }))}
          >
            <Icon.Add />
          </Button.Icon>
        </Header.Actions>
      </Header.Header>
      <List.Frame
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={() => {
          retrieve({ key: rangeKey });
        }}
      >
        <List.Items>{childRangeListItem}</List.Items>
      </List.Frame>
    </Align.Space>
  );
};
