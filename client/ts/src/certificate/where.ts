// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Lines } from "@/certificate/where.gen";

/** The typed lines of a certificate Where. */
export const LINES: Lines = {
  racks: (item, keys) => item.source === "file" && keys.includes(item.rack),
};
