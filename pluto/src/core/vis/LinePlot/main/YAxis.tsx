// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, memo, useMemo } from "react";

import { Optional, XY, Location, CrudeOuterLocation } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { useAxisPosition } from "@/core/vis/LinePlot/main/LinePlot";

export interface YAxisProps
  extends PropsWithChildren,
    Optional<
      Omit<z.input<typeof AetherLinePlot.YAxis.stateZ>, "position">,
      "color" | "font" | "gridColor"
    > {}

export const YAxis = memo(
  ({ children, location = "left", ...props }: YAxisProps): ReactElement => {
    const theme = Theming.use();

    const memoProps = useMemo(
      () => ({
        position: XY.ZERO,
        color: theme.colors.gray.p2,
        gridColor: theme.colors.gray.m1,
        location,
        font: Theming.font(theme, "small"),
        ...props,
      }),
      [theme, props]
    );

    const [{ key, path }, , setState] = Aether.useStateful({
      type: AetherLinePlot.YAxis.TYPE,
      schema: AetherLinePlot.YAxis.stateZ,
      initialState: memoProps,
    });

    const gridStyle = useAxisPosition(
      new Location(location).crude as CrudeOuterLocation,
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
      <>
        <div className="y-axis" style={gridStyle} ref={resizeRef}></div>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </>
    );
  }
);
YAxis.displayName = "YAxisC";
