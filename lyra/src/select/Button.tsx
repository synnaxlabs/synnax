// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Button.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { Button as Base } from "@/button";
import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/types";
import { useItemState } from "@/select/Context";
import { Frame, type FrameProps } from "@/select/Frame";
import { Text } from "@/text";

/** How the group draws its buttons. */
export type Variant = "outlined" | "text";

interface ContextValue {
  preview: boolean;
  variant: Variant;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { preview: false, variant: "text" },
  displayName: "Select.Buttons",
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
  /**
   * "text" (the default) spaces the buttons like tabs: the unselected ones are plain
   * text and only the selected one carries a chassis. "outlined" packs bordered
   * buttons into one control, for controls that float over a canvas.
   */
  variant?: Variant;
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
  variant = "text",
  className,
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
  const ctx = useMemo(() => ({ preview, variant }), [preview, variant]);
  const text = variant === "text";
  return (
    <Frame<K, record.Keyed<K>>
      closeDialogOnSelect={false}
      {...listProps}
      {...selectionProps}
    >
      <Context value={ctx}>
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
      </Context>
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
  const { preview, variant } = useContext();
  if (preview && !selected) return null;
  return (
    <Base.Toggle
      preview={preview}
      variant={variant}
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
