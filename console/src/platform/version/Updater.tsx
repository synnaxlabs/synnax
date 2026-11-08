// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { useAsyncEffect, useCombinedStateAndRef } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { id, TimeSpan } from "@synnaxlabs/x";

import { Notifications } from "@/platform/notifications";
import { find, useTarget } from "@/platform/version/check";
import { useInfoModal } from "@/platform/version/useInfoModal";

const STATUS_KEY_PREFIX = "versionUpdate";
const CHECK_INTERVAL = TimeSpan.seconds(30);

/**
 * Checks for an update now and every 30 seconds until one is found, and adds a status
 * when it is. A change of the Core, its version, or the grant starts the checks again.
 * @returns true when an update is available.
 */
export const useCheckForUpdates = (): boolean => {
  const addStatus = Status.useAdder();
  const target = useTarget();
  const [available, setAvailable, availableRef] =
    useCombinedStateAndRef<boolean>(false);

  /** Never rejects: a check the network cannot serve is logged and retried later. */
  const checkForUpdates = async (signal: AbortSignal) => {
    if (availableRef.current) return;
    try {
      const found = await find(target);
      if (found == null) return;
      await found.update.close();
      if (signal.aborted) return;
      setAvailable(true);
      addStatus({
        key: `${STATUS_KEY_PREFIX}-${id.create()}`,
        variant: "info",
        message: `Update available`,
      });
    } catch (err) {
      console.error(err);
    }
  };

  useAsyncEffect(
    async (signal) => {
      setAvailable(false);
      await checkForUpdates(signal);
      if (signal.aborted) return;
      const i = setInterval(
        () => void checkForUpdates(signal),
        CHECK_INTERVAL.milliseconds,
      );
      return () => clearInterval(i);
    },
    [target?.endpoint, target?.version, target?.latestGranted],
  );

  return available;
};

export const Notification: Notifications.Notification = ({ status, silence }) => (
  <Status.Notification
    status={status}
    silence={silence}
    actions={[<OpenUpdateDialogAction key="update" />]}
  />
);
Notification.match = Notifications.matchPrefix(STATUS_KEY_PREFIX);

const OpenUpdateDialogAction = () => {
  const openInfo = useInfoModal();
  return (
    <Button.Button variant="outlined" size="small" onClick={() => openInfo()}>
      Update
    </Button.Button>
  );
};
