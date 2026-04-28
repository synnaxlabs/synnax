// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This file holds the iterator wire-format types so that both iterator.ts and
// codec.ts can import them without creating a cycle through adapter.ts. It is
// expected to be replaced by an Oracle-generated types.gen.ts entry in the
// future, mirroring the writer's WriterCommand pattern.

import { errors, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { channel } from "@/channel";
import { frameZ } from "@/framer/frame";

export enum IteratorCommand {
  Open = 0,
  Next = 1,
  Prev = 2,
  SeekFirst = 3,
  SeekLast = 4,
  SeekLE = 5,
  SeekGE = 6,
  Valid = 7,
  Error = 8,
}

export enum IteratorResponseVariant {
  None = 0,
  Ack = 1,
  Data = 2,
}

export const iteratorReqZ = z.object({
  command: z.enum(IteratorCommand),
  span: TimeSpan.z.optional(),
  bounds: TimeRange.z.optional(),
  stamp: TimeStamp.z.optional(),
  keys: channel.keyZ.array().optional(),
  chunkSize: z.number().optional(),
  downsampleFactor: z.int().optional(),
});

export interface IteratorRequest extends z.infer<typeof iteratorReqZ> {}

export const iteratorResZ = z.object({
  variant: z.enum(IteratorResponseVariant),
  ack: z.boolean(),
  command: z.enum(IteratorCommand),
  error: errors.payloadZ.optional().nullable(),
  frame: frameZ.optional(),
});

export interface IteratorResponse extends z.infer<typeof iteratorResZ> {}
