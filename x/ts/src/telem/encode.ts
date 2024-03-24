// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary } from "@/binary";
import { DataType, Density, Rate, TimeSpan, TimeStamp } from "@/telem/telem";

const valueOfEncoder = (value: unknown): unknown => value?.valueOf();

binary.registerCustomTypeEncoder({ Class: TimeStamp, write: valueOfEncoder });
binary.registerCustomTypeEncoder({ Class: TimeSpan, write: valueOfEncoder });
binary.registerCustomTypeEncoder({
  Class: DataType,
  write: (v: unknown) => (v as DataType).toString(),
});
binary.registerCustomTypeEncoder({ Class: Rate, write: valueOfEncoder });
binary.registerCustomTypeEncoder({ Class: Density, write: valueOfEncoder });
