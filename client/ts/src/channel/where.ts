// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { isCalculated } from "@/channel/payload";
import { type Lines } from "@/channel/where.gen";

/** The typed lines of a channel Where. */
export const LINES: Lines = {
  virtual: (ch, value) => (ch.virtual && !isCalculated(ch)) === value,
  namePatterns: null,
};
