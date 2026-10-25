// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clipboard } from "@synnaxlabs/lyra/clipboard";
import { Status } from "@synnaxlabs/lyra/status";

export const useCopy = (): ((text: string, what: string, name?: string) => void) => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  return (text: string, what: string, name = "") => {
    handleError(
      async () => {
        await clipboard.writeText(text);
        addStatus({
          variant: "success",
          name,
          message: `Copied ${what} to clipboard`,
        });
      },
      { message: `Failed to copy ${what} to clipboard`, name },
    );
  };
};
