// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { pointerZ, primitiveTypeZ } from "@/json/json";
import { dataTypeZ } from "@/telem";

/** The encodings of a timestamp in a JSON value. */
export const TIME_FORMATS = [
  "iso8601",
  "unix_sec",
  "unix_ms",
  "unix_us",
  "unix_ns",
] as const;

export const timeFormatZ = z.enum(TIME_FORMATS);

export type TimeFormat = z.infer<typeof timeFormatZ>;

/** A field that the Core inferred from a set of JSON documents. */
export const inferredFieldZ = z.object({
  /** pointer locates the field in each document. */
  pointer: pointerZ,
  /** dataType is the data type of a channel that holds the field. */
  dataType: dataTypeZ,
  /** type is the JSON type of most values of the field. */
  type: primitiveTypeZ.exclude(["null"]),
  /** timeFormat is the encoding of a timestamp field. Undefined for any other field. */
  timeFormat: timeFormatZ.optional(),
  /** enum holds the labels of a string field with few distinct values. */
  enum: z.array(z.string()),
  /** count is the number of documents that hold the field. */
  count: z.number(),
  /** mismatched is the number of values that do not have the JSON type of the field. */
  mismatched: z.number(),
});

export interface InferredField extends z.infer<typeof inferredFieldZ> {}

/** The fields shared by documents from sources that hold the same set of fields. */
export const shapeZ = z.object({
  /** sources name the sources of the documents, such as topics or record keys. */
  sources: z.array(z.string()),
  /** fields are the fields of the documents, in the order they first appear. */
  fields: z.array(inferredFieldZ),
  /** index is the position in fields of the timestamp that indexes the documents. */
  index: z.number().transform((i) => (i < 0 ? undefined : i)),
  /** count is the number of documents. */
  count: z.number(),
});

export interface Shape extends z.infer<typeof shapeZ> {}
