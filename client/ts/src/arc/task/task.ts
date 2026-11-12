// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Key, keyZ } from "@/arc/types.gen";
import { type task } from "@/task";

/** The type of the task that runs an Arc. */
export const TYPE = "arc";

const arcKeyZ = z.object({ arcKey: keyZ });

/** Returns the key of the Arc that the task runs, or undefined for any other task. */
export const arcKeyOf = (t: Pick<task.Task, "type" | "config">): Key | undefined => {
  if (t.type !== TYPE) return undefined;
  const parsed = arcKeyZ.safeParse(t.config);
  return parsed.success ? parsed.data.arcKey : undefined;
};
