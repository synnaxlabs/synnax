// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnknownAction } from "@reduxjs/toolkit";

const undefinedActionMessage = "[drift] - unexpected undefined action";
const undefinedActionTypeMessage = "[drift] - unexpected undefined action type";

/**
 * Ensures an action is valid, and throws an error if it is not.
 * @param a - The action to validate.
 */
export const validateAction = (meta: {
  emitted?: boolean;
  action?: UnknownAction;
  emitter?: string;
}): void => {
  meta.emitted ??= false;
  if (meta.action == null) {
    console.warn(undefinedActionMessage, meta);
    throw new Error(undefinedActionMessage);
  }
  if (meta.action.type == null || meta.action.type.length === 0) {
    console.warn(undefinedActionTypeMessage, meta);
    throw new Error(undefinedActionTypeMessage);
  }
};
