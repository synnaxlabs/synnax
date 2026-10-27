// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import { UNLINK_URL } from "@/platform/license/urls";

const TIMEOUT = TimeSpan.seconds(5);

/**
 * Asks the hub to log out the machine the renewal secret belongs to. The machine leaves
 * its account, and the secret stops working.
 * @throws {Error} if the hub refuses, or does not answer within five seconds.
 */
export const unlink = async (secret: string): Promise<void> => {
  const res = await fetch(UNLINK_URL, {
    method: "POST",
    headers: { authorization: `Bearer ${secret}` },
    signal: AbortSignal.timeout(TIMEOUT.milliseconds),
  });
  if (!res.ok) throw new Error(`Could not log out: ${res.status} ${res.statusText}`);
};
