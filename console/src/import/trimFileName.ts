// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

const NAME_ENDING = ".json";
const NAME_ENDING_LENGTH = NAME_ENDING.length;

export const trimFileName = (name: string) => {
  if (name.endsWith(NAME_ENDING)) return name.slice(0, -NAME_ENDING_LENGTH);
  return name;
};
