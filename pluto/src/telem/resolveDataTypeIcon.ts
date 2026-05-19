// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { telem } from "@synnaxlabs/x/telem";

export const resolveDataTypeIcon = (d: telem.DataType): Icon.FC | undefined => {
  if (d.equals(telem.DataType.JSON)) return Icon.JSON;
  if (d.equals(telem.DataType.BYTES)) return Icon.Binary;
  if (d.isInteger) return Icon.Binary;
  if (d.isFloat) return Icon.Decimal;
  if (d.equals(telem.DataType.STRING) || d.equals(telem.DataType.UUID))
    return Icon.String;
  if (d.equals(telem.DataType.TIMESTAMP)) return Icon.Time;
  return undefined;
};
