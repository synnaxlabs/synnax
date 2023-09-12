// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useEffect, useRef } from "react";

import { type location, type direction } from "@synnaxlabs/x";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/memo";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { lineplot } from "@/vis/lineplot/aether";
import { withinSizeThreshold } from "@/vis/lineplot/aether/grid";
import { useGridPosition } from "@/vis/lineplot/LinePlot";

export interface XAxisProps
  extends PropsWithChildren,
    Omit<z.input<typeof lineplot.xAxisStateZ>, "position" | "size">,
    Omit<Align.SpaceProps, "color"> {
  resizeDebounce?: number;
  label?: string;
  labelLevel?: Text.Level;
  labelDirection?: direction.Direction;
  onLabelChange?: (label: string) => void;
}

export const XAxis = Aether.wrap<XAxisProps>(
  "XAxis",
  ({
    aetherKey,
    children,
    resizeDebounce: debounce = 0,
    location = "bottom",
    label,
    labelLevel = "small",
    labelDirection,
    onLabelChange,
    color,
    labelSize: propsLabelSize,
    showGrid,
    type,
    bounds,
    className,
    style,
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
      type: lineplot.XAxis.TYPE,
      schema: lineplot.xAxisStateZ,
      initialState: aetherProps,
    });

    useEffect(() => setState((state) => ({ ...state, ...aetherProps })), [aetherProps]);

    const prevLabelSize = useRef(0);

    const gridStyle = useGridPosition(
      {
        loc: location as location.Outer,
        key: aetherKey,
        size: size + labelSize,
        order: "last",
      },
      "XAxis",
    );

    const font = Theming.useTypography(labelLevel);

    useEffect(() => {
      if (label == null) return;
      const dims = Text.dimensions(label, font.toString());
      const labelSize = dims.height + 12;
      const prevSize = prevLabelSize.current;
      if (!withinSizeThreshold(prevSize, labelSize)) {
        prevLabelSize.current = labelSize;
        setState((state) => ({
          ...state,
          labelSize,
        }));
      }
    }, [label]);

    return (
      <>
        <Align.Space
          className={CSS(className, CSS.B("x-axis"), CSS.loc(location))}
          style={{ ...style, ...gridStyle }}
          justify="end"
          align="center"
          {...props}
        >
          {showLabel && (
            <Text.MaybeEditable
              className={CSS(CSS.BE("x-axis", "label"), CSS.dir(labelDirection))}
              value={label as string}
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
