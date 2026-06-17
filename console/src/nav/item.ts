// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon, type Nav, type Triggers } from "@synnaxlabs/pluto";

export interface NavMenuItem {
  key: string;
  icon: Icon.ReactElement;
  tooltip: string;
  trigger: Triggers.Trigger;
  useVisible?: () => boolean;
}

export interface NavDrawerItem extends Nav.DrawerItem, NavMenuItem {}
