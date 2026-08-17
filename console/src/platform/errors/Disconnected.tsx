// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { isConnectionError as clientIsConnectionError } from "@synnaxlabs/client";
import { Flex, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Connection } from "@/platform/connection";

// Deep enough for the wrappers a failed read collects, short enough to stop a
// cycle from spinning.
const MAX_DEPTH = 8;

// Flux wraps a failed retrieve, and status.toError nests the original under
// details.error, so the reachability verdict lives below the caught error.
const causeOf = (error: unknown): unknown => {
  if (error instanceof Error) return error.cause;
  if (typeof error !== "object" || error == null || !("details" in error)) return null;
  const { details } = error;
  if (typeof details !== "object" || details == null || !("error" in details))
    return null;
  return details.error;
};

/** Reports whether the failure is the Core being unreachable, at any depth. */
export const isConnectionError = (error: unknown): boolean => {
  let current: unknown = error;
  for (let i = 0; i < MAX_DEPTH && current != null; i++) {
    if (clientIsConnectionError(current)) return true;
    current = causeOf(current);
  }
  return false;
};

/**
 * Stands in for the crash fallback when the read failed because the Core is
 * unreachable. Boundaries reset on the next connection epoch, so the panel
 * recovers by itself once the Core answers again.
 */
export const Disconnected = (): ReactElement => {
  const { details } = Synnax.useConnectionStatus();
  return (
    <Flex.Box y center gap="medium">
      <Connection.Target />
      <Connection.RetrySchedule details={details} />
      <Connection.Retry variant="outlined" size="small" />
    </Flex.Box>
  );
};
