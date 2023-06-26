// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, memo } from "react";

import { Optional, XY, Location, CrudeOuterLocation } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { useAxisPosition } from "@/core/vis/LinePlot/main/LinePlot";

export interface XAxisProps
  extends PropsWithChildren,
    Optional<
      Omit<z.input<typeof AetherLinePlot.XAxis.stateZ>, "position">,
      "color" | "font" | "gridColor"
    > {
  resizeDebounce?: number;
}

export const XAxis = memo(
  ({
    children,
    resizeDebounce: debounce = 100,
    location = "bottom",
    ...props
  }: XAxisProps): ReactElement => {
    const theme = Theming.use();
    const [{ key, path }, , setState] = Aether.useStateful({
      type: AetherLinePlot.XAxis.TYPE,
      schema: AetherLinePlot.XAxis.stateZ,
      initialState: {
        color: theme.colors.gray.p2,
        gridColor: theme.colors.gray.m1,
        position: XY.ZERO,
        font: Theming.font(theme, "small"),
        location,
        ...props,
      },
    });

    const gridStyle = useAxisPosition(
      new Location(location).crude as CrudeOuterLocation,
      key,
      "XAxis"
    );

    const resizeRef = useResize(
      (box) => setState((p) => ({ ...p, position: box.topLeft })),
      { debounce }
    );

    return (
      <>
        <div className="x-axis" style={gridStyle} ref={resizeRef} />
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </>
    );
  }
);
XAxis.displayName = "XAxis";
