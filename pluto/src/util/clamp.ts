// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const clamp = (value: number, min?: number, max?: number): number => {
  if (min != null) value = Math.max(value, min);
  if (max != null) value = Math.min(value, max);
  return value;
};
