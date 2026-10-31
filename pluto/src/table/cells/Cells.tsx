// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/cells/Cells.css";

import { table } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text as BaseText } from "@synnaxlabs/lyra/text";
import { type border, box, color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Cell as Base } from "@/table/cells/Cell";
import { Problem } from "@/vis/problem";
import { Value as BaseValue } from "@/vis/value";

export const textConfigZ = table.textCellConfigZ;
export type TextConfig = table.TextCellConfig;

export type CellProps<C extends table.CellConfig = table.CellConfig> = C & {
  cellKey: string;
  box: box.Box;
  /**
   * Rounding of the cell's corners in px. Only cells at the table's outer corners
   * round.
   */
  borderRadius?: border.CrudeRadius;
  selected: boolean;
  editable: boolean;
  onSelect: (key: string, ev: React.MouseEvent) => void;
  onChange: (config: C) => void;
};

export const Text = ({
  cellKey,
  onChange,
  value = "",
  selected,
  editable,
  onSelect,
  box: b,
  align = "center",
  level = "h5",
  weight = 400,
  fillColor,
  textColor,
}: CellProps<TextConfig>): ReactElement => {
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);
  const handleValueChange = (value: string) =>
    onChange({ variant: "text", value, level, weight, align, fillColor, textColor });
  const cellStyle = {
    backgroundColor: color.cssString(fillColor),
    width: box.width(b),
  };
  const editableStyle = { justifyContent: align };
  return (
    <Base
      id={cellKey}
      className={CSS.cls(
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.BEM("table", "cell", "text"),
      )}
      selected={selected}
      height={box.height(b)}
      onClick={handleSelect}
      onContextMenu={handleSelect}
      style={cellStyle}
    >
      <BaseText.Editable
        level={level}
        value={value}
        weight={weight}
        color={textColor == null ? undefined : color.cssString(textColor)}
        onChange={handleValueChange}
        style={editableStyle}
        allowDoubleClick={editable}
        allowEmpty
        outline={false}
      />
    </Base>
  );
};

export const valueConfigZ = table.valueCellConfigZ;
export type ValueConfig = table.ValueCellConfig;

export const Value = ({
  cellKey,
  channel,
  rollingAverage,
  precision,
  notation,
  borderRadius,
  level = "h5",
  textColor,
  redline,
  fillColor,
  selected,
  box: b,
  onSelect,
  stalenessTimeout = 5,
  stalenessColor,
}: CellProps<ValueConfig>) => {
  const t = BaseValue.stringSource({ channel, rollingAverage, precision, notation });
  const backgroundTelem = BaseValue.backgroundTelem(t, redline, fillColor);
  const { problems } = BaseValue.use({
    aetherKey: cellKey,
    box: b,
    telem: t,
    level,
    color: textColor,
    stalenessTimeout,
    stalenessColor,
    backgroundTelem,
    location: { x: "center", y: "center" },
    clip: true,
    borderRadius,
  });
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);
  // Use the column-driven box width, not BaseValue's natural text width: when
  // row indicators are hidden, the first data row determines column widths via
  // table-layout: fixed, so the cell must be locked to the stored column size
  // or canvas/DOM alignment drifts.
  const cellStyle = { width: box.width(b) };

  return (
    <Base
      id={cellKey}
      selected={selected}
      height={box.height(b)}
      onClick={handleSelect}
      onContextMenu={handleSelect}
      style={cellStyle}
      className={CSS.cls(
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.BEM("table", "cell", "value"),
      )}
    >
      <Problem.Mark problems={problems} variant="glyph" />
    </Base>
  );
};
