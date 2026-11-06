// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AriaAttributes } from "react";

import { context } from "@/context";
import { type KeyboardHandlers } from "@/select/useCursor";
import { type Status } from "@/status";

/** True inside a closed select dialog, whose children render into a detached node. */
export const [ClosedContext, useClosed] = context.create<boolean>({
  defaultValue: false,
  displayName: "Select.ClosedContext",
});

/** True inside a `Select.List`. A fixed item outside one is pinned. */
export const [ListContext, useInList] = context.create<boolean>({
  defaultValue: false,
  displayName: "Select.ListContext",
});

/** How a {@link Buttons} group draws its items. */
export type ButtonsVariant = "outlined" | "text" | "card";

export interface ButtonsContextValue extends Pick<
  AriaAttributes,
  "aria-invalid" | "aria-describedby"
> {
  disabled?: boolean;
  preview: boolean;
  variant: ButtonsVariant;
  status?: Status.Variant;
}

/** Set inside a Buttons group, whose items draw as toggle buttons. */
export const [ButtonsContext, useButtonsContext] =
  context.create<ButtonsContextValue | null>({
    defaultValue: null,
    displayName: "Select.ButtonsContext",
  });

const [KeyboardContext, useKeyboardContext] = context.create<KeyboardHandlers>({
  displayName: "Select.KeyboardContext",
  providerName: "Select.Frame",
});
export { KeyboardContext };

/**
 * Returns the key handlers that move the enclosing frame's selection. Spread them on
 * the focusable element whose keys move it. Only a frame whose `cursor` is
 * `"selection"` answers them.
 */
export const useKeyboard = (): KeyboardHandlers =>
  useKeyboardContext("Select.useKeyboard");
