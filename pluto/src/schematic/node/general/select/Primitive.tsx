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
import { type Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Select as BaseSelect } from "@synnaxlabs/lyra/select";
import { color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

interface RenderProps extends Partial<
  Pick<
    schematic.SelectNodeConfig,
    "fillColor" | "orientation" | "size" | "disabled" | "inlineSize" | "onClickDelay"
  >
> {
  options: schematic.SelectNodeConfig["options"];
  className?: string;
  /** value is the key of the last sent option, or null when none is known. */
  value?: string | null;
  /** readback renders under the field and shows the state the device holds. */
  readback?: ReactNode;
  onSend: (value: number) => void;
}

/** @returns the name an option shows, falling back to its value. */
export const optionName = (o: schematic.SelectNodeConfig["options"][number]): string =>
  o.name || `Option ${o.value}`;

const DIALOG_PROPS: Dialog.DialogProps = {
  className: CSS.BE("select-symbol", "dialog"),
};

export const Select = ({
  className,
  orientation = "left",
  fillColor,
  value = null,
  readback,
  onSend,
  options,
  size,
  disabled,
  inlineSize,
  onClickDelay,
}: RenderProps): ReactElement => {
  const draft = Primitive.useDraft(value);
  const matched = options.find((o) => o.key === draft.value);
  const symbolColor = color.rgbaString(fillColor);
  const style: CSSProperties = {
    [Primitive.SYMBOL_COLOR]: symbolColor,
    [Primitive.SYMBOL_FIELD_WIDTH]: inlineSize && `${inlineSize}px`,
  };
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(
        CSS.B("select-symbol"),
        CSS.symbolColored,
        CSS.B("symbol-field"),
        symbolColor != null && CSS.M("colored"),
        draft.dirty && CSS.M("dirty"),
        className,
      )}
      style={style}
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
        <Flex.Box x pack size={size}>
          <BaseSelect.Simple<string>
            value={draft.value ?? undefined}
            onChange={draft.edit}
            disabled={disabled}
            resourceName="option"
            triggerProps={{ size, className: CSS.BE("symbol-field", "control") }}
            dialogProps={DIALOG_PROPS}
          >
            {options.map((o) => (
              <BaseSelect.Item key={o.key} itemKey={o.key}>
                {optionName(o)}
              </BaseSelect.Item>
            ))}
          </BaseSelect.Simple>
          <BaseButton.Button
            variant="filled"
            size={size}
            onClick={() => {
              if (matched == null) return;
              onSend(matched.value);
              draft.send();
            }}
            onClickDelay={onClickDelay}
            disabled={disabled}
          >
            Set
          </BaseButton.Button>
        </Flex.Box>
        {readback}
      </Flex.Box>
    </Primitive.Div>
  );
};
