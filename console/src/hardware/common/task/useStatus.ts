// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/pluto";
import { id, TimeStamp } from "@synnaxlabs/x";
import { type z } from "zod";

const defaultStatus = <StatusData extends z.ZodType>(): task.Status<
  ReturnType<typeof task.statusDetailsZ<StatusData>>
> => ({
  key: id.create(),
  name: "Task Status",
  variant: "disabled",
  message: "Task has not been configured",
  time: TimeStamp.now(),
  details: { task: "", running: false, data: {} as any },
});

export const useStatus = <Schema extends z.ZodType>(ctx?: Form.ContextValue<Schema>) =>
  Form.useFieldValue<task.Status>("status", { ctx, optional: true }) ?? defaultStatus();
