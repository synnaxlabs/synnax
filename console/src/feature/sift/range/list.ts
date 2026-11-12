// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, sift, status, type Synnax, TimeSpan } from "@synnaxlabs/client";
import { Flux, Status as PStatus } from "@synnaxlabs/pluto";
import { z } from "zod";

import { INTEGRATION } from "@/feature/sift/device/types";
import { KEY_PREFIX } from "@/feature/sift/range/types";
import {
  LIST_UPLOADS_COMMAND_TYPE,
  SCAN_SCHEMAS,
  SCAN_TYPE,
} from "@/feature/sift/task/types";

const LIST_TIMEOUT = TimeSpan.seconds(15);

const listResultZ = z.object({ uploads: status.keyZ.array() });

/**
 * Lists the keys of the statuses of the Sift uploads of a range, newest first, through
 * the scan task of a Core rack. Empty when no Core rack runs Sift.
 *
 * @throws if the scan task fails the command.
 */
const list = async (client: Synnax, range: ranger.Key): Promise<status.Key[]> => {
  const [rack] = await client.racks.retrieve({
    where: { embedded: { eq: true }, integrations: { any: [INTEGRATION] } },
    limit: 1,
  });
  if (rack == null) return [];
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: rack.key,
    one: true,
    schemas: SCAN_SCHEMAS,
  });
  const res = await scanTask.executeCommandSync({
    type: LIST_UPLOADS_COMMAND_TYPE,
    timeout: LIST_TIMEOUT,
    args: { range },
  });
  if (res.variant === "error") throw status.toError(res);
  return listResultZ.parse(res.details?.data).uploads;
};

type ListQuery = { range: ranger.Key };

const { useResult, useRefresh } = Flux.createRetrieve<ListQuery, status.Key[]>({
  name: "Sift uploads",
  retrieve: async ({ client, query: { range } }) => await list(client, range),
});

const rangeOf = (s: status.Status): ranger.Key | undefined => {
  const details = sift.uploadDetailsZ.safeParse("details" in s ? s.details : null);
  return details.success ? details.data.range : undefined;
};

/**
 * Lists the keys of the Sift uploads of a range, newest first, and keeps them live. A
 * null range lists nothing.
 */
export const useList = (range: ranger.Key | null): Flux.Result<status.Key[]> => {
  const res = useResult(range == null ? null : { range });
  const keys = res.data ?? [];
  const refresh = useRefresh();
  PStatus.useSetSynchronizer((s) => {
    if (range == null || !s.key.startsWith(KEY_PREFIX) || keys.includes(s.key)) return;
    if (rangeOf(s) === range) refresh({ range });
  });
  PStatus.useDeleteSynchronizer((key) => {
    if (range != null && keys.includes(key)) refresh({ range });
  });
  return res;
};
