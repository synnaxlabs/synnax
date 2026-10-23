// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { type Client } from "@/components/client/choice";

export interface Info {
  key: Client;
  name: string;
  icon: Icon.ReactElement;
}

export const INFO: Info[] = [
  { key: "console", name: "Console", icon: <Icon.Visualize /> },
  { key: "python", name: "Python", icon: <Icon.Python /> },
  { key: "typescript", name: "TypeScript", icon: <Icon.TypeScript /> },
  { key: "cpp", name: "C++", icon: <Icon.CPlusPlus /> },
];
