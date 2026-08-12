// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, task } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { type z } from "zod";

import { useStatus } from "@/platform/task/useStatus";

/**
 * {@link task.drifted} against the values the form holds. Both hashes are
 * server-assigned, so an edit surfaces here once its autosave lands, not on keystroke.
 */
export const useDrifted = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
): boolean => {
  const configHash = Form.useFieldValue<string>("configHash", { ctx, optional: true });
  const rack = Form.useFieldValue<rack.Key>("rack", { ctx, optional: true }) ?? 0;
  const status = useStatus(ctx);
  // A form that has not loaded its task yet holds no hash to compare.
  if (configHash == null) return false;
  return task.drifted({ configHash, rack, status });
};
