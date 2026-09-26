// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/select/select.css";

import { type schematic } from "@synnaxlabs/client";
import { Button as BaseButton } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { type Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Select as BaseSelect } from "@synnaxlabs/lyra/select";
import { type ReactElement, useMemo } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface RenderProps extends Partial<
  Pick<
    schematic.SelectNodeConfig,
    "color" | "orientation" | "size" | "disabled" | "inlineSize" | "onClickDelay"
  >
> {
  options: schematic.SelectNodeConfig["options"];
  className?: string;
  value?: string;
  onChange: (key: string | null) => void;
  onSend?: (value: number) => void;
}

const DIALOG_PROPS: Dialog.DialogProps = {
  className: CSS.BE("select-symbol", "dialog"),
};

export const Select = ({
  className,
  orientation = "left",
  color,
  value,
  onChange,
  onSend,
  options,
  size,
  disabled,
  inlineSize,
  onClickDelay,
}: RenderProps): ReactElement => {
  const data = useMemo(
    () => options.map((o) => ({ key: o.key, name: o.name || `Option ${o.value}` })),
    [options],
  );
  const matched = options.find((o) => o.key === value);
  const triggerStyle = useMemo(() => ({ minWidth: inlineSize }), [inlineSize]);
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(CSS.B("select-symbol"), className)}
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
      <Flex.Box x pack size={size}>
        <BaseSelect.Static
          data={data}
          value={value}
          onChange={(key: string | null) => onChange(key)}
          disabled={disabled}
          resourceName="option"
          triggerProps={{ color, size }}
          dialogProps={DIALOG_PROPS}
          style={triggerStyle}
        />
        {onSend != null && (
          <BaseButton.Button
            variant="filled"
            size={size}
            onClick={() => {
              if (matched != null) onSend?.(matched.value);
            }}
            onClickDelay={onClickDelay}
            color={color}
            disabled={disabled}
          >
            Send
          </BaseButton.Button>
        )}
      </Flex.Box>
    </Primitive.Div>
  );
};
