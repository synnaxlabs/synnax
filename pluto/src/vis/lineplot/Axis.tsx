// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/lineplot/Axis.css";

import { type bounds, direction } from "@synnaxlabs/x";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { CSS } from "@/css";
import { useMemoDeepEqualProps as useMemoDeepEqual } from "@/memo";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { lineplot } from "@/vis/lineplot/aether";
import {
  coreAxisStateZ,
  parseAutoBounds,
  withinSizeThreshold,
} from "@/vis/lineplot/aether/axis";
import { useGridEntry } from "@/vis/lineplot/LinePlot";

export interface AxisProps
  extends PropsWithChildren,
    Omit<z.input<typeof lineplot.xAxisStateZ>, "position" | "size">,
    Omit<Align.SpaceProps, "color"> {
  label?: string;
  labelLevel?: Text.Level;
  labelDirection?: direction.Direction;
  onLabelChange?: (label: string) => void;
  onAutoBoundsChange?: (bounds: bounds.Bounds) => void;
}

export const axisFactory = (dir: direction.Direction): FC<AxisProps> => {
  const defaultLocation = dir === "x" ? "bottom" : "left";
  const aetherType = dir === "x" ? lineplot.XAxis.TYPE : lineplot.YAxis.TYPE;
  const cssClass = `${dir}-axis`;
  return Aether.wrap<AxisProps>(
    aetherType,
    ({
      aetherKey,
      children,
      location = defaultLocation,
      label = "",
      labelLevel = "small",
      labelDirection = dir,
      onLabelChange,
      color,
      labelSize: propsLabelSize,
      showGrid,
      type,
      bounds,
      className,
      tickSpacing,
      autoBounds,
      autoBoundUpdateInterval,
      onAutoBoundsChange,
      style,
      ...props
    }): ReactElement => {
      const showLabel = (label?.length ?? 0) > 0;

      const aetherProps = useMemoDeepEqual({
        location,
        showGrid,
        type,
        bounds,
        label,
        labelDirection,
        tickSpacing,
        autoBounds,
        autoBoundUpdateInterval,
      });

      const [{ path }, { size, labelSize, ...state }, setState] = Aether.use({
        aetherKey,
        type: aetherType,
        schema: coreAxisStateZ,
        initialState: aetherProps,
      });

      useEffect(
        () => setState((state) => ({ ...state, ...aetherProps })),
        [aetherProps],
      );
      useEffect(() => {
        const { lower, upper } = parseAutoBounds(state.autoBounds);
        if (state.bounds == null) return;
        if (
          (lower && bounds?.lower !== state.bounds.lower) ||
          (upper && bounds?.upper !== state.bounds.upper)
        )
          onAutoBoundsChange?.(state.bounds);
      }, [state.autoBounds, state.bounds]);

      const gridStyle = useGridEntry(
        {
          loc: location,
          key: `${aetherType}-${aetherKey}`,
          size: size + labelSize,
          order: "last",
        },
        "XAxis",
      );

      const font = Theming.useTypography(labelLevel).toString();

      const prevLabelSize = useRef(0);

      useEffect(() => {
        if (dir === "y") {
          if (label == null) return;
          const dims = Text.dimensions(label, font);
          let labelSize =
            dims[direction.dimension(direction.construct(labelDirection))];
          if (labelSize > 0) labelSize += 12;
          setState((state) => ({ ...state, labelSize }));
        } else {
          const dims = Text.dimensions(label, font);
          let labelSize = dims.height * 1.3;
          if (labelSize > 0) labelSize += 12;
          const prevSize = prevLabelSize.current;
          if (!withinSizeThreshold(prevSize, labelSize)) {
            prevLabelSize.current = labelSize;
            setState((state) => ({
              ...state,
              labelSize,
            }));
          }
        }
      }, [label, labelDirection, font]);

      return (
        <>
          <Align.Space
            className={CSS(
              className,
              CSS.B("axis"),
              CSS.B(cssClass),
              CSS.loc(location),
            )}
            style={{ ...style, ...gridStyle }}
            align="center"
            justify={location !== "left" ? "end" : "start"}
            direction={direction.swap(dir)}
            {...props}
          >
            {showLabel && (
              <Text.MaybeEditable
                className={CSS(CSS.BE("axis", "label"), CSS.dir(labelDirection))}
                value={label}
                onChange={onLabelChange}
                level={labelLevel}
              />
            )}
          </Align.Space>
          <Aether.Composite path={path}>{children}</Aether.Composite>
        </>
      );
    },
  );
};

export const XAxis = axisFactory("x");
export const YAxis = axisFactory("y");
