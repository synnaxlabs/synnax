// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/palette/Palette.css";

import {
  Button,
  Dialog,
  Flex,
  Icon,
  Text,
  Tooltip,
  Triggers,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { Command } from "@/app/command";
import { Search } from "@/app/search";
import { Command as FeatureCommand } from "@/feature/command";
import { CSS } from "@/platform/css";

const SEARCH_TRIGGER: Triggers.Trigger = ["Control", "P"];
const COMMAND_TRIGGER: Triggers.Trigger = ["Control", "Shift", "P"];

const TRIGGER_CONFIG: Triggers.ModeConfig<"search" | "command"> = {
  command: [COMMAND_TRIGGER],
  search: [SEARCH_TRIGGER],
  defaultMode: "command",
};

export const TooltipContent = (): ReactElement => (
  <Flex.Box gap="small">
    <Flex.Box x justify="between" align="center">
      <Text.Text level="small">Search</Text.Text>
      <Flex.Box x empty gap="tiny">
        <Triggers.Text trigger={SEARCH_TRIGGER} level="small" />
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x justify="between" align="center">
      <Text.Text level="small">Command Palette</Text.Text>
      <Flex.Box x gap="tiny">
        <Triggers.Text trigger={COMMAND_TRIGGER} level="small" />
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);

const flattenedConfig = Triggers.flattenConfig(TRIGGER_CONFIG);

const inputPlaceholder = (
  <>
    <Icon.Search />
    Type to search or {FeatureCommand.PREFIX} to view commands
  </>
);

export const Palette = (): ReactElement => {
  const [value, setValue] = useState("");
  const [visible, setVisible, visibleRef] = useCombinedStateAndRef<boolean>(false);

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start" || visibleRef.current) return;
      const mode = Triggers.determineMode(TRIGGER_CONFIG, triggers);
      setValue(mode === "command" ? FeatureCommand.PREFIX : "");
      setVisible(true);
    },
    [visibleRef],
  );

  const triggers = useMemo(() => flattenedConfig, [flattenedConfig]);

  Triggers.use({ triggers, callback: handleTrigger });

  const List = value.startsWith(FeatureCommand.PREFIX) ? Command.List : Search.List;

  return (
    <Tooltip.Dialog location="right" hide={visible}>
      <TooltipContent />
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
          justify="center"
          textColor={9}
          gap="small"
          full="x"
        >
          <Icon.Search />
        </Button.Button>
        <Dialog.Dialog
          className={CSS.BE("palette", "content")}
          bordered={false}
          pack
          rounded={1}
        >
          <List value={value} inputPlaceholder={inputPlaceholder} onChange={setValue} />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Tooltip.Dialog>
  );
};
