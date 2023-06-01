// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, memo } from "react";

import { Optional, ZERO_XY } from "@synnaxlabs/x";

import { useAxisPosition } from "./LinePlot";

import { Bob } from "@/core/bob/main";
import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import {
  YAxisState as WorkerYAxisState,
  YAxis as WorkerYAxis,
} from "@/core/vis/LinePlot/worker";

export interface YAxisProps
  extends PropsWithChildren,
    Optional<Omit<WorkerYAxisState, "position">, "color" | "font"> {}

export const YAxis = memo(
  ({ children, location = "left", ...props }: YAxisProps): ReactElement => {
    const theme = Theming.use();
    const font = `${theme.typography.tiny.size * theme.sizes.base}px ${
      theme.typography.family
    }`;
    const {
      key,
      path,
      state: [, setState],
    } = Bob.useComponent<WorkerYAxisState>(WorkerYAxis.TYPE, {
      position: ZERO_XY,
      color: theme.colors.gray.p2,
      location,
      font,
      ...props,
    });
    const gridStyle = useAxisPosition(location, key);
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
      <Bob.Composite path={path}>
        <div className="y-axis" style={gridStyle} ref={resizeRef}></div>
        {children}
      </Bob.Composite>
    );
  }
);
YAxis.displayName = "YAxisC";
