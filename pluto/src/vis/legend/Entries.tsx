// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type optional } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

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

export interface EntriesProps {
  allowVisibleChange?: boolean;
  background?: Theming.Shade;
  data: optional.Optional<EntryData, "visible">[];
  onEntryChange?: (value: EntryData) => void;
  colorPickerVisible?: boolean;
  onColorPickerVisibleChange?: state.Setter<boolean>;
  entryProps?: Omit<Flex.BoxProps, "background">;
}

export const Entries = memo(
  ({
    data,
    allowVisibleChange = true,
    background = 1,
    entryProps,
    ...rest
  }: EntriesProps): ReactElement => (
    <>
      {data
        .sort((a, b) => a.label.localeCompare(b.label))
        .map(({ key, visible = true, ...entryRest }) => (
          <Entry
            key={key}
            entry={{ key, visible, ...entryRest }}
            allowVisibleChange={allowVisibleChange}
            background={background}
            {...entryProps}
            {...rest}
          />
        ))}
    </>
  ),
);
Entries.displayName = "Legend.Entries";

interface EntryProps
  extends Omit<EntriesProps, "data">,
    Omit<Flex.BoxProps, "background"> {
  entry: EntryData;
}

const Entry = ({
  allowVisibleChange = true,
  entry,
  onEntryChange,
  colorPickerVisible,
  onColorPickerVisibleChange,
  className,
  background,
  ...rest
}: EntryProps): ReactElement => {
  const { color, key, label, visible } = entry;
  return (
    <Flex.Box
      align="center"
      className={CSS(CSS.B("legend-entry"), className)}
      gap="small"
      key={key}
      x
      {...rest}
    >
      <Flex.Box align="center" gap="small" x>
        <Color.Swatch
          allowChange={onEntryChange != null}
          draggable={false}
          onChange={(c) => onEntryChange?.({ ...entry, color: c })}
          size="tiny"
          value={color}
          onVisibleChange={onColorPickerVisibleChange}
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
          onClick={() => {
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
