// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export * from "@/ranger/alias";
export * from "@/ranger/client";
export * from "@/ranger/kv";
export * from "@/ranger/payload";
export * from "@/ranger/writer";

// Deprecated re-exports from kv - these previously lived at the ranger level
export {
  KV_SET_CHANNEL,
  KV_DELETE_CHANNEL,
  kvPairZ,
  type KVPair,
  kvPairKey,
} from "@/ranger/kv/payload";

// Deprecated re-exports from alias - these previously lived at the ranger level
export {
  SET_ALIAS_CHANNEL_NAME,
  DELETE_ALIAS_CHANNEL_NAME,
  aliasKey,
  decodeDeleteAliasChange,
} from "@/ranger/alias/payload";
