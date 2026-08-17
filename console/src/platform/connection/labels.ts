// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";

/** How each connection variant reads to the user. */
export const STATUS_LABELS: Record<connection.Status["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  error: "Unreachable",
  disabled: "Disconnected",
};
