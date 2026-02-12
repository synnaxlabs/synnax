// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export * from "@/ranger/alias";
export {
  /** @deprecated Use {@link alias.createKey} instead. */
  createKey as aliasKey,
  /** @deprecated Use {@link alias.decodeDeleteChange} instead. */
  decodeDeleteChange as decodeDeleteAliasChange,
  /** @deprecated Use {@link alias.DELETE_CHANNEL_NAME} instead. */
  DELETE_CHANNEL_NAME as DELETE_ALIAS_CHANNEL_NAME,
  /** @deprecated Use {@link SET_CHANNEL_NAME} instead. */
  SET_CHANNEL_NAME as SET_ALIAS_CHANNEL_NAME,
} from "@/ranger/alias/payload";
export * from "@/ranger/client";
export * from "@/ranger/kv";
export {
  /** @deprecated Use {@link kv.DELETE_CHANNEL_NAME} instead. */
  DELETE_CHANNEL_NAME as KV_DELETE_CHANNEL,
  /** @deprecated Use {@link kv.SET_CHANNEL_NAME} instead. */
  SET_CHANNEL_NAME as KV_SET_CHANNEL,
  /** @deprecated Use {@link kv.Pair} instead. */
  type Pair as KVPair,
  /** @deprecated Use {@link kv.createPairKey} instead. */
  createPairKey as kvPairKey,
  /** @deprecated Use {@link kv.pairZ} instead. */
  pairZ as kvPairZ,
} from "@/ranger/kv/payload";
export * from "@/ranger/payload";
export * from "@/ranger/writer";
