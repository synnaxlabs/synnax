// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { useEffect } from "react";

import { Core } from "@/feature/core";
import { Device } from "@/feature/device";
import { Rack } from "@/feature/rack";
import { Task } from "@/feature/task";
import { Notifications } from "@/platform/notifications";
import { Version } from "@/platform/version";

const NOTIFICATIONS: Notifications.Notification[] = [
  ...(DESKTOP ? [] : Core.NOTIFICATIONS),
  ...Device.NOTIFICATIONS,
  ...Rack.NOTIFICATIONS,
  ...Task.NOTIFICATIONS,
  ...Version.NOTIFICATIONS,
];

export const Feed = () => <Notifications.Feed notifications={NOTIFICATIONS} />;

const DEMO_SEED: status.Crude[] = [
  {
    variant: "error",
    name: "Pressure test",
    message: "Task failed to start",
    description: "Device Dev1 is not connected. Check the cable and try again.",
  },
  {
    variant: "warning",
    name: "Synnax Driver",
    message: "Driver on rack Test Stand 2 is not running",
  },
  { variant: "success", message: "Exported Schematic to schematic.json" },
  {
    variant: "info",
    message: "A new version of the Console is available",
    description: "Version 0.60.0 is ready to install.",
  },
  { variant: "loading", name: "Calibration", message: "Configuring task" },
];

const DEMO_DRIP: status.Crude[] = [
  { variant: "error", message: "Failed to write to channel ox_tpc_cmd" },
  { variant: "info", name: "Range", message: "Hotfire 14 started" },
  { variant: "warning", message: "Sample rate is above the hardware limit" },
  { variant: "success", name: "Arc", message: "Sequence press_and_hold deployed" },
];

const DEMO_INTERVAL_MS = 20_000;

const useDemoStatuses = (): void => {
  const add = Status.useAdder();
  useEffect(() => {
    if (!IS_DEV) return;
    DEMO_SEED.forEach((s) => add(s));
    let i = 0;
    const interval = setInterval(() => {
      add(DEMO_DRIP[i % DEMO_DRIP.length]);
      i++;
    }, DEMO_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [add]);
};

export const Inbox = () => {
  useDemoStatuses();
  return <Notifications.Inbox notifications={NOTIFICATIONS} />;
};
