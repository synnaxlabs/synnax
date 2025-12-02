// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/palette/Palette.css";

import { type ontology } from "@synnaxlabs/client";
import {
  Button,
  Dialog,
  Icon,
  Input,
  List,
  Select,
  Text,
  Tooltip,
  Triggers,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { CSS } from "@/css";
import { type Command, useCommandList } from "@/palette/command";
import { useResourceList } from "@/palette/resource";
import { TooltipContent } from "@/palette/Tooltip";
import { type TriggerConfig } from "@/palette/types";

export interface PaletteProps {
  commandSymbol: string;
  triggerConfig: TriggerConfig;
}

export const Palette = ({
  commandSymbol,
  triggerConfig,
}: PaletteProps): ReactElement => {
  const [value, setValue] = useState("");
  const [visible, setVisible, visibleRef] = useCombinedStateAndRef<boolean>(false);

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start" || visibleRef.current) return;
      const mode = Triggers.determineMode(triggerConfig, triggers);
      setValue(mode === "command" ? commandSymbol : "");
      setVisible(true);
    },
    [triggerConfig, commandSymbol, visibleRef],
  );

  const triggers = useMemo(
    () => Triggers.flattenConfig(triggerConfig),
    [triggerConfig],
  );

  Triggers.use({ triggers, callback: handleTrigger });

  return (
    <Tooltip.Dialog location="bottom" hide={visible}>
      <TooltipContent triggerConfig={triggerConfig} />
      <Dialog.Frame
        visible={visible}
        onVisibleChange={setVisible}
        className={CSS.B("palette")}
        location="bottom"
        variant="modal"
        bordered={false}
      >
        <Button.Button
          onClick={() => setVisible(true)}
          className={CSS(CSS.BE("palette", "btn"))}
          variant="outlined"
          align="center"
          size="medium"
          justify="center"
          contrast={2}
          textColor={9}
          gap="small"
          full="x"
        >
          <Icon.Search />
          {/* This span needs to remain so we properly hide it on small window sizes. */}
          <Text.Text el="span" color={9}>
            Search and Command
          </Text.Text>
        </Button.Button>
        <Dialog.Dialog
          className={CSS.BE("palette", "content")}
          bordered={false}
          pack
          rounded={1}
        >
          <DialogContent
            value={value}
            onChange={setValue}
            commandSymbol={commandSymbol}
          />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Tooltip.Dialog>
  );
};

export interface PaletteDialogProps extends Input.Control<string> {
  commandSymbol: string;
}

const commandEmptyContent = (
  <Text.Text status="disabled" center level="h4">
    <Icon.Terminal />
    No commands found
  </Text.Text>
);

const resourceEmptyContent = (
  <Text.Text status="disabled" center level="h4">
    <Icon.Resources />
    No resources found
  </Text.Text>
);

const DialogContent = ({
  commandSymbol,
  onChange,
  value,
}: PaletteDialogProps): ReactElement => {
  const { close } = Dialog.useContext();
  const resourceProps = useResourceList();
  const commandProps = useCommandList();
  const mode = value.startsWith(commandSymbol) ? "command" : "resource";
  const { handleSelect, data, getItem, subscribe, listItem, retrieve } =
    mode === "command" ? commandProps : resourceProps;
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 20 });
  const handleSearch = useCallback(
    (v: string) => {
      onChange(v);
      if (v.startsWith(commandSymbol)) v = v.slice(commandSymbol.length);
      search(v);
    },
    [search, onChange],
  );
  return (
    <Select.Frame<string, Command | ontology.Resource>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      value={value}
      onChange={handleSelect}
      onFetchMore={fetchMore}
      itemHeight={36}
      initialHover={0}
      closeDialogOnSelect
      virtual
    >
      <Input.Text
        className={CSS(CSS.BE("palette", "input"))}
        placeholder={
          <>
            <Icon.Search />
            Type to search or {commandSymbol} to view commands
          </>
        }
        size="huge"
        autoFocus
        contrast={3}
        onChange={handleSearch}
        borderColor={8}
        value={value}
        autoComplete="off"
        onKeyDown={Triggers.matchCallback([["Escape"]], close)}
        full="x"
      />
      <List.Items
        className={CSS.BE("palette", "list")}
        emptyContent={mode === "command" ? commandEmptyContent : resourceEmptyContent}
        bordered
        borderColor={8}
        displayItems={10}
      >
        {listItem}
      </List.Items>
    </Select.Frame>
  );
};
