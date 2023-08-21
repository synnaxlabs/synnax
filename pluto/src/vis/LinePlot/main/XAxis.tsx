// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useEffect, useRef } from "react";

import { Location, CrudeOuterLocation, CrudeDirection, Direction } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/aether/main";
import { Align } from "@/align";
import { CSS } from "@/css";
import { Text } from "@/text";
import { Theming } from "@/theming/main";
import { AetherLinePlot } from "@/vis/lineplot/aether";
import { withinSizeThreshold } from "@/vis/lineplot/aether/grid";
import { useGridPosition } from "@/vis/lineplot/main/LinePlot";

export interface XAxisProps
  extends PropsWithChildren,
    Omit<z.input<typeof AetherLinePlot.XAxis.z>, "position"> {
  resizeDebounce?: number;
  label?: string;
  labelLevel?: Text.Level;
  labelDirection?: CrudeDirection | Direction;
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
    ...props
  }): ReactElement => {
    const showLabel = (label?.length ?? 0) > 0;

    const [{ path }, { size, labelSize }, setState] = Aether.use({
      aetherKey,
      type: AetherLinePlot.XAxis.TYPE,
      schema: AetherLinePlot.XAxis.z,
      initialState: {
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
        <Align.Space className="x-axis" style={gridStyle} justify="end" align="center">
          {showLabel && (
            <Text.MaybeEditable
              className={CSS(
                CSS.BE("x-axis", "label"),
                CSS.loc(location),
                CSS.dir(labelDirection)
              )}
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
XAxis.displayName = "XAxis";
