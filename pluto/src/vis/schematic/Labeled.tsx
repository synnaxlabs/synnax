// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
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
      style,
      className,
      maxInlineSize = 150,
      ...props
    },
    ref,
  ) => (
      <Align.Space
        style={{
          // You may be wondering, why do we do this here? Well it's because react flow
          // uses a ResizeObserver to determine when to re-render edges. When we switch
          // from 'left' to 'right' or 'top' to 'bottom', the width and height of the
          // node remains the same, so the ResizeObserver doesn't fire. We need to redraw
          // the edges, so we add a margin to trigger it.
          marginRight: orientation === "right" ? 1 : 0,
          marginTop: orientation === "top" ? 1 : 0,
          ...style,
        }}
        size={2 / 3}
        direction={orientation}
        ref={ref}
        className={CSS(CSS.B("symbol"), className)}
        align="center"
        justify="center"
        {...props}
      >
        {value.length > 0 && (
          <Text.Editable
            className={CSS.BE("symbol", "label")}
            value={value}
            style={{ maxInlineSize }}
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
    ),
);
Labeled.displayName = "Labeled";
