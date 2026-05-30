// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/cells/Cells.css";

import { box, color, location, type record, scale, text } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Menu } from "@/menu";
import { Cell as Base } from "@/table/cells/Cell";
import { telem } from "@/telem/aether";
import { Text as BaseText } from "@/text";
import { Value as BaseValue } from "@/vis/value";

export const TEXT_TYPE = "text";
export type TextType = typeof TEXT_TYPE;
export const textPropsZ = z.object({
  value: z.string(),
  level: text.levelZ,
  weight: text.weightZ,
  align: location.xZ.or(location.centerZ),
  backgroundColor: color.crudeZ,
});
export type TextProps = z.infer<typeof textPropsZ>;

export type CellProps<P extends object = record.Unknown> = P & {
  cellKey: string;
  box: box.Box;
  selected: boolean;
  editable: boolean;
  onSelect: (key: string, ev: React.MouseEvent) => void;
  onChange: (props: P) => void;
};

export const Text = ({
  cellKey,
  onChange,
  value,
  selected,
  editable,
  onSelect,
  box: b,
  align,
  level,
  weight,
  backgroundColor,
}: CellProps<TextProps>): ReactElement => {
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);
  const handleValueChange = (value: string) =>
    onChange({ level, value, weight, align, backgroundColor });
  return (
    <Base
      id={cellKey}
      className={CSS(
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.BEM("table", "cell", "text"),
      )}
      selected={selected}
      height={box.height(b)}
      onClick={handleSelect}
      onContextMenu={handleSelect}
      style={{
        backgroundColor: color.cssString(backgroundColor),
        width: box.width(b),
      }}
    >
      <BaseText.Editable
        level={level}
        value={value}
        weight={weight}
        onChange={handleValueChange}
        style={{ justifyContent: align }}
        allowDoubleClick={editable}
        allowEmpty
        outline={false}
      />
    </Base>
  );
};

export const VALUE_TYPE = "value";
export type ValueType = typeof VALUE_TYPE;
export const valuePropsZ = z.object({
  telem: telem.stringSourceSpecZ,
  redline: BaseValue.redlineZ,
  level: text.levelZ,
  color: z.string(),
  units: z.string(),
  stalenessTimeout: z.number().default(5),
  stalenessColor: color.colorZ.default(color.ZERO),
});
export type ValueProps = z.infer<typeof valuePropsZ>;

export const Value = ({
  cellKey,
  telem: t,
  level,
  color,
  redline: { gradient, bounds },
  selected,
  box: b,
  onSelect,
  stalenessTimeout,
  stalenessColor,
}: CellProps<ValueProps>) => {
  BaseValue.use({
    aetherKey: cellKey,
    box: b,
    telem: t,
    level,
    color,
    stalenessTimeout,
    stalenessColor,
    backgroundTelem: telem.sourcePipeline("color", {
      connections: [
        { from: "source", to: "scale" },
        { from: "scale", to: "gradient" },
      ],
      segments: {
        source: t,
        scale: telem.scaleNumber({
          scale: scale.Scale.scale<number>(bounds).scale(0, 1).transform,
        }),
        gradient: telem.colorGradient({ gradient }),
      },
      outlet: "gradient",
    }),
    location: { x: "center", y: "center" },
    clip: true,
  });
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);

  return (
    <Base
      id={cellKey}
      selected={selected}
      height={box.height(b)}
      onClick={handleSelect}
      onContextMenu={handleSelect}
      // Use the column-driven box width, not BaseValue's natural text width:
      // when row indicators are hidden, the first data row determines column
      // widths via table-layout: fixed, so the cell must be locked to the
      // stored column size or canvas/DOM alignment drifts.
      style={{ width: box.width(b) }}
      className={CSS(
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.BEM("table", "cell", "value"),
      )}
    />
  );
};
