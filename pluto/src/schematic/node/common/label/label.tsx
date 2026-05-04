// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction, location } from "@synnaxlabs/x";
import { type CSSProperties, type FC, memo, type ReactElement } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Grid } from "@/schematic/node/common/grid";
import { type NodeProps } from "@/schematic/node/spec";
import { Text } from "@/text";
import { text } from "@/text/base";

export const configZ = z.object({
  label: z.string().optional(),
  level: text.levelZ.optional(),
  orientation: location.locationZ.optional(),
  direction: direction.directionZ.optional(),
  maxInlineSize: z.number().optional(),
  align: Flex.alignmentZ.optional(),
});
export type Config = z.infer<typeof configZ>;

export const gridItem = (
  props?: Config,
  onChange?: ({ label }: { label: Config }) => void,
): Grid.Item | null => {
  if (props == null) return null;
  const {
    label,
    level = "p",
    orientation = "top",
    direction,
    align,
    maxInlineSize,
  } = props;
  if (label == null || label.length === 0) return null;
  return {
    key: "label",
    element: (
      <Text.Editable
        className={CSS(CSS.BE("symbol", "label"), CSS.dir(direction))}
        level={level}
        value={label}
        onChange={(value: string) => onChange?.({ label: { ...props, label: value } })}
        allowEmpty
        style={{ textAlign: align as CSSProperties["textAlign"], maxInlineSize }}
      />
    ),
    location: orientation,
  };
};

export const defaultConfig = (label: string): Config => ({
  label,
  level: "small",
  orientation: "top",
  maxInlineSize: 150,
  align: "center",
  direction: "x",
});

export const labeledConfigZ = z.object({
  label: configZ.optional(),
  orientation: location.outerZ.optional(),
});
export type LabeledConfig = z.infer<typeof labeledConfigZ>;

interface LabeledOverrides {
  grid: Partial<Omit<Grid.GridProps, "editable">>;
}

export const createLabeled = <Config extends LabeledConfig>(
  BaseSymbol: FC<any>,
  overrides?: LabeledOverrides,
): FC<NodeProps<Config>> => {
  const C = ({
    nodeKey: symbolKey,
    onConfigChange: onChange,
    selected,
    config: data,
  }: NodeProps<Config>): ReactElement => {
    const { label, orientation = "left", ...rest } = data;
    const gridItems: Grid.Item[] = [];
    const labelItem = gridItem(label, onChange as never);
    if (labelItem != null) gridItems.push(labelItem);
    return (
      <Grid.Grid
        {...overrides?.grid}
        items={gridItems}
        editable={selected}
        symbolKey={symbolKey}
        onRotate={() =>
          onChange({
            orientation: location.rotate(orientation, "clockwise"),
          } as Partial<Config>)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({
              label: { ...label, orientation: loc },
            } as Partial<Config>);
        }}
      >
        <BaseSymbol orientation={orientation} {...rest} />
      </Grid.Grid>
    );
  };
  const M = memo(C);
  M.displayName = BaseSymbol.displayName;
  return M;
};
