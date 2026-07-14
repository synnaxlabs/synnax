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
import { useMemo } from "react";
import { type z } from "zod";

import { useStatus } from "@/platform/task/useStatus";

/**
 * Whether the running instance was deployed with a different config or rack
 * than the form now holds. Tasks that are not running never drift.
 */
export const useDrifted = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
): boolean => {
  const config = Form.useFieldValue<unknown>("config", { ctx, optional: true });
  const rackKey = Form.useFieldValue<rack.Key>("rack", { ctx, optional: true });
  const status = useStatus(ctx);
  return useMemo(() => {
    const { running, configHash, rack: deployedRack } = status.details;
    if (!running || config == null) return false;
    return task.hashConfig(config) !== configHash || (rackKey ?? 0) !== deployedRack;
  }, [config, rackKey, status]);
};
