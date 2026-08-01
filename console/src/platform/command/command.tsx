// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, type Icon, type List, Text } from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useCallback } from "react";

import { Palette } from "@/platform/palette";

export interface CommandProps extends List.ItemProps<string> {}

export interface Command extends FC<CommandProps> {
  key: string;
  commandName: string;
  sortOrder?: number;
  useVisible?: () => boolean;
}

export interface ListItemProps extends Omit<Palette.ListItemProps, "children"> {
  name: string;
  onSelect: () => void;
  icon: Icon.ReactElement;
  endContent?: ReactElement;
}

export const ListItem = ({
  name,
  icon,
  onSelect,
  endContent,
  itemKey,
  ...props
}: ListItemProps): ReactElement => (
  <Palette.ListItem
    justify="between"
    align="center"
    onSelect={onSelect}
    itemKey={itemKey}
    data-command-key={itemKey}
    {...props}
  >
    <Text.Text weight={400} gap="medium">
      {icon}
      {name}
    </Text.Text>
    {endContent != null && <Flex.Box x>{endContent}</Flex.Box>}
  </Palette.ListItem>
);

export interface CreateParams {
  key: string;
  name: string;
  icon: Icon.ReactElement;
  useOnSelect: () => () => void;
  useVisible?: () => boolean;
  sortOrder?: number;
}

export const create = ({
  key,
  name,
  icon,
  useOnSelect,
  useVisible,
  sortOrder,
}: CreateParams): Command => {
  const Cmd: Command = (listProps) => {
    const handleSelect = useOnSelect();
    const onSelect = useCallback(() => handleSelect(), [handleSelect]);
    return <ListItem {...listProps} name={name} icon={icon} onSelect={onSelect} />;
  };
  Cmd.key = key;
  Cmd.commandName = name;
  Cmd.sortOrder = sortOrder;
  Cmd.useVisible = useVisible;
  return Cmd;
};
