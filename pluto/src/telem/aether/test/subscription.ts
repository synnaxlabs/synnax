// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type telem } from "@synnaxlabs/client";

/** @returns a telemetry subscription stub whose close is the given function. */
export const mockSubscription = (close: () => void): telem.Subscription => ({
  close,
  status: () => status.create({ variant: "loading", message: "subscribing" }),
  onStatusChange: () => () => {},
});
