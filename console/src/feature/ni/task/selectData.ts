// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record } from "@synnaxlabs/x";

/**
 * Turns a name lookup into the entries a select renders, in declaration order. Declare
 * the lookup as `satisfies Record<Enum, string>` so a new enum value fails the build
 * until it has a name.
 */
export const selectData = <K extends record.Key>(
  names: Record<K, string>,
): record.KeyedNamed<K>[] =>
  record.entries(names).map(([key, name]) => ({ key, name }));
