// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";

import { Sync } from "@/sync";

export const useDeleteSynchronizer = (onDelete: (key: ranger.Key) => void): void =>
  Sync.useParsedListener(ranger.DELETE_CHANNEL_NAME, ranger.keyZ, onDelete);

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void =>
  Sync.useParsedListener(ranger.SET_CHANNEL_NAME, ranger.payloadZ, onSet);

export const useAliasSetSynchronizer = (onSet: (alias: ranger.Alias) => void): void =>
  Sync.useParsedListener(ranger.SET_ALIAS_CHANNEL_NAME, ranger.aliasZ, onSet);

export const useAliasDeleteSynchronizer = (
  onDelete: (alias: ranger.DecodedDeleteAliasChange) => void,
): void =>
  Sync.useStringListener(
    ranger.DELETE_ALIAS_CHANNEL_NAME,
    ranger.decodeDeleteAliasChange,
    onDelete,
  );
