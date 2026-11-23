// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type payload } from "@synnaxlabs/x";

import type { BrowsedField } from "@/feature/mqtt/task/types";

/** A value that a browsed payload offers. */
export interface Leaf {
  /** The JSON pointer of the value. */
  pointer: string;
  /** The JSON type of the value's sample. */
  type: payload.JSONType;
  /** The data type of a channel that reads the value. */
  dataType: string;
}

const DATA_TYPES: Record<payload.JSONType, string> = {
  number: DataType.FLOAT64.toString(),
  boolean: DataType.UINT8.toString(),
  string: DataType.STRING.toString(),
};

const createLeaf = (pointer: string, type: payload.JSONType): Leaf => ({
  pointer,
  type,
  dataType: DATA_TYPES[type],
});

/** @returns one leaf for each scalar of a browsed payload. */
export const createLeaves = (scalars: BrowsedField[]): Leaf[] =>
  scalars.map(({ pointer, type }) => createLeaf(pointer, type));

/** @returns the leaf of a payload that is not JSON, which reads it as one string. */
export const createRawLeaf = (): Leaf => createLeaf("", "string");
