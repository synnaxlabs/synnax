// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, memo } from "react";

import { Optional, XY } from "@synnaxlabs/x";

import { useAxisPosition } from "./LinePlot";

import { Bob } from "@/core/bob/main";
import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import {
  XAxisState as WorkerXAxisState,
  XAxis as WorkerXAxis,
} from "@/core/vis/LinePlot/worker";

export interface XAxisCProps
  extends PropsWithChildren,
    Optional<Omit<WorkerXAxisState, "position">, "color" | "font" | "gridColor"> {}

export const XAxis = memo(
  ({ children, location = "bottom", ...props }: XAxisCProps): ReactElement => {
    const theme = Theming.use();
    const font = `${theme.typography.tiny.size * theme.sizes.base}px ${
      theme.typography.family
    }`;
    const {
      key,
      path,
      state: [, setState],
    } = Bob.useComponent<WorkerXAxisState>(WorkerXAxis.TYPE, {
      color: theme.colors.gray.p3,
      gridColor: theme.colors.gray.m2,
      position: XY.zero,
      font,
      location,
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
        <div
          className="x-axis"
          style={{ ...gridStyle, backgroundColor: "var(--pluto-gray-z)" }}
          ref={resizeRef}
        />
        {children}
      </Bob.Composite>
    );
  }
);
XAxis.displayName = "XAxisC";
