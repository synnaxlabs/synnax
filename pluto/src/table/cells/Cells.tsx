// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/cells/Cells.css";

import { type box, color, location, type record, scale } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Menu } from "@/menu";
import { Cell as Core } from "@/table/Table";
import { telem } from "@/telem/aether";
import { Text as CoreText } from "@/text";
import { Value as CoreValue } from "@/vis/value";

export const TEXT_TYPE = "text";
export type TextType = typeof TEXT_TYPE;
export const textPropsZ = z.object({
  value: z.string(),
  level: CoreText.levelZ,
  weight: CoreText.weightZ,
  align: location.x.or(location.center),
  backgroundColor: color.crudeZ,
});
export type TextProps = z.infer<typeof textPropsZ>;

export type CellProps<P extends object = record.Unknown> = P & {
  cellKey: string;
  box: box.Box;
  selected: boolean;
  onSelect: (key: string, ev: React.MouseEvent) => void;
  onChange: (props: P) => void;
};

export const Text = ({
  cellKey,
  onChange,
  value,
  selected,
  onSelect,
  align,
  level,
  weight,
  backgroundColor,
}: CellProps<TextProps>): ReactElement => {
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);
  const handleValueChange = (value: string) =>
    onChange({ level, value, weight, align, backgroundColor });
  return (
    <Core
      id={cellKey}
      className={CSS(
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.BEM("table", "cell", "text"),
      )}
      selected={selected}
      onClick={handleSelect}
      onContextMenu={handleSelect}
      style={{ backgroundColor: color.cssString(backgroundColor) }}
    >
      <CoreText.Editable
        level={level}
        value={value}
        weight={weight}
        onChange={handleValueChange}
        style={{ justifyContent: align }}
        allowEmpty
        outline={false}
      />
    </Core>
  );
};

export const VALUE_TYPE = "value";
export type ValueType = typeof VALUE_TYPE;
export const valuePropsZ = z.object({
  telem: telem.stringSourceSpecZ,
  redline: CoreValue.redlineZ,
  level: CoreText.levelZ,
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
  const { width } = CoreValue.use({
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
  });
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);

  return (
    <Core
      id={cellKey}
      selected={selected}
      onClick={handleSelect}
      onContextMenu={handleSelect}
      style={{ height: "5rem", width }}
      className={CSS(
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.BEM("table", "cell", "value"),
      )}
    />
  );
};
