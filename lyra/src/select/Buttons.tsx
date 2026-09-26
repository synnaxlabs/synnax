// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Buttons.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Frame, type FrameProps } from "@/select/Frame";
import { ButtonsContext, type ButtonsVariant } from "@/select/scope";
import { Text } from "@/text";

export interface ButtonsProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>
  extends
    Omit<Flex.BoxProps, "onSelect" | "onChange">,
    Omit<
      FrameProps<K, E>,
      | "getItem"
      | "subscribe"
      | "data"
      | "virtual"
      | "itemHeight"
      | "overscan"
      | "onFetchMore"
    > {
  /** Whether to render the buttons flat and inert, for use inside a preview. */
  preview?: boolean;
  /**
   * "text" (the default) spaces the buttons like tabs: the unselected ones are plain
   * text and only the selected one carries a chassis. "outlined" packs bordered
   * buttons into one control, for controls that float over a canvas.
   */
  variant?: ButtonsVariant;
}

/**
 * A row of toggle buttons acting as one selection, one per {@link Item} child. Use it
 * in place of a dropdown when the options are few and fixed.
 *
 * @example
 * <Select.Buttons value={mode} onChange={setMode}>
 *   <Select.Item itemKey="fast">Fast</Select.Item>
 * </Select.Buttons>
 */
export const Buttons = <K extends record.Key = record.Key>({
  value,
  onChange,
  allowNone,
  multiple,
  preview = false,
  variant = "text",
  className,
  children,
  ...rest
}: ButtonsProps<K>): ReactElement => {
  // Type assertion here because there are weird unions from these being widened to
  // their full types and then TS not being able to prove that they are compatible.
  const selectionProps = {
    allowNone,
    multiple,
    value,
    onChange,
  } as FrameProps<K, record.Keyed<K>>;
  const isEmpty = value == null || (Array.isArray(value) && value.length === 0);
  const ctx = useMemo(() => ({ preview, variant }), [preview, variant]);
  const text = variant === "text";
  return (
    <Frame<K, record.Keyed<K>> closeDialogOnSelect={false} {...selectionProps}>
      <ButtonsContext value={ctx}>
        <Flex.Box
          x
          pack={!text}
          gap={text ? "tiny" : undefined}
          className={CSS.cls(
            className,
            CSS.B("select-btns"),
            CSS.BM("select-btns", variant),
          )}
          {...rest}
        >
          {preview && isEmpty ? <Text.Text color={8}>None</Text.Text> : children}
        </Flex.Box>
      </ButtonsContext>
    </Frame>
  );
};
