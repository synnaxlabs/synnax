// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DataType, type TypedArray } from "@synnaxlabs/x";

export const randomSeries = (length: number, dataType: DataType): TypedArray => {
  // create random bytes of the correct length
  const bytes = new Uint8Array(length * dataType.density.valueOf());
  for (let i = 0; i < bytes.byteLength; i++) bytes[i] = Math.floor(Math.random() * 256);
  return new dataType.Array(bytes.buffer);
};
