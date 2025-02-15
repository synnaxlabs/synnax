// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { Align, type Icon, List, type Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type Export } from "@/export";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Modals } from "@/modals";
import { type RootState, type RootStore } from "@/store";

export const CommandListItem = (
  props: List.ItemProps<string, Command>,
): ReactElement => {
  const {
    entry: { icon, name, endContent },
  } = props;
  return (
    <List.ItemFrame
      highlightHovered
      style={{ height: "6.5rem" }}
      justify="spaceBetween"
      align="center"
      {...props}
    >
      <Text.WithIcon startIcon={icon} level="p" weight={400} shade={9} size="medium">
        {name}
      </Text.WithIcon>
      {endContent != null && <Align.Space direction="x">{endContent}</Align.Space>}
    </List.ItemFrame>
  );
};

export interface CommandSelectionContext {
  store: RootStore;
  client: Synnax | null;
  placeLayout: Layout.Placer;
  confirm: Modals.PromptConfirm;
  addStatus: Status.AddStatusFn;
  handleException: Status.ExceptionHandler;
  ingestors: Record<string, Import.FileIngestor>;
  extractors: Record<string, Export.Extractor>;
  rename: Modals.PromptRename;
}

export interface Command {
  key: string;
  name: ReactElement | string;
  icon?: Icon.Element;
  visible?: (state: RootState) => boolean;
  onSelect: (ctx: CommandSelectionContext) => void;
  endContent?: ReactElement[];
}
