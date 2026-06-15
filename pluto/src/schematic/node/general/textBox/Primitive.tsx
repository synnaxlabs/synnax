// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/textBox/textBox.css";

import { type schematic } from "@synnaxlabs/client";
import { direction } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { symbolColorVar } from "@/schematic/symbolColor";
import { Text } from "@/text";

interface RenderProps extends Omit<schematic.NodeConfigTextBox, "variant"> {
  className?: string;
  onChange?: (value: string) => void;
}

export const TextBox = ({
  className,
  orientation = "left",
  width,
  color: colorVal,
  level,
  autoFit,
  align = "center",
  value,
  onChange,
}: RenderProps): ReactElement => {
  const isVertical = direction.construct(orientation) === "y";
  const size = autoFit ? "fit-content" : width;
  const style = useMemo<CSSProperties>(
    () => ({
      textAlign: align as CSSProperties["textAlign"],
      [CSS.var("symbol-color")]: symbolColorVar(colorVal),
      ...(isVertical ? { height: size } : { width: size }),
    }),
    [align, colorVal, isVertical, size],
  );

  return (
    <Primitive.Div
      style={style}
      orientation={orientation}
      className={CSS(
        CSS.B("text-box"),
        CSS.B("symbol-colored"),
        CSS.loc(orientation),
        className,
      )}
    >
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={0}
        right={100}
        bottom={100}
      />
      <Text.MaybeEditable
        className={CSS.BE("symbol", "label")}
        level={level}
        value={value ?? ""}
        onChange={onChange}
      />
    </Primitive.Div>
  );
};
