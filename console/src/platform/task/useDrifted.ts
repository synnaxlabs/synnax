// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import type { z } from "zod";

import { formPath } from "@/platform/task/types";
import { useStatus } from "@/platform/task/useStatus";

/**
 * {@link task.drifted} against the values the form holds. Both hashes are
 * server-assigned, so an edit surfaces here once its autosave lands, not on keystroke.
 */
export const useDrifted = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
): boolean => {
  const configHash = Form.useFieldValue(formPath.configHash, { ctx });
  const rack = Form.useFieldValue(formPath.rack, { ctx });
  const status = useStatus(ctx);
  return task.drifted({ configHash, rack, status });
};

/**
 * Whether a start deploys the config and rack that the form holds: they differ from
 * those the task last ran with, the task never ran, or its last start failed.
 */
export const useUndeployed = (): boolean => {
  const configHash = Form.useFieldValue(formPath.configHash);
  const rack = Form.useFieldValue(formPath.rack);
  const status = useStatus();
  if (status == null) return true;
  const { variant, details } = status;
  // A failed start reports the hash that it tried, though nothing ran.
  if (variant === "error" && !details.running) return true;
  return details.configHash !== configHash || details.rack !== rack;
};
