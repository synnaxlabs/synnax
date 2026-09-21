// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/TopicListItem.css";

import { Flex, Form as PForm, type List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

export interface TopicListItemProps extends List.ItemProps<string> {
  path: string;
  extra?: ReactNode;
}

export const TopicListItem = ({ path, extra, ...props }: TopicListItemProps) => {
  const itemPath = `${path}.${props.itemKey}`;
  const topic = PForm.useFieldValue<string>(`${itemPath}.topic`);
  return (
    <Select.ListItem justify="between" align="center" x {...props}>
      <Flex.Box y gap="tiny" className={CSS.BE("mqtt-topic-list-item", "text")}>
        <Text.Text
          level="small"
          weight={500}
          status={topic === "" ? "disabled" : undefined}
          className={CSS.BE("mqtt-topic-list-item", "topic")}
        >
          {topic === "" ? "No topic" : `\u2066${topic}\u2069`}
        </Text.Text>
        {extra}
      </Flex.Box>
      <Task.EnableDisableButton path={`${itemPath}.disabled`} />
    </Select.ListItem>
  );
};
