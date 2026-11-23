// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { json, payload } from "@synnaxlabs/x";

import type { Device } from "@/feature/mqtt/device";

/** A value that a browsed payload offers. */
export interface Leaf extends Pick<
  json.InferredField,
  "pointer" | "type" | "kind" | "timeFormat"
> {}

/** The kind of a browsed value that the Core gave none, by its JSON type. */
const SAMPLE_KINDS = {
  number: "float",
  boolean: "boolean",
  string: "string",
} as const satisfies Record<payload.JSONType, payload.Kind>;

/** @returns one leaf for each scalar of a browsed payload. */
export const createLeaves = (scalars: Device.BrowsedField[]): Leaf[] =>
  scalars.map(({ pointer, type, kind = SAMPLE_KINDS[type], timeFormat }) => ({
    pointer,
    type,
    kind,
    timeFormat,
  }));

/** @returns the leaf of a payload that is not JSON, which reads it as one string. */
export const createRawLeaf = (): Leaf => ({
  pointer: "",
  type: "string",
  kind: "string",
});
