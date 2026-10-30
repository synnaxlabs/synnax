// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/input/input.css";

import { type schematic } from "@synnaxlabs/client";
import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input as BaseInput } from "@synnaxlabs/lyra/input";
import { color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface PrimitiveProps extends Partial<
  Omit<schematic.InputNodeConfig, "variant" | "label" | "scale">
> {
  /** value is the last sent string, or null when none is known. */
  value?: string | null;
  /** readback renders under the field and shows the value the device holds. */
  readback?: ReactNode;
  className?: string;
  onSend?: (value: string) => void;
}

export const Input = ({
  className,
  value = null,
  readback,
  orientation = "left",
  fillColor,
  size,
  onSend,
  disabled,
  onClickDelay,
  inlineSize,
}: PrimitiveProps): ReactElement => {
  // The field shows a missing value as empty, so the two compare equal.
  const draft = Primitive.useDraft(value ?? "");
  const shown = draft.value ?? "";
  const symbolColor = color.rgbaString(fillColor);
  const style: CSSProperties = {
    [Primitive.SYMBOL_COLOR]: symbolColor,
    [Primitive.SYMBOL_FIELD_WIDTH]: inlineSize && `${inlineSize}px`,
  };
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(
        CSS.B("input-symbol"),
        CSS.symbolColored,
        CSS.B("symbol-field"),
        symbolColor != null && CSS.M("colored"),
        draft.dirty && CSS.M("dirty"),
        className,
      )}
      style={style}
    >
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={0}
        right={100}
        bottom={100}
      />
      <Flex.Box y align="stretch" gap={0}>
        <BaseInput.Text
          value={shown}
          onChange={draft.edit}
          className={CSS.BE("symbol-field", "control")}
          size={size}
          borderWidth={1}
          disabled={disabled}
        >
          <BaseButton.Button
            size={size}
            variant="filled"
            onClick={() => {
              onSend?.(shown);
              draft.send();
            }}
            onClickDelay={onClickDelay}
          >
            Set
          </BaseButton.Button>
        </BaseInput.Text>
        {readback}
      </Flex.Box>
    </Primitive.Div>
  );
};
