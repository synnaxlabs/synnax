// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status, type Synnax, TimeSpan } from "@synnaxlabs/client";
import { z } from "zod";

import {
  LIST_ASSETS_COMMAND_TYPE,
  SCAN_SCHEMAS,
  SCAN_TYPE,
} from "@/feature/sift/task/types";

const LIST_TIMEOUT = TimeSpan.seconds(15);

const listResultZ = z.object({
  /** The names of the assets the device can see, in the order Sift lists them. */
  assets: z.string().array(),
  /** The asset of the newest upload from the device, or empty. */
  last: z.string(),
});

export interface ListResult extends z.infer<typeof listResultZ> {}

/**
 * Lists the Sift assets that a device can see, through the scan task on its rack.
 *
 * @throws {Error} if the scan task fails the command, such as when Sift is unreachable.
 */
export const list = async (client: Synnax, key: device.Key): Promise<ListResult> => {
  const dev = await client.devices.retrieve({ key });
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: dev.rack,
    one: true,
    schemas: SCAN_SCHEMAS,
  });
  const res = await scanTask.executeCommandSync({
    type: LIST_ASSETS_COMMAND_TYPE,
    timeout: LIST_TIMEOUT,
    args: { device: key },
  });
  if (res.variant === "error") throw status.toError(res);
  return listResultZ.parse(res.details?.data);
};
