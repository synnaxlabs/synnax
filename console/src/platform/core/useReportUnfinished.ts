// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { errors } from "@synnaxlabs/x";

import { UnfinishedLoginError } from "@/platform/core/oidc";

/**
 * @returns a function that reports a provider login the person did not finish as
 * information under the message. It throws any other error back.
 */
export const useReportUnfinished = (): ((e: unknown, message: string) => void) => {
  const addStatus = Status.useAdder();
  return (e, message) => {
    const err = errors.fromUnknown(e);
    if (!UnfinishedLoginError.matches(err)) throw err;
    addStatus({ variant: "info", message, description: err.message });
  };
};
