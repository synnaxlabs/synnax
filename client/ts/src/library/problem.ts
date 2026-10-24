// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** One rule a library document breaks. */
export const problemZ = z.object({
  /**
   * The dotted location of the bad value, such as "messages.0.name", or empty for a
   * problem with the whole document.
   */
  path: z.string(),
  /** What is wrong. */
  message: z.string(),
});
export interface Problem extends z.infer<typeof problemZ> {}
