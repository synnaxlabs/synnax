// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

// --- VERY IMPORTANT ---
// Synnax's encoders (in the binary package inside x) automatically convert the case of
// keys in objects to snake_case and back to camelCase when encoding and decoding
// respectively. This is done to ensure that the keys are consistent across all
// languages and platforms. Sometimes items have keys that are uuids, which have dashes,
// and those get messed up. So we just use regular JSON for these items.
export const parseWithoutKeyConversion = (s: string): record.Unknown =>
  s ? JSON.parse(s) : {};
