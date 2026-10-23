// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@/context";

/** True inside a closed select dialog, whose children render into a detached node. */
export const [ClosedContext, useClosed] = context.create<boolean>({
  defaultValue: false,
  displayName: "Select.ClosedContext",
});

/** How a {@link Buttons} group draws its items. */
export type ButtonsVariant = "outlined" | "text";

export interface ButtonsContextValue {
  preview: boolean;
  variant: ButtonsVariant;
}

/** Set inside a Buttons group, whose items draw as toggle buttons. */
export const [ButtonsContext, useButtonsContext] =
  context.create<ButtonsContextValue | null>({
    defaultValue: null,
    displayName: "Select.ButtonsContext",
  });
