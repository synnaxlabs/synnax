// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const encodeVarint = (value: number): Uint8Array => {
  const bytes: number[] = [];
  while (value >= 0x80) {
    bytes.push((value & 0x7f) | 0x80);
    value >>>= 7;
  }
  bytes.push(value);
  return new Uint8Array(bytes);
};

export const decodeVarint = (data: Uint8Array): [number, number] => {
  let result = 0;
  let shift = 0;
  let i = 0;
  while (i < data.length) {
    const byte = data[i];
    result |= (byte & 0x7f) << shift;
    if ((byte & 0x80) === 0) return [result, i + 1];
    shift += 7;
    i++;
  }
  throw new Error("incomplete varint");
};
