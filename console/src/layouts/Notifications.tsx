// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Hardware } from "@/hardware";
import { Notifications as Core } from "@/notifications";
import { Version } from "@/version";

const NOTIFICATION_ADAPTERS: Core.Adapter[] = [
  ...Cluster.NOTIFICATION_ADAPTERS,
  ...Hardware.NOTIFICATION_ADAPTERS,
  ...Version.NOTIFICATION_ADAPTERS,
];

export const Notifications = (): ReactElement => (
  <Core.Notifications adapters={NOTIFICATION_ADAPTERS} />
);
