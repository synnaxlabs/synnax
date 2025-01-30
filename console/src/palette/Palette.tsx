// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/palette/Palette.css";

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Dropdown,
  Input,
  List,
  Status,
  Synnax,
  Text,
  Tooltip,
  Triggers,
} from "@synnaxlabs/pluto";
import {
  type FC,
  type ReactElement,
  useCallback,
  useLayoutEffect,
  useMemo,
  useState,
} from "react";
import { useStore } from "react-redux";

import { Confirm } from "@/confirm";
import { CSS } from "@/css";
import { INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";
import {
  type Command,
  CommandListItem,
  type CommandSelectionContext,
} from "@/palette/command";
import { createResourceListItem } from "@/palette/resource";
import { TooltipContent } from "@/palette/Tooltip";
import { type Mode, type TriggerConfig } from "@/palette/types";
import { type RootAction, type RootState } from "@/store";

export interface PaletteProps {
  commands: Command[];
  services: Ontology.Services;
  triggers: TriggerConfig;
  commandSymbol: string;
}

type Entry = Command | ontology.Resource;
type Key = string;

export const Palette = ({
  commands,
  services,
  triggers: triggerConfig,
  commandSymbol,
}: PaletteProps): ReactElement => {
  const dropdown = Dropdown.use();

  const [value, setValue] = useState("");
  const store = useStore<RootState>();

  const newCommands = commands.filter((c) => c.visible?.(store.getState()) ?? true);

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start" || dropdown.visible) return;
      const mode = Triggers.determineMode(triggerConfig, triggers);
      setValue(mode === "command" ? commandSymbol : "");
      dropdown.open();
    },
    [dropdown.visible, dropdown.open, triggerConfig.command, commandSymbol],
  );

  const triggers = useMemo(
    () => Triggers.flattenConfig(triggerConfig),
    [triggerConfig],
  );

  Triggers.use({ triggers, callback: handleTrigger });

  return (
    <List.List>
      <Tooltip.Dialog location="bottom" hide={dropdown.visible}>
        <TooltipContent triggers={triggerConfig} />
        <Dropdown.Dialog
          {...dropdown}
          keepMounted={false}
          visible={dropdown.visible}
          className={CSS.B("palette")}
          location="bottom"
          variant="modal"
        >
          <Button.Button
            onClick={dropdown.open}
            className={CSS(CSS.BE("palette", "btn"))}
            variant="outlined"
            align="center"
            size="medium"
            justify="center"
            startIcon={<Icon.Search />}
            shade={7}
          >
            Quick Search & Command
          </Button.Button>
          <PaletteDialog
            value={value}
            onChange={setValue}
            commands={newCommands}
            services={services}
            commandSymbol={commandSymbol}
            close={dropdown.close}
          />
        </Dropdown.Dialog>
      </Tooltip.Dialog>
    </List.List>
  );
};

interface PaletteListProps {
  mode: Mode;
  services: Ontology.Services;
  commandSelectionContext: CommandSelectionContext;
}

const PaletteList = ({
  mode,
  services,
  commandSelectionContext,
}: PaletteListProps): ReactElement => {
  const item = useMemo(() => {
    const Item = (
      mode === "command" ? CommandListItem : createResourceListItem(services)
    ) as FC<List.ItemProps<string, ontology.Resource | Command>>;
    return componentRenderProp(Item);
  }, [commandSelectionContext, mode, services]);
  return (
    <List.Core className={CSS.BE("palette", "list")} itemHeight={27} grow>
      {item}
    </List.Core>
  );
};

export interface PaletteDialogProps extends Input.Control<string> {
  services: Ontology.Services;
  commandSymbol: string;
  commands: Command[];
  close: () => void;
}

const PaletteDialog = ({
  value,
  onChange,
  commands,
  services,
  commandSymbol,
  close,
}: PaletteDialogProps): ReactElement => {
  const { setSourceData } = List.useDataUtilContext<Key, Entry>();
  const addStatus = Status.useAggregator();
  const handleException = Status.useExceptionHandler();
  const client = Synnax.use();
  const store = useStore<RootState, RootAction>();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();

  const mode = value.startsWith(commandSymbol) ? "command" : "resource";

  useLayoutEffect(() => setSourceData(mode === "command" ? commands : []), [mode]);

  const confirm = Confirm.useModal();

  const cmdSelectCtx = useMemo<CommandSelectionContext>(
    () => ({
      store,
      placeLayout,
      confirm,
      client,
      addStatus,
      handleException,
      ingestors: INGESTORS,
    }),
    [store, placeLayout, confirm, client, addStatus, handleException],
  );

  const handleSelect = useCallback(
    (key: Key, { entries }: List.UseSelectOnChangeExtra<Key, Entry>) => {
      close();
      if (mode === "command") {
        (entries[0] as Command).onSelect(cmdSelectCtx);
        return;
      }
      if (client == null) return;
      const id = new ontology.ID(key);
      const t = services[id.type];
      t.onSelect?.({
        services,
        store,
        addStatus,
        placeLayout,
        removeLayout,
        handleException,
        client,
        selection: entries as ontology.Resource[],
      });
    },
    [
      close,
      mode,
      client,
      services,
      store,
      addStatus,
      placeLayout,
      removeLayout,
      handleException,
    ],
  );

  const { value: searchValue, onChange: onSearchChange } = List.useSearch<
    string,
    ontology.Resource
  >({ value, onChange, searcher: client?.ontology });

  const { value: filterValue, onChange: onFilterChange } = List.useFilter({
    value,
    onChange,
    transformBefore: (term) => term.slice(1),
  });

  const handleChange = useCallback(
    (value: string) => {
      const mode = value.startsWith(commandSymbol) ? "command" : "resource";
      if (mode === "command") onFilterChange(value);
      else onSearchChange(value);
    },
    [commandSymbol, searchValue, onFilterChange, onChange],
  );

  const actualValue = mode === "command" ? filterValue : searchValue;

  return (
    <List.Selector value={null} onChange={handleSelect} allowMultiple={false}>
      <List.Hover initialHover={0}>
        <Align.Pack
          className={CSS.BE("palette", "content")}
          direction="y"
          borderShade={4}
        >
          <Input.Text
            className={CSS(CSS.BE("palette", "input"))}
            placeholder={
              <Text.WithIcon level="h3" startIcon={<Icon.Search />}>
                Type to search or {commandSymbol} to view commands
              </Text.WithIcon>
            }
            size="huge"
            autoFocus
            onChange={handleChange}
            value={actualValue}
            autoComplete="off"
          />
          <PaletteList
            mode={mode}
            services={services}
            commandSelectionContext={cmdSelectCtx}
          />
        </Align.Pack>
      </List.Hover>
    </List.Selector>
  );
};
