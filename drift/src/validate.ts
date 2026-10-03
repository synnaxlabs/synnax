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

/** Ensures an action is valid, and throws an error if it is not. */
export const validateAction = (action?: UnknownAction): void => {
  if (action == null) throw new Error(undefinedActionMessage);
  if (action.type == null || action.type.length === 0)
    throw new Error(undefinedActionTypeMessage);
};
