// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/palette/Palette.css";

import { ontology } from "@synnaxlabs/client";
import {
  Align,
  Button,
  componentRenderProp,
  Dropdown,
  Icon,
  Input,
  List,
  Status,
  Synnax,
  Text,
  Tooltip,
  Triggers,
} from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useCallback, useMemo, useState } from "react";
import { useStore } from "react-redux";

import { CSS } from "@/css";
import { EXTRACTORS } from "@/extractors";
import { INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";
import {
  type Command,
  CommandListItem,
  type CommandSelectionContext,
} from "@/palette/command";
import { createResourceListItem } from "@/palette/resource";
import { TooltipContent } from "@/palette/Tooltip";
import { type Mode, type TriggerConfig } from "@/palette/types";
import { type RootAction, type RootState } from "@/store";

type Key = string;
type Entry = Command | ontology.Resource;

export interface PaletteProps {
  commands: Command[];
  commandSymbol: string;
  triggerConfig: TriggerConfig;
}

export const Palette = ({
  commands,
  commandSymbol,
  triggerConfig,
}: PaletteProps): ReactElement => {
  const { close, open, visible } = Dropdown.use();

  const [value, setValue] = useState("");
  const store = useStore<RootState>();

  const newCommands = commands.filter(
    ({ visible }) => visible?.(store.getState()) ?? true,
  );

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start" || visible) return;
      const mode = Triggers.determineMode(triggerConfig, triggers);
      setValue(mode === "command" ? commandSymbol : "");
      open();
    },
    [visible, triggerConfig, commandSymbol, open],
  );

  const triggers = useMemo(
    () => Triggers.flattenConfig(triggerConfig),
    [triggerConfig],
  );

  Triggers.use({ triggers, callback: handleTrigger });

  const data = value.startsWith(commandSymbol) ? newCommands : [];

  return (
    <List.List<Key, Entry> data={data}>
      <Tooltip.Dialog location="bottom" hide={visible}>
        <TooltipContent triggerConfig={triggerConfig} />
        <Dropdown.Dialog
          close={close}
          keepMounted={false}
          visible={visible}
          className={CSS.B("palette")}
          location="bottom"
          variant="modal"
          bordered={false}
        >
          <Button.Button
            onClick={open}
            className={CSS(CSS.BE("palette", "btn"))}
            variant="outlined"
            align="center"
            size="medium"
            justify="center"
            startIcon={<Icon.Search />}
            shade={2}
            textShade={9}
            iconSpacing="small"
          >
            Search & Command
          </Button.Button>
          <PaletteDialog
            value={value}
            onChange={setValue}
            commandSymbol={commandSymbol}
            close={close}
          />
        </Dropdown.Dialog>
      </Tooltip.Dialog>
    </List.List>
  );
};

const transformBefore = (term: string): string => term.slice(1);

export interface PaletteDialogProps
  extends Input.Control<string>,
    Pick<Dropdown.DialogProps, "close"> {
  commandSymbol: string;
}

const PaletteDialog = ({
  close,
  commandSymbol,
  onChange,
  value,
}: PaletteDialogProps): ReactElement => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const store = useStore<RootState, RootAction>();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();

  const mode = value.startsWith(commandSymbol) ? "command" : "search";

  const confirm = Modals.useConfirm();
  const rename = Modals.useRename();

  const cmdSelectCtx = useMemo<CommandSelectionContext>(
    () => ({
      addStatus,
      client,
      confirm,
      extractors: EXTRACTORS,
      handleError,
      ingestors: INGESTORS,
      placeLayout,
      rename,
      store,
    }),
    [addStatus, client, confirm, handleError, placeLayout, rename, store],
  );

  const services = Ontology.useServices();

  const handleSelect = useCallback(
    (key: Key, { entries }: List.UseSelectOnChangeExtra<Key, Entry>) => {
      close();
      if (mode === "command") {
        (entries[0] as Command).onSelect(cmdSelectCtx);
        return;
      }
      if (client == null) return;
      const { type } = new ontology.ID(key);
      services[type].onSelect?.({
        services,
        store,
        addStatus,
        placeLayout,
        removeLayout,
        handleError,
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
      handleError,
    ],
  );

  const { onChange: onSearchChange } = List.useSearch<string, ontology.Resource>({
    onChange,
    searcher: client?.ontology,
    value,
  });

  const { onChange: onFilterChange } = List.useFilter({
    onChange,
    transformBefore,
    value,
  });

  const handleChange = useCallback(
    (value: string) => {
      if (value.startsWith(commandSymbol)) onFilterChange(value);
      else onSearchChange(value);
    },
    [onFilterChange, onSearchChange],
  );

  return (
    <List.Selector<Key, Entry>
      value={null}
      onChange={handleSelect}
      allowMultiple={false}
    >
      <List.Hover<Key, Entry> initialHover={0}>
        <Align.Pack className={CSS.BE("palette", "content")} y bordered={false}>
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
            value={value}
            autoComplete="off"
            onKeyDown={Triggers.matchCallback([["Escape"]], () => close())}
          />
          <PaletteList mode={mode} services={services} />
        </Align.Pack>
      </List.Hover>
    </List.Selector>
  );
};

interface PaletteListProps {
  mode: Mode;
  services: Ontology.Services;
}

const PaletteList = ({ mode, services }: PaletteListProps): ReactElement => {
  const item = useMemo(() => {
    const Item = (
      mode === "command" ? CommandListItem : createResourceListItem(services)
    ) as FC<List.ItemProps<Key, Entry>>;
    return componentRenderProp(Item);
  }, [mode, services]);
  return (
    <List.Core<Key, Entry>
      className={CSS.BE("palette", "list")}
      itemHeight={27}
      grow
      bordered
      borderShade={6}
      background={0}
    >
      {item}
    </List.Core>
  );
};
