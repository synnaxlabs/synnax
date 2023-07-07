// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useEffect, useMemo, useRef } from "react";

import {
  Optional,
  Location,
  CrudeOuterLocation,
  CrudeDirection,
  Direction,
  Deep,
} from "@synnaxlabs/x";
import { z } from "zod";

import { withinSizeThreshold } from "../aether/axis";

import { Aether } from "@/core/aether/main";
import { CSS } from "@/core/css";
import { useMemoCompare, useResize } from "@/core/hooks";
import { Input, Space, Text, TypographyLevel } from "@/core/std";
import { Theming } from "@/core/theming";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { useAxisPosition } from "@/core/vis/LinePlot/main/LinePlot";

import "@/core/vis/LinePlot/main/YAxis.css";

export interface YAxisProps
  extends PropsWithChildren,
    Optional<
      Omit<z.input<typeof AetherLinePlot.YAxis.z>, "position">,
      "color" | "font" | "gridColor"
    > {
  label?: string;
  labelLevel?: TypographyLevel;
  onLabelChange?: (label: string) => void;
  labelDirection?: CrudeDirection;
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
    labelDirection = Direction.x,
    ...props
  }): ReactElement => {
    const theme = Theming.use();

    const showLabel = (label?.length ?? 0) > 0;

    const memoProps = useMemoCompare(
      () => ({
        color: theme.colors.gray.p2,
        gridColor: theme.colors.gray.m2,
        location,
        font: Theming.fontString(theme, "small"),
        ...props,
      }),
      ([theme, props], [prevTheme, prevProps]) => {
        return Deep.equal(props, prevProps);
      },
      [theme, props]
    );

    const prevLabelSize = useRef(0);

    const [{ path }, { size, labelSize }, setState] = Aether.use({
      aetherKey,
      type: AetherLinePlot.YAxis.TYPE,
      schema: AetherLinePlot.YAxis.z,
      initialState: memoProps,
    });

    useEffect(() => {
      setState((state) => ({
        ...state,
        ...memoProps,
      }));
    }, [memoProps]);

    const gridStyle = useAxisPosition(
      {
        loc: new Location(location).crude as CrudeOuterLocation,
        key: aetherKey,
        size: size + labelSize,
      },
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

    const font = Theming.useTypography(labelLevel);

    useEffect(() => {
      if (label == null) return;
      const dims = Text.dimensions(label, font.toString());
      const labelSize = dims[new Direction(labelDirection).dimension];
      const prevSize = prevLabelSize.current;
      if (!withinSizeThreshold(prevSize, labelSize)) {
        setState((state) => ({
          ...state,
          labelSize,
        }));
      }
    }, [label]);

    return (
      <>
        <Space
          className="y-axis"
          style={gridStyle}
          ref={resizeRef}
          align="start"
          justify="center"
        >
          {showLabel && (
            <Text.MaybeEditable
              className={CSS(
                CSS.BE("y-axis", "label"),
                CSS.dir(labelDirection),
                CSS.loc(location)
              )}
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
