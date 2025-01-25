// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/legend/Container.css";

import { type ReactElement, memo } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Legend as Core } from "@/vis/legend";
import { type LineSpec, useContext, useGridEntry } from "@/vis/lineplot/LinePlot";

export interface LegendProps extends Omit<Core.SimpleProps, "data" | "onEntryChange"> {
  variant?: "floating" | "fixed";
  onLineChange?: (line: LineSpec) => void;
}

export const Floating = memo(
  ({ className, style, onLineChange, ...props }: LegendProps): ReactElement | null => {
    const { lines } = useContext("Legend");
    useContext("Legend");
    return <Core.Simple data={lines} onEntryChange={onLineChange} {...props} />;
  },
);

const Fixed = ({ onLineChange }: LegendProps) => {
  const { lines } = useContext("Legend");
  const key = useUniqueKey();
  const gridStyle = useGridEntry(
    {
      key,
      size: lines.length > 0 ? 36 : 0,
      loc: "top",
      order: 5,
    },
    "Legend",
  );

  return (
    <Align.Space
      className={CSS.BE("legend", "container")}
      align="center"
      direction="x"
      size="large"
      style={{
        ...gridStyle,
        padding: "0 1rem",
        height: "var(--pluto-height-medium)",
        margin: "1rem 0 ",
      }}
    >
      {Core.legendSwatches(lines, onLineChange)}
    </Align.Space>
  );
};

export const Legend = ({
  variant = "floating",
  ...props
}: LegendProps): ReactElement | null =>
  variant === "floating" ? <Floating {...props} /> : <Fixed {...props} />;
