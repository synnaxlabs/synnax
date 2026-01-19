// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Header, Icon, Ranger } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { Layout } from "@/layout";
import { createCreateLayout } from "@/range/Create";
import { List } from "@/range/list/List";

export interface ChildRangesProps {
  rangeKey: string;
}

export const ChildRanges: FC<ChildRangesProps> = ({ rangeKey }) => {
  const placeLayout = Layout.usePlacer();
  const { data, getItem, subscribe, retrieve } = Ranger.useListChildren({
    initialQuery: { key: rangeKey },
  });
  return (
    <Flex.Box y>
      <Header.Header level="h4" bordered borderColor={5}>
        <Header.Title color={11} weight={450}>
          Child Ranges
        </Header.Title>
        <Header.Actions>
          <Button.Button
            size="medium"
            variant="text"
            onClick={() => placeLayout(createCreateLayout({ parent: rangeKey }))}
          >
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <List
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        retrieve={retrieve}
        showParent={false}
        emptyContent={null}
      />
    </Flex.Box>
  );
};
