// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x/telem";
import { z } from "zod";

import { label } from "@/label";

export const keyZ = z.uuid();
export type Key = z.infer<typeof keyZ>;
export const nameZ = z.string().min(1);
export type Name = z.infer<typeof nameZ>;
export type Keys = Key[];
export type Names = Name[];
export type Params = Key | Name | Keys | Names;

export const payloadZ = z.object({
  key: keyZ,
  name: nameZ,
  timeRange: TimeRange.z,
  color: z.string().optional(),
  labels: label.labelZ
    .array()
    .or(z.null().transform(() => undefined))
    .optional(),
  get parent(): z.ZodUnion<readonly [z.ZodNull, typeof payloadZ]> {
    // Using as unknown is bad, but unfortunately resolving the output type of this
    // transform is nearly impossible.
    return payloadZ
      .optional()
      .nullable()
      .transform((p) => (p === undefined ? null : p)) as unknown as z.ZodUnion<
      readonly [z.ZodNull, typeof payloadZ]
    >;
  },
});

export type Payload = z.infer<typeof payloadZ>;

export const newZ = payloadZ
  .omit({ parent: true, labels: true })
  .partial({ key: true });
export interface New extends z.input<typeof newZ> {}
