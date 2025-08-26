// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type state } from "@/state";
import { Text } from "@/text";
import { type Theming } from "@/theming";
import { stopPropagation } from "@/util/event";

export interface EntryData {
  color: color.Crude;
  key: string;
  label: string;
  visible: boolean;
}

export interface EntryProps {
  allowVisibleChange?: boolean;
  background?: Theming.Shade;
  entry: EntryData;
  onEntryChange?: (value: EntryData) => void;
  onVisibleChange?: state.Setter<boolean>;
}

export const Entry = ({
  allowVisibleChange = true,
  background,
  entry,
  onEntryChange,
  onVisibleChange,
}: EntryProps): ReactElement => {
  const { color, key, label, visible } = entry;
  return (
    <Flex.Box
      align="center"
      className={CSS.B("legend-entry")}
      gap="small"
      justify="between"
      key={key}
      x
    >
      <Flex.Box align="center" gap="small" x>
        <Color.Swatch
          allowChange={onEntryChange != null}
          draggable={false}
          onChange={(c) => onEntryChange?.({ ...entry, color: c })}
          onVisibleChange={onVisibleChange}
          size="tiny"
          value={color}
        />
        <Text.MaybeEditable
          color={entry.visible ? 10 : 7}
          level="small"
          onChange={(l) => onEntryChange?.({ ...entry, label: l })}
          onDoubleClick={stopPropagation}
          overflow="nowrap"
          value={label}
        />
      </Flex.Box>
      {allowVisibleChange && (
        <Button.Button
          className={CSS.B("visible-toggle")}
          contrast={background}
          onClick={(e) => {
            e.stopPropagation();
            onEntryChange?.({ ...entry, visible: !visible });
          }}
          onDoubleClick={stopPropagation}
          size="tiny"
          variant="text"
        >
          {visible ? <Icon.Visible /> : <Icon.Hidden />}
        </Button.Button>
      )}
    </Flex.Box>
  );
};
