// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/setpoint/setpoint.css";

import { type schematic } from "@synnaxlabs/client";
import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input as BaseInput } from "@synnaxlabs/lyra/input";
import { color } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type ReactElement,
  type ReactNode,
  useMemo,
  useRef,
  useState,
} from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface RenderProps extends Partial<
  Omit<schematic.SetpointNodeConfig, "variant" | "label" | "scale">
> {
  className?: string;
  style?: CSSProperties;
  /** value is the last commanded value, or null when none is known. */
  value?: number | null;
  /** readback renders under the field and shows the value the device holds. */
  readback?: ReactNode;
  onSend: (value: number) => void;
}

export const Setpoint = ({
  orientation = "left",
  className,
  style,
  units,
  fillColor,
  onSend,
  size = "small",
  disabled,
  onClickDelay,
  value = null,
  readback,
  inlineSize,
}: RenderProps): ReactElement => {
  // A draft is a typed value not yet sent. Without one, the field follows the target,
  // and stays blank while no target is known.
  const [draft, setDraft] = useState<number | null>(null);
  const shown = draft ?? value;
  const dirty = draft != null && draft !== value;
  const inputRef = useRef<HTMLInputElement>(null);
  const symbolColor = color.rgbaString(fillColor);
  const mergedStyle = useMemo(
    () => ({
      ...style,
      [CSS.variable("symbol-color")]: symbolColor,
      [CSS.variable("symbol-field-width")]: inlineSize && `${inlineSize}px`,
    }),
    [style, symbolColor, inlineSize],
  );
  return (
    <Primitive.Div
      className={CSS.cls(
        CSS.B("setpoint"),
        CSS.B("symbol-colored"),
        CSS.B("symbol-field"),
        symbolColor != null && CSS.M("colored"),
        dirty && CSS.M("dirty"),
        className,
      )}
      orientation={orientation}
      style={mergedStyle}
    >
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={0.5}
          top={50}
          id="1"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={100}
          top={50}
          id="2"
        />
        <Handle.Handle
          location="top"
          orientation={orientation}
          left={50}
          top={-2}
          id="3"
        />
        <Handle.Handle
          location="bottom"
          orientation={orientation}
          left={50}
          top={102}
          id="4"
        />
      </Handle.Boundary>
      <Flex.Box y align="stretch" gap={0}>
        <BaseInput.Numeric
          ref={inputRef}
          size={size}
          className={CSS.BE("symbol-field", "control")}
          value={shown ?? NaN}
          emptyValue={NaN}
          onChange={(v) => setDraft(Number.isNaN(v) ? null : v)}
          showDragHandle={false}
          selectOnFocus
          endContent={units === "" ? undefined : units}
          borderWidth={1}
          disabled={disabled}
        >
          <BaseButton.Button
            size={size}
            variant="filled"
            onClick={() => {
              if (shown == null) return;
              onSend(shown);
              setDraft(null);
            }}
            onClickDelay={onClickDelay}
            // WebKit leaves the input focused on a button press, so the typed value
            // would never commit. Blurring commits it before the click or hold sends.
            onMouseDown={() => inputRef.current?.blur()}
          >
            Set
          </BaseButton.Button>
        </BaseInput.Numeric>
        {readback}
      </Flex.Box>
    </Primitive.Div>
  );
};
