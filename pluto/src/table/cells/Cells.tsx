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
import { box, color, scale } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Menu } from "@/menu";
import { Cell as Base } from "@/table/cells/Cell";
import { telem } from "@/telem/aether";
import { Text as BaseText } from "@/text";
import { Value as BaseValue } from "@/vis/value";

export const textConfigZ = table.cellConfigTextZ;
export type TextConfig = table.CellConfigText;

export type CellProps<C extends table.CellConfig = table.CellConfig> = C & {
  cellKey: string;
  box: box.Box;
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
  backgroundColor,
}: CellProps<TextConfig>): ReactElement => {
  const handleSelect = (e: React.MouseEvent) => onSelect(cellKey, e);
  const handleValueChange = (value: string) =>
    onChange({ variant: "text", value, level, weight, align, backgroundColor });
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
        backgroundColor:
          backgroundColor == null ? undefined : color.cssString(backgroundColor),
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

export const valueConfigZ = table.cellConfigValueZ;
export type ValueConfig = table.CellConfigValue;

export const Value = ({
  cellKey,
  channel,
  rollingAverage,
  precision,
  notation,
  level = "h5",
  color: textColor,
  redline,
  selected,
  box: b,
  onSelect,
  stalenessTimeout = 5,
  stalenessColor,
}: CellProps<ValueConfig>) => {
  const t = useMemo(
    () => BaseValue.stringSource({ channel, rollingAverage, precision, notation }),
    [channel, rollingAverage, precision, notation],
  );
  const backgroundTelem = useMemo(() => {
    if (redline == null) return undefined;
    const { bounds, gradient } = redline;
    return telem.sourcePipeline("color", {
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
    });
  }, [t, redline]);
  BaseValue.use({
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
