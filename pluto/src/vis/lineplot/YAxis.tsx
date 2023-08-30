// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useEffect } from "react";

import { Location, CrudeOuterLocation, Direction, CrudeDirection } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/hooks";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { lineplot } from "@/vis/lineplot/aether";
import { useGridPosition } from "@/vis/lineplot/LinePlot";

import "@/vis/lineplot/YAxis.css";

export interface YAxisProps
  extends PropsWithChildren,
    Omit<z.input<typeof lineplot.yAxisStateZ>, "position" | "size">,
    Omit<Align.SpaceProps, "color"> {
  label?: string;
  labelLevel?: Text.Level;
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

    const aetherProps = useMemoDeepEqualProps({
      location,
      showGrid,
      type,
      bounds,
      label,
    });

    const [{ path }, { size, labelSize }, setState] = Aether.use({
      aetherKey,
      type: lineplot.YAxis.TYPE,
      schema: lineplot.yAxisStateZ,
      initialState: aetherProps,
    });

    useEffect(() => setState((state) => ({ ...state, ...aetherProps })), [aetherProps]);

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
        <Align.Space
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
        </Align.Space>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </>
    );
  }
);
