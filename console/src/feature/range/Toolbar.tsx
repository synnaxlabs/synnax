// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { Content } from "@/feature/range/toolbar/Content";
import { type Nav } from "@/platform/nav";

export const TOOLBAR: Nav.Toolbar = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  trigger: ["R"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  useVisible: () => Access.useRetrieveGranted(ranger.TYPE_RESOURCE_ID),
};
