// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";

import { Sync } from "@/sync";

export const useStatusSynchronizer = (
  onStatusChange: (status: rack.Status) => void,
): void =>
  Sync.useParsedListener(rack.STATUS_CHANNEL_NAME, rack.statusZ, onStatusChange);
