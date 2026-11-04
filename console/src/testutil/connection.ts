// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, license, MissingLicenseError } from "@synnaxlabs/client";

export const UNLICENSED_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "error",
  message: license.STATE_MESSAGES.missing,
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    reason: "unlicensed",
    error: new MissingLicenseError(license.STATE_MESSAGES.missing),
  },
};

/** A connection from a client 0.61.0 to a Core 0.60.1. */
export const INCOMPATIBLE_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "success",
  message: "Connected",
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    clientVersion: "0.61.0",
    nodeVersion: "0.60.1",
    clientServerCompatible: false,
  },
};

/** An unlicensed Core 0.60.1, as a check from a client 0.61.0 reports it. */
export const INCOMPATIBLE_UNLICENSED_STATUS: connection.Status = {
  ...UNLICENSED_STATUS,
  details: {
    ...UNLICENSED_STATUS.details,
    clientVersion: "0.61.0",
    nodeVersion: "0.60.1",
    clientServerCompatible: false,
  },
};
