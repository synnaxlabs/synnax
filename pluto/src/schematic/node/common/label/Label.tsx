// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  type CSSProperties,
  type FC,
  memo,
  type ReactElement,
  useCallback,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { type Primitive } from "@/schematic/node/common/primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Text } from "@/text";

export const configZ = schematic.labelConfigZ;
export type Config = schematic.LabelConfig;

export interface LabelProps {
  config: Config;
  onChange?: (next: { label: Config }) => void;
}

interface InternalProps
  extends LabelProps, Omit<Text.EditableProps, "value" | "onChange"> {
  style?: CSSProperties;
}

const Internal = ({ config, onChange, style: baseStyle, ...rest }: InternalProps) => {
  const { label = "", level = "p", direction: dir, align, maxInlineSize } = config;
  const handleLabelChange = useCallback(
    (value: string) => onChange?.({ label: { ...config, label: value } }),
    [config],
  );
  const style = useMemo(
    () => ({
      textAlign: align as CSSProperties["textAlign"],
      maxInlineSize,
      ...baseStyle,
    }),
    [align, maxInlineSize, baseStyle],
  );
  return (
    <Text.Editable
      {...rest}
      style={style}
      alignSelf={align}
      className={CSS(CSS.BE("symbol", "label"), CSS.dir(dir))}
      level={level}
      value={label}
      onChange={handleLabelChange}
      allowEmpty
    />
  );
};

export const Label = Grid.createItem(({ config, onChange }: Partial<LabelProps>) => {
  if (config == null || config.label == null || config.label.length == 0) return null;
  const orientation = config.orientation ?? "top";
  return (
    <Grid.Item
      itemKey="label"
      location={orientation}
      onLocationChange={(loc) => onChange?.({ label: { ...config, orientation: loc } })}
    >
      <Internal config={config} onChange={onChange} />
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

export const labeledConfigZ = schematic.labeledConfigZ;
export type LabeledConfig = schematic.LabeledConfig;

interface LabeledOverrides {
  grid: Partial<Omit<Grid.GridProps, "editable">>;
}

export const createLabeled = <C extends LabeledConfig>(
  BaseSymbol: FC<Omit<C, "label"> & Primitive.SVGBasedProps>,
  overrides?: LabeledOverrides,
): FC<NodeProps<C>> => {
  // BaseSymbol's prop type is derived from C so callers are checked, but the node only
  // renders it with the shared SVG props; the config's symbol-specific fields reach it
  // at runtime via the rest spread.
  const Sym = BaseSymbol as FC<Primitive.SVGBasedProps>;
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
      <Sym orientation={orientation} {...rest} />
    </Grid.Grid>
  );
  const M = memo(Inner) as FC<NodeProps<C>>;
  M.displayName = BaseSymbol.displayName;
  return M;
};
