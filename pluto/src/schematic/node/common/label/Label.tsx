// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction, location } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type FC,
  memo,
  type ReactElement,
  useCallback,
  useMemo,
} from "react";
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

export interface LabelProps {
  config?: Config;
  onChange?: (next: { label: Config }) => void;
}

export const Label = Grid.createItem<LabelProps>(({ config, onChange }) => {
  if (config == null) return null;
  const {
    label,
    level = "p",
    orientation = "top",
    direction: dir,
    align,
    maxInlineSize,
  } = config;
  if (label == null || label.length === 0) return null;
  const handleLabelChange = useCallback(
    (value: string) => onChange?.({ label: { ...config, label: value } }),
    [config],
  );
  const handleLocationChange = useCallback(
    (orientation: location.Location) =>
      onChange?.({ label: { ...config, orientation } }),
    [config],
  );
  const style = useMemo(
    () => ({
      textAlign: align as CSSProperties["textAlign"],
      maxInlineSize,
    }),
    [align, maxInlineSize],
  );
  return (
    <Grid.Item
      itemKey="label"
      location={orientation}
      onLocationChange={handleLocationChange}
    >
      <Text.Editable
        className={CSS(CSS.BE("symbol", "label"), CSS.dir(dir))}
        level={level}
        value={label}
        onChange={handleLabelChange}
        allowEmpty
        style={style}
      />
    </Grid.Item>
  );
});
Label.displayName = "Label.GridItem";

export const defaultConfig = (label: string): Config => ({
  label,
  level: "h5",
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

export const createLabeled = <C extends LabeledConfig>(
  BaseSymbol: FC<any>,
  overrides?: LabeledOverrides,
): FC<NodeProps<C>> => {
  const Inner = ({
    nodeKey,
    onConfigChange,
    selected,
    config: { label, orientation = "left", ...rest },
  }: NodeProps<LabeledConfig>): ReactElement => (
    <Grid.Grid
      {...overrides?.grid}
      editable={selected}
      nodeKey={nodeKey}
      orientation={orientation}
      onRotate={onConfigChange}
    >
      <Label config={label} onChange={onConfigChange} />
      <BaseSymbol orientation={orientation} {...rest} />
    </Grid.Grid>
  );
  const M = memo(Inner) as unknown as FC<NodeProps<C>>;
  M.displayName = BaseSymbol.displayName;
  return M;
};
