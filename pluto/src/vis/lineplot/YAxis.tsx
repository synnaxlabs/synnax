// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useEffect } from "react";

import { direction } from "@synnaxlabs/x";
import { type z } from "zod";

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
  labelDirection?: direction.Crude;
}

export const YAxis = Aether.wrap<YAxisProps>(
  "YAxis",
  ({
    aetherKey,
    children,
    location: l = "left",
    label,
    labelLevel = "small",
    onLabelChange,
    labelDirection = "x",
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
      location: l,
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
        loc: l,
        key: aetherKey,
        size: size + labelSize,
        order: "last",
      },
      "YAxis",
    );

    const font = Theming.useTypography(labelLevel);

    useEffect(() => {
      if (label == null) return;
      const dims = Text.dimensions(label, font.toString());
      let labelSize = dims[direction.dimension(direction.construct(labelDirection))];
      if (labelSize > 0) labelSize += 6;
      setState((state) => ({
        ...state,
        labelSize,
      }));
    }, [label, labelDirection]);

    return (
      <>
        <Align.Space
          className={CSS(className, CSS.loc(l), CSS.B("y-axis"))}
          style={{ ...style, ...gridStyle }}
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
  },
);
