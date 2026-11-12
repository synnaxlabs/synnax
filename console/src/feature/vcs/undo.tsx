// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { Notifications } from "@/platform/notifications";
import { Runtime } from "@/platform/runtime";
import { Vcs } from "@/platform/vcs";

const UNDO_PREFIX = "vcs-undo-";
const LINK_PREFIX = "vcs-link-";

const undoDetailsZ = z.object({
  inverse: vcs.inverseZ,
  /** The message of the status that reports the undo. */
  undone: z.string(),
});

const linkDetailsZ = z.object({ url: z.string(), label: z.string() });

export interface UndoToast {
  name: string;
  message: string;
  description?: string;
  inverse: vcs.Inverse;
  /** The message to report once the undo succeeds, such as "Restored changes". */
  undone: string;
}

/**
 * Returns a function that reports a change to the Core with an Undo action. Undo writes
 * the inverse back. A toast with nothing to undo has no Undo action.
 */
export const useUndoToast = (): ((toast: UndoToast) => void) => {
  const addStatus = Status.useAdder();
  return ({ name, message, description, inverse, undone }) => {
    const empty =
      inverse.files.length === 0 &&
      inverse.removed.length === 0 &&
      inverse.configuration === "";
    if (empty) {
      addStatus({ variant: "success", name, message, description });
      return;
    }
    addStatus<typeof undoDetailsZ>({
      key: `${UNDO_PREFIX}${id.create()}`,
      variant: "success",
      name,
      message,
      description,
      details: { inverse, undone },
    });
  };
};

export interface LinkToast {
  name: string;
  message: string;
  description?: string;
  url: string;
  label: string;
}

/** Returns a function that reports a result with a button that opens a link. */
export const useLinkToast = (): ((toast: LinkToast) => void) => {
  const addStatus = Status.useAdder();
  return ({ name, message, description, url, label }) =>
    addStatus<typeof linkDetailsZ>({
      key: `${LINK_PREFIX}${id.create()}`,
      variant: "success",
      name,
      message,
      description,
      details: { url, label },
    });
};

const UndoAction = ({
  status,
  silence,
}: Notifications.NotificationProps): ReactElement => {
  const client = Synnax.use();
  const refresh = Vcs.useRefresh();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const handleClick = () =>
    handleError(
      async () => {
        if (client == null) return;
        const { inverse, undone } = undoDetailsZ.parse(status.details);
        silence(status.key);
        await client.vcs.write(inverse);
        await refresh();
        const name = inverse.configuration === "" ? status.name : inverse.configuration;
        addStatus({ variant: "success", name, message: undone });
      },
      { message: "Failed to undo", name: status.name },
    );
  return (
    <Button.Button variant="outlined" size="small" onClick={handleClick}>
      Undo
    </Button.Button>
  );
};

const UndoNotification: Notifications.Notification = (props) => (
  <Status.Notification
    status={props.status}
    silence={props.silence}
    actions={[<UndoAction key="undo" {...props} />]}
  />
);
UndoNotification.match = Notifications.matchPrefix(UNDO_PREFIX);

const LinkAction = ({ status }: Notifications.NotificationProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  const { url, label } = linkDetailsZ.parse(status.details);
  const handleClick = () =>
    handleError(async () => await Runtime.openExternal(url), {
      message: "Failed to open link",
      name: status.name,
    });
  return (
    <Button.Button variant="outlined" size="small" onClick={handleClick}>
      {label}
    </Button.Button>
  );
};

const LinkNotification: Notifications.Notification = (props) => (
  <Status.Notification
    status={props.status}
    silence={props.silence}
    actions={[<LinkAction key="link" {...props} />]}
  />
);
LinkNotification.match = Notifications.matchPrefix(LINK_PREFIX);

export const NOTIFICATIONS: Notifications.Notification[] = [
  UndoNotification,
  LinkNotification,
];
