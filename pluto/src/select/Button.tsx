// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button as Base } from "@/button";
import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { useItemState } from "@/select/Context";
import { Frame, type FrameProps } from "@/select/Frame";
import { Text } from "@/text";

const [PreviewContext, usePreview] = context.create<boolean>({
  defaultValue: false,
  displayName: "Select.Buttons.Preview",
});

export interface ButtonsProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>
  extends
    Omit<Flex.BoxProps, "onSelect" | "onChange">,
    Omit<FrameProps<K, E>, "getItem" | "subscribe" | "data"> {
  /** The selectable keys, in render order. */
  keys: K[] | readonly K[];
  /** Whether to render the buttons flat and inert, for use inside a preview. */
  preview?: boolean;
}

/**
 * A packed row of {@link Button}s acting as one selection. Use it in place of a dropdown
 * when the options are few and fixed.
 *
 * @example
 * <Select.Buttons keys={MODES} value={mode} onChange={setMode}>
 *   <Select.Button itemKey="fast">Fast</Select.Button>
 * </Select.Buttons>
 */
export const Buttons = <K extends record.Key = record.Key>({
  keys,
  value,
  onChange,
  allowNone,
  multiple,
  preview = false,
  children,
  ...rest
}: ButtonsProps<K>): ReactElement => {
  const listProps = List.useKeysData<K>(keys);
  // Type assertion here because there are weird unions from these being widened to
  // their full types and then TS not being able to prove that they are compatible.
  const selectionProps = {
    allowNone,
    multiple,
    value,
    onChange,
  } as FrameProps<K, record.Keyed<K>>;
  const isEmpty = value == null || (Array.isArray(value) && value.length === 0);
  return (
    <Frame<K, record.Keyed<K>>
      closeDialogOnSelect={false}
      {...listProps}
      {...selectionProps}
    >
      <PreviewContext value={preview}>
        <Flex.Box pack {...rest}>
          {preview && isEmpty ? <Text.Text color={8}>None</Text.Text> : children}
        </Flex.Box>
      </PreviewContext>
    </Frame>
  );
};

/** Props for {@link Button}. */
export interface ButtonProps<K extends record.Key = record.Key> extends Omit<
  Base.ToggleProps,
  "onChange" | "value"
> {
  /** The key this button selects. */
  itemKey: K;
}

/** One option inside {@link Buttons}. It toggles on when its `itemKey` is selected. */
export const Button = <K extends record.Key = record.Key>({
  itemKey,
  className,
  ...rest
}: ButtonProps<K>): ReactElement | null => {
  const { selected, onSelect } = useItemState<K>(itemKey);
  const preview = usePreview();
  if (preview && !selected) return null;
  return (
    <Base.Toggle
      preview={preview}
      {...rest}
      id={itemKey.toString()}
      onChange={onSelect}
      value={selected}
      className={CSS.cls(
        className,
        CSS.B("select-btn"),
        CSS.selected(selected),
        selected && CONTEXT_SELECTED,
        CONTEXT_TARGET,
      )}
    />
  );
};
