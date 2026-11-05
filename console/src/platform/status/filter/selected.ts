// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";

/**
 * @returns the variants that set selects. A variant must match every member that set
 * holds.
 */
export const getSelected = (set?: status.Where["variant"]): status.Variant[] => {
  if (set?.eq == null) return set?.in ?? [];
  return set.in == null || set.in.includes(set.eq) ? [set.eq] : [];
};
