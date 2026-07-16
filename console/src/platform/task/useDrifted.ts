// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { type z } from "zod";

import { useStatus } from "@/platform/task/useStatus";

/**
 * Whether the running instance was deployed with a different config or rack than the
 * saved task now holds. Tasks that are not running never drift. Both hashes are
 * server-assigned, so an edit surfaces here once its autosave lands, not on keystroke.
 */
export const useDrifted = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
): boolean => {
  const configHash = Form.useFieldValue<string>("configHash", { ctx, optional: true });
  const rackKey = Form.useFieldValue<rack.Key>("rack", { ctx, optional: true });
  const {
    running,
    configHash: deployedHash,
    rack: deployedRack,
  } = useStatus(ctx).details;
  if (!running || configHash == null) return false;
  return configHash !== deployedHash || (rackKey ?? 0) !== deployedRack;
};
