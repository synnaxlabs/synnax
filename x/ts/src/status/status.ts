// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod/v4";

import { type Optional } from "@/optional";
import { status } from "@/status";
import { TimeStamp } from "@/telem";

export const variantZ = z.enum([
  "success",
  "info",
  "warning",
  "error",
  "loading",
  "disabled",
]);

// Represents one of the possible variants of a status message.
export type Variant = z.infer<typeof variantZ>;

const undefinedOptional = z.undefined().optional();

export const statusZ = <D extends z.ZodType = typeof undefinedOptional>(
  details: D = undefinedOptional as unknown as D,
) =>
  z.object({
    key: z.string(),
    variant: status.variantZ,
    message: z.string(),
    description: z.string().optional(),
    time: TimeStamp.z,
    details,
  });

export type Status<D = undefined> = {
  key: string;
  variant: Variant;
  message: string;
  description?: string;
  time: TimeStamp;
} & (D extends undefined ? {} : { details: D });

export type New<D = undefined> = Optional<Status<D>, "time" | "key">;
