// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/stringDisplay/stringDisplay.css";

import { color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/stringDisplay/config";
import { symbolColorVar } from "@/schematic/symbolColor";
import { Text } from "@/text";

interface RenderProps extends Omit<Config, "label" | "variant"> {
  className?: string;
  value?: string;
  stale?: boolean;
}

export const StringDisplay = ({
  className,
  color: colorVal,
  textColor,
  stalenessColor,
  level = "p",
  orientation = "left",
  inlineSize,
  value,
  stale = false,
}: RenderProps): ReactElement => {
  const style = useMemo<CSSProperties>(
    () => ({
      [CSS.var("symbol-color")]: symbolColorVar(colorVal),
      minWidth: inlineSize,
    }),
    [colorVal, inlineSize],
  );
  const resolvedTextColor = stale ? stalenessColor : textColor;
  return (
    <Primitive.Div
      className={CSS(CSS.B("string-display"), CSS.B("symbol-colored"), className)}
      style={style}
    >
      <div className={CSS.BE("string-display", "content")}>
        <Text.Text
          level={level}
          color={
            resolvedTextColor != null ? color.cssString(resolvedTextColor) : undefined
          }
          variant="code"
        >
          {value}
        </Text.Text>
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
