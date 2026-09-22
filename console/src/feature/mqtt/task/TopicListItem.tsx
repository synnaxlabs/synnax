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

import { type SparkplugTagID } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

export interface TopicListItemProps extends List.ItemProps<string> {
  path: string;
  extra?: ReactNode;
}

interface BaseProps extends TopicListItemProps {
  title: string;
  placeholder: string;
}

const Base = ({ path, title, placeholder, extra, ...props }: BaseProps) => (
  <Select.ListItem justify="between" align="center" x {...props}>
    <Flex.Box y gap="tiny" className={CSS.BE("mqtt-topic-list-item", "text")}>
      <Text.Text
        level="small"
        weight={500}
        status={title === "" ? "disabled" : undefined}
        className={CSS.BE("mqtt-topic-list-item", "topic")}
      >
        {title === "" ? placeholder : `\u2066${title}\u2069`}
      </Text.Text>
      {extra}
    </Flex.Box>
    <Task.EnableDisableButton path={`${path}.${props.itemKey}.disabled`} />
  </Select.ListItem>
);

export const TopicListItem = (props: TopicListItemProps) => {
  const topic = PForm.useFieldValue<string>(`${props.path}.${props.itemKey}.topic`);
  return <Base {...props} title={topic} placeholder="No topic" />;
};

export const SparkplugListItem = ({ extra, ...props }: TopicListItemProps) => {
  const { group, edgeNode, device, tag } = PForm.useFieldValue<SparkplugTagID>(
    `${props.path}.${props.itemKey}`,
  );
  const node = [group, edgeNode, device].filter((id) => id !== "").join("/");
  return (
    <Base
      {...props}
      title={tag}
      placeholder="No tag"
      extra={
        <>
          <Text.Text level="small" color={9} overflow="ellipsis">
            {node === "" ? "No edge node" : node}
          </Text.Text>
          {extra}
        </>
      }
    />
  );
};
