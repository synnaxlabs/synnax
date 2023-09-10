// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { registerCustomTypeEncoder } from "@/binary";
import { DataType, Density, Rate, TimeSpan, TimeStamp } from "@/telem/telem";

const valueOfEncoder = (value: unknown): unknown => value?.valueOf();

registerCustomTypeEncoder({ Class: TimeStamp, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: TimeSpan, write: valueOfEncoder });
registerCustomTypeEncoder({
  Class: DataType,
  write: (v: unknown) => (v as DataType).toString,
});
registerCustomTypeEncoder({ Class: Rate, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: Density, write: valueOfEncoder });
