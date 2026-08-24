// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, type Icon, type List, Text, Triggers } from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useCallback } from "react";

import { Palette } from "@/platform/palette";

export interface CommandProps extends List.ItemProps<string> {}

export interface Command extends FC<CommandProps> {
  key: string;
  commandName: string;
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
    <Text.Text gap="medium">
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
  /** Shortcut that runs the same action; rendered as a hint on the list item. */
  trigger?: Triggers.Trigger;
}

export const create = ({
  key,
  name,
  icon,
  useOnSelect,
  useVisible,
  trigger,
}: CreateParams): Command => {
  const Cmd: Command = (listProps) => {
    const handleSelect = useOnSelect();
    const onSelect = useCallback(() => handleSelect(), [handleSelect]);
    return (
      <ListItem
        {...listProps}
        name={name}
        icon={icon}
        onSelect={onSelect}
        endContent={
          trigger != null ? (
            <Triggers.Text trigger={trigger} level="small" color={9} />
          ) : undefined
        }
      />
    );
  };
  Cmd.key = key;
  Cmd.commandName = name;
  Cmd.useVisible = useVisible;
  return Cmd;
};
