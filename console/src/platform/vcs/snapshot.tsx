// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { FLAGS } from "@/flags";
import { usePolledStatus, useUpdateUI } from "@/platform/vcs/Provider";
import { count, RESOURCE_ID } from "@/platform/vcs/resource";

export interface SnapshotRange {
  key: string;
  name: string;
}

/**
 * Returns a function that commits the live configuration, uncommitted changes
 * included, to a range.
 */
export const useSnapshot = (): ((range: SnapshotRange) => void) => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const update = useUpdateUI();
  return ({ key, name }) =>
    handleError(
      async () => {
        if (client == null) return;
        const snap = await client.vcs.snapshot(key);
        update((prev) => ({ ...prev, snapshots: prev.snapshots + 1 }));
        const uncommitted =
          snap.uncommitted > 0
            ? ` with ${count(snap.uncommitted, "uncommitted change")}`
            : "";
        addStatus({
          variant: "success",
          name,
          message: "Snapshotted configuration",
          description: `Saved ${snap.configuration} at ${snap.commit.shortHash}${uncommitted}.`,
        });
      },
      { message: "Failed to snapshot configuration", name },
    );
};

/** Returns whether the user may snapshot the configuration now. */
export const useSnapshotAllowed = (): boolean => {
  const status = usePolledStatus();
  const granted = Access.useCreateGranted(RESOURCE_ID);
  return FLAGS.versionControl && granted && status?.enabled === true;
};

export interface SnapshotMenuItemProps {
  range: SnapshotRange;
  /** Names the range in the label, for a menu that is not about the range. */
  named?: boolean;
}

const Item = ({ range, named = false }: SnapshotMenuItemProps): ReactElement | null => {
  const snapshot = useSnapshot();
  const allowed = useSnapshotAllowed();
  if (!allowed) return null;
  return (
    <Menu.Item itemKey="vcsSnapshot" onClick={() => snapshot(range)}>
      <Icon.Snapshot />
      {named ? `Snapshot configuration to ${range.name}` : "Snapshot configuration"}
    </Menu.Item>
  );
};

/** A menu item that snapshots the configuration to a range. */
export const SnapshotMenuItem = (props: SnapshotMenuItemProps): ReactElement | null =>
  FLAGS.versionControl ? <Item {...props} /> : null;
