// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction, type location } from "@synnaxlabs/x";
import { forwardRef, useCallback } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Text } from "@/text";

export interface LabelExtensionProps {
  label?: string;
  level?: Text.Level;
  orientation?: location.Outer;
  maxInlineSize?: number;
  align?: Align.Alignment;
  direction?: direction.Direction;
}

export interface LabeledProps
  extends Omit<Align.SpaceProps, "value" | "onChange" | "direction">,
    LabelExtensionProps {
  onChange: ({ label }: { label: LabelExtensionProps }) => void;
}

export const Labeled = forwardRef<HTMLDivElement, LabeledProps>(
  (
    {
      label: value = "",
      onChange,
      level = "p",
      children,
      orientation = "top",
      direction = "y",
      style,
      className,
      maxInlineSize = 150,
      ...props
    },
    ref,
  ) => {
    return (
      <Align.Space
        style={style}
        ref={ref}
        className={CSS(CSS.B("symbol"), className)}
        align="center"
        justify="center"
        {...props}
      >
        {value.length > 0 && (
          <Text.Editable
            className={CSS(
              CSS.BE("symbol", "label"),
              CSS.loc(orientation),
              CSS.dir(direction),
            )}
            value={value}
            onChange={useCallback(
              (label) =>
                onChange({
                  label: { label, level, orientation },
                }),
              [onChange, level],
            )}
            level={level}
          />
        )}
        {children}
      </Align.Space>
    );
  },
);
Labeled.displayName = "Labeled";
