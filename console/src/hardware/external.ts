// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Device } from "@/hardware/device";
import { Task } from "@/hardware/task";
import { type Import } from "@/import";
import { type Layout } from "@/layout";
import { type Link } from "@/link";
import { type Notifications } from "@/notifications";
import { type Palette } from "@/palette";
import { type Selector } from "@/selector";

export * from "@/hardware/device";
export * from "@/hardware/rack";
export * from "@/hardware/task";

export const COMMANDS: Palette.Command[] = [...Device.COMMANDS, ...Task.COMMANDS];

export const EXTRACTORS: Export.Extractors = Task.EXTRACTORS;

export const FILE_INGESTORS: Import.FileIngestors = Task.FILE_INGESTORS;

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Device.LAYOUTS,
  ...Task.LAYOUTS,
};

export const LINK_HANDLERS: Record<string, Link.Handler> = {
  task: Task.handleLink,
};

export const NAV_DRAWER_ITEMS: Layout.NavDrawerItem[] = [
  ...Device.NAV_DRAWER_ITEMS,
  ...Task.NAV_DRAWER_ITEMS,
];

export const NOTIFICATION_ADAPTERS: Notifications.Adapter[] =
  Device.NOTIFICATION_ADAPTERS;

export const SELECTABLES: Selector.Selectable[] = Task.SELECTABLES;
