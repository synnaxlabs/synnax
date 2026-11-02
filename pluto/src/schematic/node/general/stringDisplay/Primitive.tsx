// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/stringDisplay/stringDisplay.css";

import { type schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { color } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type PropsWithChildren,
  type ReactElement,
  type Ref,
} from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { LEVEL_SIZES } from "@/schematic/node/common/size";

interface RenderProps extends PropsWithChildren<
  Partial<
    Pick<
      schematic.StringDisplayNodeConfig,
      "strokeColor" | "orientation" | "level" | "inlineSize"
    >
  >
> {
  className?: string;
  /** contentRef receives the element the canvas draws the text into. */
  contentRef?: Ref<HTMLDivElement>;
}

export const StringDisplay = ({
  className,
  strokeColor,
  level = "p",
  orientation = "left",
  inlineSize,
  contentRef,
  children,
}: RenderProps): ReactElement => {
  const symbolColor = color.rgbaString(strokeColor);
  const style: CSSProperties = {
    [Primitive.SYMBOL_COLOR]: symbolColor,
    width: inlineSize,
    height: Component.HEIGHTS[LEVEL_SIZES[level]],
  };
  return (
    <Primitive.Div
      className={CSS.cls(
        CSS.B("string-display"),
        CSS.symbolColored,
        CSS.B("symbol-readout"),
        symbolColor != null && CSS.M("colored"),
        className,
      )}
      style={style}
    >
      <div ref={contentRef} className={CSS.BE("string-display", "content")}>
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
