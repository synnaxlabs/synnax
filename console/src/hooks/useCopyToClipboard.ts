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

export const useCopyToClipboard = (): ((text: string, name: string) => void) => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (text: string, name: string) => {
      handleError(async () => {
        await navigator.clipboard.writeText(text);
        addStatus({ variant: "success", message: `Copied ${name} to clipboard.` });
      }, `Failed to copy ${name} to clipboard`);
    },
    [addStatus, handleError],
  );
};
