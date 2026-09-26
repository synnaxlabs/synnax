// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

/**
 * Returns a callback that asks the Desktop shell to open a directory in the file
 * manager, and reports a failure under the given message.
 */
export const useReveal = (
  reveal: () => Promise<void>,
  failureMessage: string,
): (() => void) => {
  const handleError = Status.useErrorHandler();
  return useCallback(
    () => handleError(reveal, failureMessage),
    [handleError, reveal, failureMessage],
  );
};
