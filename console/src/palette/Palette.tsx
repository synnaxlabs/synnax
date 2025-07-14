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
  Align,
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
          startIcon={<Icon.Search />}
          shade={2}
          textShade={9}
          iconSpacing="small"
        >
          Search & Command
        </Button.Button>
        <DialogContent
          value={value}
          onChange={setValue}
          commandSymbol={commandSymbol}
        />
      </Dialog.Frame>
    </Tooltip.Dialog>
  );
};

export interface PaletteDialogProps extends Input.Control<string> {
  commandSymbol: string;
}

const DialogContent = ({
  commandSymbol,
  onChange,
  value,
}: PaletteDialogProps): ReactElement => {
  const { close } = Dialog.useContext();
  const resourceProps = useResourceList();
  const commandProps = useCommandList();
  const { handleSelect, data, getItem, subscribe, listItem } = value.startsWith(
    commandSymbol,
  )
    ? commandProps
    : resourceProps;
  return (
    <Select.Frame<string, Command | ontology.Resource>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      value={value}
      onChange={handleSelect}
    >
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
          onChange={onChange}
          value={value}
          autoComplete="off"
          onKeyDown={Triggers.matchCallback([["Escape"]], () => close())}
        />
        <List.Items className={CSS.BE("palette", "list")}>{listItem}</List.Items>
      </Align.Pack>
    </Select.Frame>
  );
};
