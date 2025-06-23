// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";

import { Sync } from "@/sync";

export const useSetSynchronizer = (onSet: (label: label.Label) => void): void =>
  Sync.useParsedListener(label.SET_CHANNEL_NAME, label.labelZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: label.Key) => void): void =>
  Sync.useParsedListener(label.DELETE_CHANNEL_NAME, label.keyZ, onDelete);
