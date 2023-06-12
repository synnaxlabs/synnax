// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, memo } from "react";

import { Optional, OuterLocationT, XY, Location } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import { useAxisPosition } from "@/core/vis/LinePlot/main/LinePlot";
import {
  YAxisState as WorkerYAxisState,
  YAxis as WorkerYAxis,
} from "@/core/vis/LinePlot/worker";

export interface YAxisProps
  extends PropsWithChildren,
    Optional<Omit<WorkerYAxisState, "position">, "color" | "font" | "gridColor"> {}

export const YAxis = memo(
  ({ children, location = "left", ...props }: YAxisProps): ReactElement => {
    const theme = Theming.use();
    const {
      key,
      path,
      state: [, setState],
    } = Aether.use<WorkerYAxisState>(WorkerYAxis.TYPE, {
      position: XY.ZERO,
      color: theme.colors.gray.p2,
      gridColor: theme.colors.gray.m1,
      location,
      font: Theming.font(theme, "small"),
      ...props,
    });
    const gridStyle = useAxisPosition(
      new Location(location).v as OuterLocationT,
      key,
      "YAxis"
    );
    const resizeRef = useResize(
      (box) => {
        setState((state) => ({
          ...state,
          position: box.topLeft,
        }));
      },
      { debounce: 100 }
    );
    return (
      <Aether.Composite path={path}>
        <div className="y-axis" style={gridStyle} ref={resizeRef}></div>
        {children}
      </Aether.Composite>
    );
  }
);
YAxis.displayName = "YAxisC";
