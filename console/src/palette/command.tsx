// Copyright 2025 Synnax Labs, Inc.
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
  Flex,
  type Icon,
  List,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type compare } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useContext,
  useMemo,
} from "react";
import { useStore } from "react-redux";

import { type Export } from "@/export";
import { EXTRACTORS } from "@/extractors";
import { type Import } from "@/import";
import { FILE_INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { type UseListReturn } from "@/palette/list";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface CommandListItemProps extends List.ItemProps<string> {}

interface ContextValue {
  commands: Command[];
}

const CommandContext = createContext<ContextValue>({ commands: [] });

export interface CommandProviderProps extends PropsWithChildren {
  commands: Command[];
}

export const CommandProvider = ({ commands, children }: CommandProviderProps) => {
  const ctxValue = useMemo(() => ({ commands }), [commands]);
  return <CommandContext value={ctxValue}>{children}</CommandContext>;
};

export const useCommandContext = (): ContextValue => useContext(CommandContext);

export const listItem = Component.renderProp(
  (props: CommandListItemProps): ReactElement | null => {
    const { itemKey } = props;
    const cmd = List.useItem<string, Command>(itemKey);
    if (cmd == null) return null;
    const { icon, name, endContent } = cmd;
    return (
      <Select.ListItem highlightHovered justify="between" align="center" {...props}>
        <Text.Text weight={400} gap="medium">
          {icon}
          {name}
        </Text.Text>
        {endContent != null && <Flex.Box x>{endContent}</Flex.Box>}
      </Select.ListItem>
    );
  },
);

const sort: compare.Comparator<Command> = (a, b) => {
  if (a.sortOrder != null && b.sortOrder != null) return a.sortOrder - b.sortOrder;
  if (typeof a.name === "string" && typeof b.name === "string")
    return a.name.localeCompare(b.name);
  return 0;
};

export const useCommandList = (): UseListReturn<Command> => {
  const store = useStore<RootState, RootAction>();
  const { commands } = useCommandContext();
  const data = commands.filter(({ visible }) => visible?.(store.getState()) ?? true);
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const confirm = Modals.useConfirm();
  const rename = Modals.useRename();
  const handleSelect = useCallback(
    (key: string) => {
      const cmd = commands.find((c) => c.key === key);
      if (cmd == null) return;
      cmd.onSelect({
        addStatus,
        client,
        confirm,
        extractors: EXTRACTORS,
        handleError,
        fileIngestors: FILE_INGESTORS,
        placeLayout,
        rename,
        store,
      });
    },
    [addStatus, client, confirm, handleError, placeLayout, rename, store],
  );
  const listProps = List.useStaticData<string, Command>({ data, sort });
  return { ...listProps, handleSelect, listItem };
};

export interface CommandSelectionContext {
  store: RootStore;
  client: Client | null;
  placeLayout: Layout.Placer;
  confirm: Modals.PromptConfirm;
  addStatus: Status.Adder;
  rename: Modals.PromptRename;
  handleError: Status.ErrorHandler;
  fileIngestors: Import.FileIngestors;
  extractors: Export.Extractors;
}

export interface Command {
  key: string;
  name: ReactElement | string;
  sortOrder?: number;
  icon?: Icon.ReactElement;
  visible?: (state: RootState) => boolean;
  onSelect: (ctx: CommandSelectionContext) => void;
  endContent?: ReactElement[];
}
