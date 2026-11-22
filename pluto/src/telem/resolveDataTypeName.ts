// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, strings } from "@synnaxlabs/x";

const ALL_CAPS = [DataType.UUID, DataType.JSON];

/** Returns the name a person reads for a data type, such as "Timestamp". */
export const resolveDataTypeName = (d: DataType): string => {
  if (ALL_CAPS.some((c) => c.equals(d))) return d.toString().toUpperCase();
  if (d.isNumeric && !d.equals(DataType.TIMESTAMP) && !d.equals(DataType.BOOLEAN))
    return d.toString();
  return strings.capitalize(d.toString());
};
