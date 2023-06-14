// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, LazyArray, SampleValue } from "@synnaxlabs/x";

export const convertArrays = (arrs: LazyArray[]): LazyArray[] =>
  arrs.map((a) => {
    let offset: SampleValue = 0;
    if (a.dataType.equals(DataType.TIMESTAMP)) offset = BigInt(-a.data[0]);
    return a.convert(DataType.FLOAT32, offset);
  });
