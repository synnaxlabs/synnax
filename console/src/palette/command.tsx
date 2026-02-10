// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import {
  Component,
  context,
  Flex,
  Flux,
  type Icon,
  List,
  type Pluto,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type compare } from "@synnaxlabs/x";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
} from "react";
import { useStore } from "react-redux";

import { type Export } from "@/export";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { type UseListReturn } from "@/palette/list";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface CommandProps extends List.ItemProps<string> {
  placeLayout: Layout.Placer;
  confirm: Modals.PromptConfirm;
  rename: Modals.PromptRename;
  handleError: Status.ErrorHandler;
  addStatus: Status.Adder;
  extractors: Export.Extractors;
  store: RootStore;
  fluxStore: Pluto.FluxStore;
  client: Client | null;
}

export interface Command extends FC<CommandProps> {
  key: string;
  commandName: string;
  sortOrder?: number;
  useVisible?: () => boolean;
}

export interface CommandListItemProps extends List.ItemProps<string> {
  name: string;
  icon?: Icon.ReactElement;
  onSelect: () => void;
  endContent?: ReactElement;
}

const SYNTHETIC_CLICK_DETAIL = 0;

const BaseCommandListItem = ({
  name,
  icon,
  onSelect,
  endContent,
  itemKey,
  ...props
}: CommandListItemProps & Record<string, unknown>): ReactElement => {
  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      // Only trigger on the synthetic click, which means we won't accidentally call
      // `onSelect` twice.
      if (e.detail === SYNTHETIC_CLICK_DETAIL) onSelect();
    },
    [onSelect],
  );
  return (
    <Select.ListItem
      highlightHovered
      justify="between"
      align="center"
      onClick={handleClick}
      itemKey={itemKey}
      data-command-key={itemKey}
      {...props}
    >
      <Text.Text weight={400} gap="medium">
        {icon}
        {name}
      </Text.Text>
      {endContent != null && <Flex.Box x>{endContent}</Flex.Box>}
    </Select.ListItem>
  );
};

export const CommandListItem = Component.removeProps(BaseCommandListItem, [
  "placeLayout",
  "confirm",
  "rename",
  "handleError",
  "addStatus",
  "extractors",
  "store",
  "fluxStore",
  "client",
]);

export interface SimpleCommandConfig {
  key: string;
  name: string;
  icon?: Icon.ReactElement;
  layout: Layout.PlacerArgs;
  useVisible?: () => boolean;
  sortOrder?: number;
}

export const createSimpleCommand = ({
  key,
  name,
  icon,
  layout,
  useVisible,
  sortOrder,
}: SimpleCommandConfig): Command => {
  const C: Command = ({ placeLayout, ...listProps }) => {
    const handleSelect = useCallback(() => placeLayout(layout), [placeLayout]);
    return (
      <CommandListItem {...listProps} name={name} icon={icon} onSelect={handleSelect} />
    );
  };
  C.key = key;
  C.commandName = name;
  C.sortOrder = sortOrder;
  C.useVisible = useVisible;
  return C;
};

interface ContextValue {
  commands: Command[];
  extractors: Export.Extractors;
}

const [CommandContext, useCommandContext] = context.create<ContextValue>({
  defaultValue: { commands: [], extractors: {} },
  displayName: "Palette.CommandContext",
});
export { useCommandContext };

export interface CommandProviderProps extends PropsWithChildren {
  commands: Command[];
  extractors: Export.Extractors;
}

export const CommandProvider = ({
  commands,
  extractors,
  children,
}: CommandProviderProps) => {
  const ctxValue = useMemo(
    () => ({ commands, extractors }),
    [commands, extractors],
  );
  return <CommandContext value={ctxValue}>{children}</CommandContext>;
};

const sort: compare.Comparator<Command> = (a, b) => {
  if (a.sortOrder != null && b.sortOrder != null) return a.sortOrder - b.sortOrder;
  return a.commandName.localeCompare(b.commandName);
};

export const useCommandList = (): UseListReturn<Command> => {
  const store = useStore<RootState, RootAction>();
  const client = Synnax.use();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const { commands, extractors } = useCommandContext();

  const visibilities = commands.map((cmd) => cmd.useVisible?.() ?? true);
  const visibleCommands = useMemo(
    () => commands.filter((_, i) => visibilities[i]),
    [commands, visibilities],
  );

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();
  const confirm = Modals.useConfirm();
  const rename = Modals.useRename();

  const commandProps = useMemo(
    () => ({
      placeLayout,
      confirm,
      rename,
      handleError,
      addStatus,
      extractors,
      store,
      fluxStore,
      client,
    }),
    [
      placeLayout,
      confirm,
      rename,
      handleError,
      addStatus,
      extractors,
      store,
      fluxStore,
      client,
    ],
  );

  const commandMap = useMemo(
    () => new Map(commands.map((cmd) => [cmd.key, cmd])),
    [commands],
  );

  const handleSelect = useCallback((key: string) => {
    const element = document.querySelector(`[data-command-key="${key}"]`);
    if (element == null || !(element instanceof HTMLElement)) return;
    element.click();
  }, []);

  const listItem = useMemo(
    () =>
      Component.renderProp((props: List.ItemProps<string>): ReactElement | null => {
        const cmd = commandMap.get(props.itemKey);
        if (cmd == null) return null;
        const Cmd = cmd;
        return <Cmd {...commandProps} {...props} />;
      }),
    [commandMap, commandProps],
  );

  const listProps = List.useStaticData<string, Command>({
    data: visibleCommands,
    sort,
  });

  return { ...listProps, handleSelect, listItem };
};
