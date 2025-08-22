// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/legend/Container.css";

import { memo, type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Legend as Core } from "@/vis/legend";
import { LegendSwatches } from "@/vis/legend/Simple";
import { type LineSpec, useContext, useGridEntry } from "@/vis/lineplot/LinePlot";

export interface LegendProps extends Omit<Core.SimpleProps, "data" | "onEntryChange"> {
  variant?: "floating" | "fixed";
  onLineChange?: (line: LineSpec) => void;
}

export const Floating = memo(
  ({ className, style, onLineChange, ...rest }: LegendProps): ReactElement | null => {
    const { lines } = useContext("Legend");
    useContext("Legend");
    return <Core.Simple data={lines} onEntryChange={onLineChange} {...rest} />;
  },
);
Floating.displayName = "FloatingLegend";

const Fixed = ({ onLineChange }: LegendProps) => {
  const { lines } = useContext("Legend");
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    { key, size: lines.length > 0 ? 36 : 0, loc: "top", order: 5 },
    "Legend",
  );

  return (
    <Flex.Box
      className={CSS(CSS.BE("legend", "container"), CSS.B("legend", "fixed"))}
      align="center"
      justify="start"
      x
      style={gridStyle}
    >
      <LegendSwatches data={lines} onEntryChange={onLineChange} background={0} />
    </Flex.Box>
  );
};

export const Legend = ({
  variant = "floating",
  ...rest
}: LegendProps): ReactElement | null =>
  variant === "floating" ? <Floating {...rest} /> : <Fixed {...rest} />;
