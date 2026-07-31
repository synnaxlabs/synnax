// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/stringDisplay/stringDisplay.css";

import { type dimensions } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type PropsWithChildren,
  type ReactElement,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/stringDisplay/config";
import { symbolColorVar } from "@/schematic/symbolColor";

interface RenderProps extends PropsWithChildren<Omit<Config, "label" | "variant">> {
  className?: string;
  dimensions?: dimensions.Dimensions;
}

export const StringDisplay = ({
  className,
  color: colorVal,
  dimensions,
  orientation = "left",
  children,
  inlineSize,
}: RenderProps): ReactElement => {
  const style = useMemo<CSSProperties>(
    () => ({
      [CSS.var("symbol-color")]: symbolColorVar(colorVal),
      height: dimensions?.height,
    }),
    [colorVal, dimensions?.height],
  );
  return (
    <Primitive.Div
      className={CSS(CSS.B("string-display"), CSS.B("symbol-colored"), className)}
      style={style}
    >
      <div
        className={CSS.BE("string-display", "content")}
        style={{
          flexGrow: 1,
          minWidth: dimensions?.width,
          inlineSize,
          maxWidth: dimensions?.width,
        }}
      >
        {children}
      </div>
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={-2}
        right={100}
        bottom={102}
      />
    </Primitive.Div>
  );
};
