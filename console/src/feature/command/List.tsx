// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Icon, List as Base, Text } from "@synnaxlabs/pluto";
import { type compare } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { type Command } from "@/platform/command";
import { Palette } from "@/platform/palette";

const sort: compare.Comparator<Command.Command> = (a, b) => {
  if (a.sortOrder != null && b.sortOrder != null) return a.sortOrder - b.sortOrder;
  return a.commandName.localeCompare(b.commandName);
};

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

export const List = ({ commands, ...rest }: ListProps): ReactElement => {
  // The list of commands is statically defined, which makes it safe to
  // call these hooks using a map function.
  const visibilities = commands.map((cmd) => cmd.useVisible?.() ?? true);
  const data = useMemo(
    () => commands.filter((_, i) => visibilities[i]),
    [commands, visibilities],
  );
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
