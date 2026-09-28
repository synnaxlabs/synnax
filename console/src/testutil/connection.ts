// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, MissingLicenseError } from "@synnaxlabs/client";

export const UNLICENSED_MESSAGE = "No license is active on this Core";

/** The status of a client whose Core refuses requests for want of a license. */
export const UNLICENSED_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "error",
  message: UNLICENSED_MESSAGE,
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    reason: "unlicensed",
    error: new MissingLicenseError(UNLICENSED_MESSAGE),
  },
};
