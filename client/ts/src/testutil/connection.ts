// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type TimeSpan } from "@synnaxlabs/x";

import { type connection } from "@/connection";

/**
 * Resolves once the handle's change stream is live, so every later cluster write
 * reaches the client's cache.
 */
export const waitForStreamLive = async (
  handle: connection.Handle,
  timeout?: TimeSpan,
): Promise<connection.Status> =>
  await observe.until(
    handle,
    () => handle.status,
    ({ variant, details }) => variant === "success" && details.streamLive,
    timeout,
  );
