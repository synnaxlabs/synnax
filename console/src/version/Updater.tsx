// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { check } from "@tauri-apps/plugin-updater";
import { useState } from "react";

import { Layout } from "@/layout";
import { NotificationAdapter } from "@/notifications/Notifications";
import { infoLayout } from "@/version/Info";

export const useCheckForUpdates = (): boolean => {
  const addStatus = Status.useAggregator();

  const [available, setAvailable] = useState(false);

  const checkForUpdates = async () => {
    const update = await check();
    if (update?.available !== true || available) return;
    setAvailable(true);
    addStatus({
      key: "versionUpdate",
      variant: "info",
      message: `Update available`,
    });
  };

  useAsyncEffect(async () => {
    await checkForUpdates();
    const i = setInterval(checkForUpdates, TimeSpan.seconds(30).milliseconds);
    return () => clearInterval(i);
  }, []);

  return available;
};

export const notificationAdapter: NotificationAdapter = (status) => {
  if (!status.key.startsWith("versionUpdate")) return null;
  return {
    ...status,
    actions: [<OpenUpdateDialogAction key="update" />],
  };
};

const OpenUpdateDialogAction = () => {
  const place = Layout.usePlacer();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => place(infoLayout)}>
      Update
    </Button.Button>
  );
};
