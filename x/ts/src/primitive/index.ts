// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:client/ts/src/testutil/telem.ts
import { TimeStamp } from "@synnaxlabs/x";

export const secondsLinspace = (start: number, n: number): TimeStamp[] =>
  Array.from({ length: n }, (_, i) => start + i).map((n) => TimeStamp.seconds(n));
========
export * as primitive from "@/primitive/primitive";
>>>>>>>> 413a05fe0ab771a45f9260dc04bb6d5356fa0034:x/ts/src/primitive/index.ts
