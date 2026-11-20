// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/legend/Grouped.css";

import { Badge } from "@synnaxlabs/lyra/badge";
import { Color } from "@synnaxlabs/lyra/color";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import type { color } from "@synnaxlabs/x";
import { Fragment, type ReactElement, type ReactNode, useState } from "react";

import { CSS } from "@/css";
import { Container, type ContainerProps } from "@/vis/legend/Container";
import {
  Entries,
  type EntriesProps,
  useHover,
  VisibleButton,
} from "@/vis/legend/Entries";

/** A short note shown next to the name of a group, with a tooltip that explains it. */
export interface GroupBadge {
  label: string;
  tooltip: string;
}

export interface GroupData {
  key: string;
  name: string;
  data: EntriesProps["data"];
  /** The color of every entry of the group. A group with a color has a swatch. */
  color?: color.Crude;
  /** Whether the group shows. A group with a visibility has a toggle. */
  visible?: boolean;
  badge?: GroupBadge;
  /** Shown in place of the entries. */
  note?: string;
}

export interface GroupedProps
  extends
    Omit<ContainerProps, "value" | "onChange" | "background" | "draggable" | "gap">,
    Pick<
      EntriesProps,
      | "background"
      | "allowVisibleChange"
      | "onColorChange"
      | "onLabelChange"
      | "onVisibleChange"
      | "onHoverChange"
      | "sorted"
    > {
  data: GroupData[];
  position?: ContainerProps["value"];
  onPositionChange?: ContainerProps["onChange"];
  onGroupColorChange?: (key: string, color: color.Color) => void;
  /** Called like onVisibleChange, with the key of a group. */
  onGroupVisibleChange?: EntriesProps["onVisibleChange"];
  /** Shown under the groups. */
  footer?: ReactNode;
  /** Lists the entries of each group under its name instead of beside it. */
  nested?: boolean;
}

export const Grouped = ({
  data,
  background = 1,
  allowVisibleChange = true,
  onColorChange,
  onLabelChange,
  onVisibleChange,
  onHoverChange,
  onGroupColorChange,
  onGroupVisibleChange,
  sorted,
  footer,
  position,
  onPositionChange,
  nested = false,
  ...rest
}: GroupedProps): ReactElement | null => {
  const [pickerVisible, setPickerVisible] = useState(false);
  if (data.length === 0) return null;
  return (
    <Container
      {...rest}
      draggable={!pickerVisible}
      value={position}
      onChange={onPositionChange}
      gap="small"
      background={background}
      className={CSS.cls(
        CSS.B("legend-grouped"),
        nested && CSS.BM("legend-grouped", "nested"),
      )}
    >
      {data.map((group, i) => {
        const entries = (
          <Flex.Box y grow className={CSS.B("legend-entries")} empty>
            {group.note != null && (
              <Text.Text level="small" color={7} className={CSS.B("legend-note")}>
                {group.note}
              </Text.Text>
            )}
            <Entries
              data={group.data}
              onColorChange={onColorChange}
              onLabelChange={onLabelChange}
              onVisibleChange={onVisibleChange}
              onHoverChange={onHoverChange}
              sorted={sorted}
              colorPickerVisible={pickerVisible}
              onColorPickerVisibleChange={setPickerVisible}
              allowVisibleChange={allowVisibleChange}
              background={background}
              entryProps={entryProps}
              shade={nested ? 9 : 10}
            />
          </Flex.Box>
        );
        const name = (
          <GroupName
            group={group}
            onColorChange={onGroupColorChange}
            onVisibleChange={onGroupVisibleChange}
            onHoverChange={onHoverChange}
            onColorPickerVisibleChange={setPickerVisible}
          />
        );
        if (nested)
          return (
            <Flex.Box key={group.key} y empty className={CSS.B("legend-group")}>
              {name}
              {entries}
            </Flex.Box>
          );
        return (
          <Fragment key={group.key}>
            {name}
            {entries}
            {i !== data.length - 1 && (
              <Divider.Divider x padded className={CSS.B("legend-divider")} />
            )}
          </Fragment>
        );
      })}
      {footer != null && <div className={CSS.B("legend-footer")}>{footer}</div>}
    </Container>
  );
};

interface GroupNameProps {
  group: GroupData;
  onColorChange?: GroupedProps["onGroupColorChange"];
  onVisibleChange?: GroupedProps["onGroupVisibleChange"];
  onHoverChange?: GroupedProps["onHoverChange"];
  onColorPickerVisibleChange: EntriesProps["onColorPickerVisibleChange"];
}

const GroupName = ({
  group: { key, name, color, visible, badge },
  onColorChange,
  onVisibleChange,
  onHoverChange,
  onColorPickerVisibleChange,
}: GroupNameProps): ReactElement => {
  const hover = useHover(key, onHoverChange);
  if (color == null && visible == null)
    return (
      <Text.Text level="small" color={9} weight={500} className={CSS.B("legend-name")}>
        {name}
      </Text.Text>
    );
  return (
    <Flex.Box
      x
      align="center"
      gap="small"
      className={CSS.cls(CSS.B("legend-name"), Menu.CONTEXT_TARGET)}
      data-menu-key={key}
      {...hover}
    >
      <Flex.Box x align="center" gap="small">
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
        <Text.Text
          level="small"
          color={visible === false ? 7 : 10}
          weight={500}
          overflow="nowrap"
        >
          {name}
        </Text.Text>
        {badge != null && (
          <Tooltip.Dialog location={{ x: "center", y: "top" }}>
            {badge.tooltip}
            <Badge.Badge>{badge.label}</Badge.Badge>
          </Tooltip.Dialog>
        )}
      </Flex.Box>
      {visible != null && onVisibleChange != null && (
        <VisibleButton itemKey={key} visible={visible} onChange={onVisibleChange} />
      )}
    </Flex.Box>
  );
};

const entryProps = { justify: "between", grow: true } as const;
