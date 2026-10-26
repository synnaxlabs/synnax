// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { useUsageDataModal } from "@/feature/account/useUsageDataModal";
import { Command } from "@/platform/command";

const OpenUsageDataCommand = Command.create({
  key: "open_usage_data",
  name: "Change usage data settings",
  icon: <Icon.Analyze />,
  useOnSelect: useUsageDataModal,
});

export const COMMANDS = [OpenUsageDataCommand];
