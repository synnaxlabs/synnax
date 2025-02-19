// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";

import { Device } from "@/hardware/device";
import { Task } from "@/hardware/task";
import { type Layout } from "@/layout";
import { type Link } from "@/link";
import { type Notifications } from "@/notifications";
import { type Palette } from "@/palette";

export * from "@/hardware/device";
export * from "@/hardware/rack";
export * from "@/hardware/task";

export const COMMANDS: Palette.Command[] = [...Device.COMMANDS, ...Task.COMMANDS];

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Device.LAYOUTS,
  ...Task.LAYOUTS,
};

export const LINK_HANDLERS: Record<string, Link.Handler> = {
  [task.ONTOLOGY_TYPE]: Task.handleLink,
};

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [Task.Toolbar];

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] =
  Device.NOTIFICATION_ADAPTERS;

export const SELECTABLES: Layout.Selectable[] = Task.SELECTABLES;
