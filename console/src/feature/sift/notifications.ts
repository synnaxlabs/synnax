// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Task } from "@/feature/sift/task";
import { type Notifications } from "@/platform/notifications";

const RULE: Notifications.Rule = { kind: () => "state" };

/**
 * The notification rules of the Sift statuses that no thing owns. They notify only on
 * a problem, never on progress.
 */
export const NOTIFICATION_KEY_RULES: Notifications.KeyRules = {
  [Task.PROGRESS_KEY_PREFIX]: RULE,
};
