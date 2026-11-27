// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";

import type { ContextValue } from "@/form/Context";

/**
 * Asks the part of the form that shows the first error status to take focus. Does
 * nothing when the form has no error.
 * @throws {Error} with the error's message when no part takes focus.
 */
export const focusFirstError = ({
  getStatuses,
  focus,
}: Pick<ContextValue, "getStatuses" | "focus">): void => {
  const first = getStatuses().find(({ variant }) => variant === "error");
  if (first == null) return;
  const target = first.key
    ?.split(deep.SEPARATOR)
    .reduce<deep.Path<unknown>>((parent, key) => parent[key], deep.path<unknown>());
  if (target == null || !focus(target, { optional: true }))
    throw new Error(first.message);
};
