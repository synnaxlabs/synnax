// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const sortFunc = <V>(t: string): ((a: V, b: V) => number) => {
  switch (t) {
    case "string":
      return (a: V, b: V) => (a as string).localeCompare(b as string);
    case "number":
      return (a: V, b: V) => (a as number) - (b as number);
    default:
      console.warn("sortFunc: unknown type");
      return () => 0;
  }
};
