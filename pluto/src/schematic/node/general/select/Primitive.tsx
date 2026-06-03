// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/select/select.css";

import { type CSSProperties, type ReactElement, useMemo } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/select/config";
import { Select as BaseSelect } from "@/select";

interface RenderProps extends Omit<Config, "sink" | "variant"> {
  className?: string;
  value?: string;
  onChange: (key: string | null) => void;
  onSend?: (value: number) => void;
}

const RIGHT_HANDLE_STYLE: CSSProperties = { zIndex: 5 };

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
}: RenderProps): ReactElement => {
  const data = useMemo(
    () => options.map((o) => ({ key: o.key, name: o.name || `Option ${o.value}` })),
    [options],
  );
  const matched = options.find((o) => o.key === value);
  const resolved = useColor(color);
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS(CSS.B("select-symbol"), className)}
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
          style={RIGHT_HANDLE_STYLE}
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
          triggerProps={{ color: resolved, size }}
          style={{ minWidth: inlineSize }}
        />
        {onSend != null && (
          <BaseButton.Button
            variant="filled"
            size={size}
            onClick={() => {
              if (matched != null) onSend?.(matched.value);
            }}
            color={resolved}
            disabled={disabled}
          >
            Send
          </BaseButton.Button>
        )}
      </Flex.Box>
    </Primitive.Div>
  );
};
