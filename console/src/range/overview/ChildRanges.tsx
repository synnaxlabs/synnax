// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Button, Header, Icon, Ranger } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { Layout } from "@/layout";
import { createCreateLayout } from "@/range/Create";
import { List } from "@/range/list/List";

export interface ChildRangesProps {
  rangeKey: string;
}

export const ChildRanges: FC<ChildRangesProps> = ({ rangeKey }) => {
  const placeLayout = Layout.usePlacer();
  const { data, getItem, subscribe, retrieve } = Ranger.useChildren({
    initialParams: { key: rangeKey },
  });
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
      <List data={data} getItem={getItem} subscribe={subscribe} retrieve={retrieve} />
    </Align.Space>
  );
};
