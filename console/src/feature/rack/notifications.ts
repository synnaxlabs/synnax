// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Notifications } from "@/platform/notifications";

/**
 * The notification rule for rack statuses. A rack is always in use. Its warnings wait,
 * because the Core gives each new rack a warning until its Driver reports.
 */
export const NOTIFICATION_RULE: Notifications.Rule = {
  kind: () => "state",
  warningsDelayed: true,
};
