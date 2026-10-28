// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List as Base } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { type compare } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Command } from "@/platform/command";
import { Palette } from "@/platform/palette";

const sort: compare.Comparator<Command.Command> = (a, b) =>
  a.commandName.localeCompare(b.commandName);

const ListItem = (props: Base.ItemProps<string>): ReactElement | null => {
  const Cmd = Base.useItem<string, Command.Command>(props.itemKey);
  if (Cmd == null) return null;
  return <Cmd {...props} />;
};

const listItem = Component.renderProp(ListItem);

const emptyContent = (
  <Text.Text status="disabled" center level="h4">
    <Icon.Terminal />
    No commands found
  </Text.Text>
);

export const PREFIX = ">";

export interface ListProps extends Palette.ListProps<Command.Command> {
  commands: Command.Command[];
}

const useVisibleCommands = (commands: Command.Command[]): Command.Command[] => {
  "use no memo"; // Calls a hook per command; commands is a fixed module-level list.
  const visibilities = commands.map((cmd) => cmd.useVisible?.() ?? true);
  return commands.filter((_, i) => visibilities[i]);
};

export const List = ({ commands, ...rest }: ListProps): ReactElement => {
  const data = useVisibleCommands(commands);
  const listProps = Base.useStaticData<string, Command.Command>({ data, sort });
  return (
    <Palette.BaseList
      {...listProps}
      {...rest}
      prefix={PREFIX}
      listItem={listItem}
      emptyContent={emptyContent}
    />
  );
};
List.displayName = "Command.List";
