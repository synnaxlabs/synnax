// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type z } from "zod";

import { formPath } from "@/platform/task/types";

/** Returns undefined for a task with no status, which the Core allows. */
export const useStatus = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
): task.Status | undefined => Form.useFieldValue(formPath.status, { ctx }) ?? undefined;

/** Returns false for a task with no status. */
export const useIsRunning = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
): boolean => useStatus(ctx)?.details.running === true;
