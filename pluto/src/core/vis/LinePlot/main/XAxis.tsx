// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useEffect, useRef } from "react";

import { Optional, Location, CrudeOuterLocation } from "@synnaxlabs/x";
import { z } from "zod";

import { withinSizeThreshold } from "../aether/axis";

import { Aether } from "@/core/aether/main";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { Space, Text, TypographyLevel } from "@/core/std";
import { Theming } from "@/core/theming";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { useGridPosition } from "@/core/vis/LinePlot/main/LinePlot";

export interface XAxisProps
  extends PropsWithChildren,
    Optional<
      Omit<z.input<typeof AetherLinePlot.XAxis.z>, "position">,
      "color" | "font" | "gridColor"
    > {
  resizeDebounce?: number;
  label?: string;
  labelLevel?: TypographyLevel;
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
    onLabelChange,
    ...props
  }): ReactElement => {
    const theme = Theming.use();
    const showLabel = (label?.length ?? 0) > 0;

    const [{ path }, { size, labelSize }, setState] = Aether.use({
      aetherKey,
      type: AetherLinePlot.XAxis.TYPE,
      schema: AetherLinePlot.XAxis.z,
      initialState: {
        color: theme.colors.gray.p2,
        gridColor: theme.colors.gray.m2,
        font: Theming.fontString(theme, "small"),
        location,
        ...props,
      },
    });

    const prevLabelSize = useRef(0);

    const gridStyle = useGridPosition(
      {
        loc: new Location(location).crude as CrudeOuterLocation,
        key: aetherKey,
        size: size + labelSize,
        order: "last",
      },
      "XAxis"
    );

    const resizeRef = useResize(
      (box) => setState((p) => ({ ...p, position: box.topLeft })),
      { debounce }
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
        <Space
          className="x-axis"
          style={gridStyle}
          ref={resizeRef}
          justify="end"
          align="center"
        >
          {showLabel && (
            <Text.MaybeEditable
              className={CSS(CSS.BE("x-axis", "label"), CSS.loc(location))}
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
XAxis.displayName = "XAxis";
