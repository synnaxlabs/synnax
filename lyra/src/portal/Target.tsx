// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";

const [Base, useBase] = context.create<HTMLElement | null>({
  defaultValue: null,
  displayName: "Portal.Target",
});

/**
 * Returns the element that overlays in the enclosing {@link Target} portal into, or
 * null outside a Target or while its element is not mounted.
 */
export const useTarget = (): HTMLElement | null => useBase();

export interface TargetProps extends PropsWithChildren {
  /** The element that overlays portal into. Null keeps each overlay's own default. */
  element: HTMLElement | null;
}

/**
 * Makes the dialogs, context menus, and tooltips in its children portal into element
 * instead of their default target.
 */
export const Target = ({ element, children }: TargetProps): ReactElement => (
  <Base value={element}>{children}</Base>
);
