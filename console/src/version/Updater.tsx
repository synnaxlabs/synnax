// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon, Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { id, TimeSpan } from "@synnaxlabs/x";
import { check } from "@tauri-apps/plugin-updater";
import { useState, useSyncExternalStore } from "react";

import { Layout } from "@/layout";
import { type Notifications } from "@/notifications";
import { Runtime } from "@/runtime";
import { INFO_LAYOUT } from "@/version/Info";

const STATUS_KEY_PREFIX = "versionUpdate";

// Update-notification silencing is session-local: snoozing suppresses notifications
// until the Console is reloaded.
let silenced = false;
const silenceListeners = new Set<() => void>();

const silenceUpdateNotifications = (): void => {
  if (silenced) return;
  silenced = true;
  silenceListeners.forEach((listener) => listener());
};

const useUpdateNotificationsSilenced = (): boolean =>
  useSyncExternalStore(
    (listener) => {
      silenceListeners.add(listener);
      return () => silenceListeners.delete(listener);
    },
    () => silenced,
  );

export const useCheckForUpdates = (): boolean => {
  const addStatus = Status.useAdder();
  const isSilenced = useUpdateNotificationsSilenced();
  const [available, setAvailable] = useState(false);

  const checkForUpdates = async (addNotification: boolean) => {
    if (Runtime.ENGINE !== "tauri") return;
    if (available) return;
    const update = await check();
    if (update == null) return;
    setAvailable(true);
    if (addNotification)
      addStatus({
        key: `${STATUS_KEY_PREFIX}-${id.create()}`,
        variant: "info",
        message: `Update available`,
      });
  };

  useAsyncEffect(
    async (signal) => {
      await checkForUpdates(!isSilenced);
      if (signal.aborted) return;
      const i = setInterval(
        () => void checkForUpdates(!isSilenced),
        TimeSpan.seconds(30).milliseconds,
      );
      return () => clearInterval(i);
    },
    [isSilenced],
  );

  return available;
};

export const notificationAdapter: Notifications.Adapter = (status, silence) => {
  if (!status.key.startsWith(STATUS_KEY_PREFIX)) return null;
  return {
    ...status,
    actions: [
      <OpenUpdateDialogAction key="update" />,
      <SilenceAction key="silence" onClick={() => silence(status.key)} />,
    ],
  };
};

export const OpenUpdateDialogAction = () => {
  const placeLayout = Layout.usePlacer();
  return (
    <Button.Button
      variant="outlined"
      size="small"
      onClick={() => placeLayout(INFO_LAYOUT)}
    >
      Update
    </Button.Button>
  );
};

interface SilenceActionProps {
  onClick: () => void;
  key: string;
}

const SilenceAction = ({ onClick }: SilenceActionProps) => {
  const handleClick = () => {
    silenceUpdateNotifications();
    onClick();
  };
  return (
    <Button.Button variant="text" onClick={handleClick}>
      <Icon.Snooze />
    </Button.Button>
  );
};
