// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import { type connection } from "@/connection";

/** Resolves once the handle's status satisfies the predicate. */
export const waitForStatus = async (
  handle: connection.Handle,
  predicate: (status: connection.Status) => boolean,
  timeout: TimeSpan = TimeSpan.seconds(5),
): Promise<connection.Status> => {
  if (predicate(handle.status)) return handle.status;
  return await new Promise<connection.Status>((resolve, reject) => {
    const timer = setTimeout(() => {
      detach();
      reject(new Error("timed out waiting for connection status"));
    }, timeout.milliseconds);
    const detach = handle.onChange((status) => {
      if (!predicate(status)) return;
      clearTimeout(timer);
      detach();
      resolve(status);
    });
  });
};
