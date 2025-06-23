// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod/v4";

import { id } from "@/id";
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

const unknownOptional = z.unknown().optional();

export const statusZ = <D extends z.ZodType = typeof unknownOptional>(
  details: D = unknownOptional as unknown as D,
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

export type Crude<D = undefined> = Optional<Status<D>, "time" | "key">;

interface ExceptionDetails {
  stack: string;
}

export const fromException = (
  exc: unknown,
  message?: string,
): Status<ExceptionDetails> => {
  if (!(exc instanceof Error)) throw exc;
  return create<ExceptionDetails>({
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
    details: {
      stack: exc.stack ?? "",
    },
  });
};

export const create = <D = undefined>(spec: Crude<D>): Status<D> =>
  ({
    key: id.create(),
    time: TimeStamp.now(),
    ...spec,
  }) as unknown as Status<D>;
