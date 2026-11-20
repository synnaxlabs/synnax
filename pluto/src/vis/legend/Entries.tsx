// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import type { Theming } from "@synnaxlabs/lyra/theming";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { type color, compare, type optional, type state } from "@synnaxlabs/x";
import {
  type HTMLAttributes,
  memo,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
} from "react";

import { CSS } from "@/css";

export interface EntryData {
  /** The color of the swatch. An entry with no color has no swatch. */
  color?: color.Crude;
  key: string;
  label: string;
  visible: boolean;
  /** Shown before the label, in a column as wide as a swatch. */
  start?: ReactNode;
}

export interface EntriesProps {
  allowVisibleChange?: boolean;
  background?: Theming.Shade;
  data: optional.Optional<EntryData, "visible">[];
  onColorChange?: (key: string, color: color.Color) => void;
  onLabelChange?: (key: string, label: string) => void;
  /**
   * Called when the user toggles the visibility of an entry. solo is true when the user
   * holds Alt to show only that entry.
   */
  onVisibleChange?: (key: string, visible: boolean, solo: boolean) => void;
  /** Called with the key of the entry the pointer enters, and null when it leaves. */
  onHoverChange?: (key: string | null) => void;
  /** Sorts the entries by label. Otherwise they keep the order of data. */
  sorted?: boolean;
  /** The shade of the label of a visible entry. */
  shade?: Theming.Shade;
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
    sorted = true,
    ...rest
  }: EntriesProps): ReactElement => (
    <>
      {(sorted ? [...data].sort(compare.by((e) => e.label)) : data).map(
        ({ key, visible = true, ...entryRest }) => (
          <Entry
            key={key}
            entry={{ key, visible, ...entryRest }}
            allowVisibleChange={allowVisibleChange}
            background={background}
            {...entryProps}
            {...rest}
          />
        ),
      )}
    </>
  ),
);
Entries.displayName = "Legend.Entries";

interface EntryProps
  extends Omit<EntriesProps, "data">, Omit<Flex.BoxProps, "background"> {
  entry: EntryData;
}

type HoverHandlers = Pick<
  HTMLAttributes<HTMLElement>,
  "onPointerEnter" | "onPointerLeave"
>;

/**
 * @returns the pointer handlers that report the hover of the legend entry with key. An
 * entry that unmounts under the pointer gets no pointerleave, so it ends its hover.
 */
export const useHover = (
  key: string,
  onHoverChange?: (key: string | null) => void,
): HoverHandlers => {
  const hovered = useRef(false);
  const onHoverChangeRef = useSyncedRef(onHoverChange);
  useEffect(
    () => () => {
      if (hovered.current) onHoverChangeRef.current?.(null);
    },
    [onHoverChangeRef],
  );
  if (onHoverChange == null) return {};
  return {
    onPointerEnter: () => {
      hovered.current = true;
      onHoverChange(key);
    },
    onPointerLeave: () => {
      hovered.current = false;
      onHoverChange(null);
    },
  };
};

const Entry = ({
  allowVisibleChange = true,
  entry,
  onColorChange,
  onLabelChange,
  onVisibleChange,
  onHoverChange,
  sorted: _,
  shade = 10,
  colorPickerVisible,
  onColorPickerVisibleChange,
  className,
  background,
  ...rest
}: EntryProps): ReactElement => {
  const { color, key, label, visible, start } = entry;
  const hover = useHover(key, onHoverChange);
  return (
    <Flex.Box
      align="center"
      className={CSS.cls(CSS.B("legend-entry"), Menu.CONTEXT_TARGET, className)}
      data-menu-key={key}
      gap="small"
      key={key}
      x
      {...hover}
      {...rest}
    >
      <Flex.Box align="center" gap="small" x>
        {color != null && (
          <Color.Swatch
            draggable={false}
            onChange={onColorChange == null ? undefined : (c) => onColorChange(key, c)}
            size="tiny"
            variant="text"
            value={color}
            onVisibleChange={onColorPickerVisibleChange}
          />
        )}
        {start != null && (
          <span className={CSS.BE("legend-entry", "start")}>{start}</span>
        )}
        <Text.MaybeEditable
          color={entry.visible ? shade : 7}
          className={CSS.BE("legend-entry", "label")}
          level="small"
          onChange={onLabelChange == null ? undefined : (l) => onLabelChange(key, l)}
          onDoubleClick={stopPropagation}
          overflow="nowrap"
          value={label}
        />
      </Flex.Box>
      {allowVisibleChange && onVisibleChange != null && (
        <VisibleButton itemKey={key} visible={visible} onChange={onVisibleChange} />
      )}
    </Flex.Box>
  );
};

export interface VisibleButtonProps {
  itemKey: string;
  visible: boolean;
  onChange: NonNullable<EntriesProps["onVisibleChange"]>;
}

/** An eye that hides or shows an item. An Option or Alt click shows only that item. */
export const VisibleButton = ({
  itemKey,
  visible,
  onChange,
}: VisibleButtonProps): ReactElement => (
  <Button.Button
    className={CSS.B("visible-toggle")}
    aria-label={visible ? "Hide" : "Show"}
    onClick={(e) => onChange(itemKey, !visible, e.altKey)}
    onDoubleClick={stopPropagation}
    size="tiny"
    variant="glyph"
  >
    {visible ? <Icon.Visible /> : <Icon.Hidden />}
  </Button.Button>
);
