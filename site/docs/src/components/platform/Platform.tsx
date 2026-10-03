// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { type Platform } from "@/components/platform/choice";

export interface Info {
  key: Platform;
  name: string;
  icon: Icon.ReactElement;
}

export const INFO: Info[] = [
  { key: "Linux", name: "Linux", icon: <Icon.Logo.Linux /> },
  { key: "Windows", name: "Windows", icon: <Icon.Logo.Windows /> },
  { key: "macOS", name: "macOS", icon: <Icon.Logo.Apple /> },
  { key: "Docker", name: "Docker", icon: <Icon.Logo.Docker /> },
];
