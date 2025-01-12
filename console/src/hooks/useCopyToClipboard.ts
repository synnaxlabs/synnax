// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";

export const useCopyToClipboard = (): ((text: string, name: string) => void) => {
  const addStatus = Status.useAggregator();
  const handleException = Status.useExceptionHandler();
  return (text: string, name: string) => {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        addStatus({
          variant: "success",
          message: `Copied ${name} to clipboard.`,
        });
      })
      .catch((e) => handleException(e, `Failed to copy ${name} to clipboard`));
  };
};
