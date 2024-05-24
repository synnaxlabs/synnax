import { check } from "@tauri-apps/plugin-updater";
import { useEffect, useRef, useState } from "react";
import { TimeSpan } from "@synnaxlabs/x";
import { Button, Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { NotificationAdapter } from "@/notifications/Notifications";
import { Layout } from "@/layout";
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
