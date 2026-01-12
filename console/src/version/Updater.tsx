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
import { useState } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { type Notifications } from "@/notifications";
import { Runtime } from "@/runtime";
import { INFO_LAYOUT } from "@/version/Info";
import { useSelectUpdateNotificationsSilenced } from "@/version/selectors";
import { silenceUpdateNotifications } from "@/version/slice";

const STATUS_KEY_PREFIX = "versionUpdate";

export const useCheckForUpdates = (): boolean => {
  const addStatus = Status.useAdder();
  const isSilenced = useSelectUpdateNotificationsSilenced();
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
  const dispatch = useDispatch();
  const handleClick = () => {
    dispatch(silenceUpdateNotifications());
    onClick();
  };
  return (
    <Button.Button variant="text" onClick={handleClick}>
      <Icon.Snooze />
    </Button.Button>
  );
};
