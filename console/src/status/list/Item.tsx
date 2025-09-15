// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Flex, List, Select, Status as StatusComp, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface ItemProps extends List.ItemProps<status.Key> {}

export const Item = (props: ItemProps): ReactElement => {
  const { itemKey } = props;
  const { onSelect, selected } = Select.useItemState(itemKey);
  const { getItem } = List.useUtilContext<status.Key, status.Status>();
  const item = getItem?.(itemKey);

  if (item == null) return <></>;

  return (
    <List.Item<status.Key> {...props} selected={selected} onSelect={onSelect}>
      <Flex.Box style={{ padding: "0.75rem 1rem" }}>
        <Flex.Box x justify="between" align="center">
          <StatusComp.Summary variant={item.variant} level="p" weight={500}>
            {item.name}
          </StatusComp.Summary>
          <Text.DateTime
            level="small"
            color="gray"
            format="dateTime"
          >
            {item.time}
          </Text.DateTime>
        </Flex.Box>
        {item.message && (
          <Text.Text level="small" color="gray" style={{ marginTop: "0.5rem" }}>
            {item.message}
          </Text.Text>
        )}
      </Flex.Box>
    </List.Item>
  );
};