// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Button, Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { id, TimeSpan } from "@synnaxlabs/x";
import { check } from "@tauri-apps/plugin-updater";
import { useState } from "react";
import { useDispatch } from "react-redux";

import { getFlags } from "@/getFlags";
import { Layout } from "@/layout";
import { type NotificationAdapter } from "@/notifications/Notifications";
import { infoLayout } from "@/version/Info";
import { useSelectUpdateNotificationsSilenced } from "@/version/selectors";
import { silenceUpdateNotifications } from "@/version/slice";

export const useCheckForUpdates = (): boolean => {
  const addStatus = Status.useAggregator();
  const isSilenced = useSelectUpdateNotificationsSilenced();
  const [available, setAvailable] = useState(false);

  const checkForUpdates = async (addNotifications: boolean) => {
    const update = await check();
    if (getFlags().dev) return;
    if (update?.available !== true || available) return;
    setAvailable(true);
    if (addNotifications)
      addStatus({
        key: `versionUpdate-${id.id()}`,
        variant: "info",
        message: `Update available`,
      });
  };

  useAsyncEffect(async () => {
    await checkForUpdates(!isSilenced);
    const i = setInterval(
      () => checkForUpdates(!isSilenced),
      TimeSpan.seconds(30).milliseconds,
    );
    return () => clearInterval(i);
  }, [isSilenced]);

  return available;
};

export const notificationAdapter: NotificationAdapter = (status) => {
  if (!status.key.startsWith("versionUpdate")) return null;
  return {
    ...status,
    actions: [<OpenUpdateDialogAction key="update" />, <SilenceAction key="silence" />],
  };
};

export const OpenUpdateDialogAction = () => {
  const placer = Layout.usePlacer();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => placer(infoLayout)}>
      Update
    </Button.Button>
  );
};

const SilenceAction = () => {
  const dispatch = useDispatch();
  return (
    <Button.Icon variant="text" onClick={() => dispatch(silenceUpdateNotifications())}>
      <Icon.Snooze />
    </Button.Icon>
  );
};
