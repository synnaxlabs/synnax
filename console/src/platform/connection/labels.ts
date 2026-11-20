// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { connection } from "@synnaxlabs/client";

/** How each non-error connection variant reads to the user. */
export const VARIANT_LABELS: Record<connection.NonErrorStatus["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  disabled: "Disconnected",
};

const reasonLabel = (reason: connection.Reason): string => {
  switch (reason) {
    case "unreachable":
      return "Unreachable";
    case "auth":
      return "Unauthenticated";
    case "incompatible":
      return "Incompatible";
    case "unlicensed":
      return "Unlicensed";
    default: {
      const unknown: never = reason;
      throw new Error(`unknown connection reason: ${String(unknown)}`);
    }
  }
};

/**
 * Returns the short label for a connection status. An error status reads as the reason
 * the connection failed.
 */
export const label = (status: connection.Status): string =>
  status.variant === "error"
    ? reasonLabel(status.details.reason)
    : VARIANT_LABELS[status.variant];
