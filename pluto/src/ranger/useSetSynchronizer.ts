// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";

import { Synch } from "@/synch";

export const useSetSynchronizer = (onSet: (range: ranger.Payload) => void): void =>
  Synch.useParsedListener(ranger.SET_CHANNEL_NAME, ranger.payloadZ, onSet);
