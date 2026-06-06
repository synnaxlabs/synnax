// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Axis.css";

import { direction, type text } from "@synnaxlabs/x";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { lineplot } from "@/lineplot/aether";
import { baseAxisStateZ, withinSizeThreshold } from "@/lineplot/aether/axis";
import { useGridEntry } from "@/lineplot/Frame";
import { useMemoDeepEqual } from "@/memo";
import { Text } from "@/text";
import { text as aetherText } from "@/text/aether";
import { Theming } from "@/theming";

export interface AxisProps
  extends
    PropsWithChildren,
    Omit<z.input<typeof lineplot.xAxisStateZ>, "position" | "size">,
    Omit<Flex.BoxProps, "color">,
    Aether.ComponentProps {
  label?: string;
  labelLevel?: text.Level;
  labelDirection?: direction.Direction;
  onLabelChange?: (label: string) => void;
}

export const axisFactory = (dir: direction.Direction): FC<AxisProps> => {
  const defaultLocation = dir === "x" ? "bottom" : "left";
  const aetherType = dir === "x" ? lineplot.XAxis.TYPE : lineplot.YAxis.TYPE;
  const cssClass = `${dir}-axis`;
  const C = ({
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
    axisKey,
    autoBoundUpdateInterval,
    style,
    ...rest
  }: AxisProps): ReactElement => {
    const showLabel = (label?.length ?? 0) > 0;
    const cKey = useUniqueKey(aetherKey);

    const aetherProps = useMemoDeepEqual({
      location,
      showGrid,
      type,
      bounds,
      axisKey,
      label,
      labelDirection,
      tickSpacing,
      autoBounds,
      autoBoundUpdateInterval,
    });

    const [{ path }, { size, labelSize }, setState] = Aether.use({
      aetherKey: cKey,
      type: aetherType,
      schema: baseAxisStateZ,
      initialState: aetherProps,
    });

    useEffect(() => setState((state) => ({ ...state, ...aetherProps })), [aetherProps]);

    const gridStyle = useGridEntry(
      { loc: location, key: `${aetherType}-${cKey}`, size: size + labelSize, order: 1 },
      `LinePlot.${dir.toUpperCase()}Axis`,
    );

    const font = Theming.useTypography(labelLevel).toString();

    const prevLabelSize = useRef(0);

    useEffect(() => {
      if (dir === "y") {
        if (label == null) return;
        const dims = aetherText.dimensions(label, font);
        let labelSize = dims[direction.dimension(direction.construct(labelDirection))];
        if (labelSize > 0) labelSize += 9;
        setState((state) => ({ ...state, labelSize }));
      } else {
        const dims = aetherText.dimensions(label, font);
        let labelSize = dims.height * 1.3;
        if (labelSize > 0) labelSize += 12;
        const prevSize = prevLabelSize.current;
        if (!withinSizeThreshold(prevSize, labelSize)) {
          prevLabelSize.current = labelSize;
          setState((state) => ({ ...state, labelSize }));
        }
      }
    }, [label, labelDirection, font]);

    return (
      <>
        <Flex.Box
          className={CSS(className, CSS.B("axis"), CSS.B(cssClass), CSS.loc(location))}
          style={{ ...style, ...gridStyle }}
          align="center"
          justify={location !== "left" ? "end" : "start"}
          direction={direction.swap(dir)}
          {...rest}
        >
          {showLabel && (
            <Text.MaybeEditable
              className={CSS(CSS.BE("axis", "label"), CSS.dir(labelDirection))}
              value={label}
              onChange={onLabelChange}
              level={labelLevel}
            />
          )}
        </Flex.Box>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </>
    );
  };
  C.displayName = `${dir.toUpperCase()}Axis`;
  return C;
};

export const XAxis = axisFactory("x");
export const YAxis = axisFactory("y");
