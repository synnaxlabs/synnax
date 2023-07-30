// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useEffect } from "react";

import {
  Location,
  CrudeOuterLocation,
  Direction,
  Deep,
  CrudeDirection,
} from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { CSS } from "@/core/css";
import { useMemoCompare } from "@/core/hooks";
import { Space, SpaceProps, Text, TypographyLevel } from "@/core/std";
import { Theming } from "@/core/theming";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { useGridPosition } from "@/core/vis/LinePlot/main/LinePlot";

import "@/core/vis/LinePlot/main/YAxis.css";

export interface YAxisProps
  extends PropsWithChildren,
    Omit<z.input<typeof AetherLinePlot.YAxis.stateZ>, "position" | "size">,
    Omit<SpaceProps, "color"> {
  label?: string;
  labelLevel?: TypographyLevel;
  onLabelChange?: (label: string) => void;
  labelDirection?: Direction | CrudeDirection;
}

export const YAxis = Aether.wrap<YAxisProps>(
  "YAxis",
  ({
    aetherKey,
    children,
    location = "left",
    label,
    labelLevel = "small",
    onLabelChange,
    labelDirection = Direction.X,
    color,
    labelSize: propsLabelSize,
    showGrid,
    type,
    bounds,
    ...props
  }): ReactElement => {
    const showLabel = (label?.length ?? 0) > 0;

    const memoProps = useMemoCompare(
      () => ({
        location,
        showGrid,
        type,
        bounds,
      }),
      (
        [, propsLabelSize, type, showGrid, color, bounds],
        [, prevPropsLabelSize, prevType, prevShowGrid, prevColor, prevBounds]
      ) => {
        return Deep.equal(
          {
            color,
            propsLabelSize,
            type,
            showGrid,
            bounds,
          },
          {
            color: prevColor,
            propsLabelSize: prevPropsLabelSize,
            type: prevType,
            showGrid: prevShowGrid,
            bounds: prevBounds,
          }
        );
      },
      [propsLabelSize, type, showGrid, bounds]
    );

    const [{ path }, { size, labelSize }, setState] = Aether.use({
      aetherKey,
      type: AetherLinePlot.YAxis.TYPE,
      schema: AetherLinePlot.YAxis.stateZ,
      initialState: memoProps,
    });

    useEffect(() => {
      setState((state) => ({ ...state, ...memoProps }));
    }, [memoProps]);

    const gridStyle = useGridPosition(
      {
        loc: new Location(location).crude as CrudeOuterLocation,
        key: aetherKey,
        size: size + labelSize,
        order: "last",
      },
      "YAxis"
    );

    const font = Theming.useTypography(labelLevel);

    useEffect(() => {
      if (label == null) return;
      const dims = Text.dimensions(label, font.toString());
      let labelSize = dims[new Direction(labelDirection).dimension];
      if (labelSize > 0) labelSize += 6;
      setState((state) => ({
        ...state,
        labelSize,
      }));
    }, [label, labelDirection]);

    return (
      <>
        <Space
          className={CSS(CSS.loc(location), CSS.B("y-axis"))}
          style={gridStyle}
          justify="center"
          {...props}
        >
          {showLabel && (
            <Text.MaybeEditable
              className={CSS(CSS.BE("y-axis", "label"), CSS.dir(labelDirection))}
              value={label as string}
              onChange={onLabelChange}
              level={labelLevel}
            />
          )}
        </Space>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </>
    );
  }
);
