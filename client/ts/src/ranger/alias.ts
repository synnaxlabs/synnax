// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Re-export from the top-level alias module for backward compatibility
export {
  Aliaser,
  DELETE_CHANNEL_NAME as DELETE_ALIAS_CHANNEL_NAME,
  SET_CHANNEL_NAME as SET_ALIAS_CHANNEL_NAME,
} from "@/alias/client";
export {
  type Alias,
  type AliasChange,
  aliasKey,
  aliasZ,
  decodeDeleteAliasChange,
  type DecodedDeleteAliasChange,
} from "@/alias/payload";
