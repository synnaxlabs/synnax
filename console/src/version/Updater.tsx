// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Status,
  useAsyncEffect,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { id, TimeSpan } from "@synnaxlabs/x";
import { check } from "@tauri-apps/plugin-updater";

import { type Notifications } from "@/notifications";
import { Runtime } from "@/runtime";
import { useInfoModal } from "@/version/Info";

const STATUS_KEY_PREFIX = "versionUpdate";

export const useCheckForUpdates = (): boolean => {
  const addStatus = Status.useAdder();
  const [available, setAvailable, availableRef] =
    useCombinedStateAndRef<boolean>(false);

  const checkForUpdates = async () => {
    if (Runtime.ENGINE !== "tauri" || availableRef.current) return;
    const update = await check();
    if (update == null) return;
    setAvailable(true);
    addStatus({
      key: `${STATUS_KEY_PREFIX}-${id.create()}`,
      variant: "info",
      message: `Update available`,
    });
  };

  useAsyncEffect(async (signal) => {
    await checkForUpdates();
    if (signal.aborted) return;
    const i = setInterval(
      () => void checkForUpdates(),
      TimeSpan.seconds(30).milliseconds,
    );
    return () => clearInterval(i);
  }, []);

  return available;
};

export const notificationAdapter: Notifications.Adapter = (status) => {
  if (!status.key.startsWith(STATUS_KEY_PREFIX)) return null;
  return { ...status, actions: [<OpenUpdateDialogAction key="update" />] };
};

export const OpenUpdateDialogAction = () => {
  const openInfo = useInfoModal();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => openInfo()}>
      Update
    </Button.Button>
  );
};
