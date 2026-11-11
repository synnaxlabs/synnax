// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/vcs/RangeSnapshots.css";

import { vcs } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import {
  useOpenCompare,
  useOpenPlan,
  usePolledStatus,
  useUI,
} from "@/platform/vcs/Provider";
import { RESOURCE_ID } from "@/platform/vcs/resource";
import { useCloneModal } from "@/platform/vcs/useCloneModal";

interface ItemProps {
  snapshot: vcs.Snapshot;
}

const SnapshotItem = ({ snapshot }: ItemProps): ReactElement => {
  const { commit, configuration, uncommitted } = snapshot;
  const short = commit.shortHash;
  const openCompare = useOpenCompare();
  const openPlan = useOpenPlan();
  const promptClone = useCloneModal();
  const handleError = Status.useErrorHandler();
  const canUpdate = Access.useUpdateGranted(RESOURCE_ID);
  const compare = () =>
    openCompare({
      title: `Snapshot ${short}`,
      from: commit.hash,
      fromLabel: short,
      to: vcs.WORKING,
      toLabel: "Now",
    });
  const load = (message: string) =>
    openPlan({
      target: { kind: "restore", commit: commit.hash },
      title: `Load snapshot ${short}`,
      after: commit.hash,
      afterLabel: short,
      message,
    });
  const cloneFrom = () =>
    handleError(
      async () => {
        const name = await promptClone({
          description:
            "The new configuration starts from the last commit. The snapshot then loads into it as uncommitted changes, for you to commit.",
        });
        if (name == null) return;
        load(`Start ${name} from snapshot ${short}`);
      },
      { message: "Failed to create configuration", name: configuration },
    );
  const handleChange = (key: string) => {
    if (key === "compare") compare();
    else if (key === "load") load(`Load snapshot ${short}`);
    else if (key === "clone") cloneFrom();
  };
  return (
    <List.Item
      itemKey={snapshot.tag}
      className={CSS.BE("vcs-range-snapshots", "item")}
      x
      align="center"
      justify="between"
      onClick={compare}
    >
      <Text.Text weight={450} gap="small" overflow="fade">
        <Icon.SourceControl />
        {configuration}
        <Text.Text el="span" level="small" variant="code" color={8}>
          {short}
        </Text.Text>
        {uncommitted > 0 && (
          <Text.Text el="span" level="small" color={8}>
            {uncommitted} uncommitted
          </Text.Text>
        )}
      </Text.Text>
      <Flex.Box x align="center" gap="small" shrink={false}>
        <Telem.Text.TimeSpanSince level="small" format="semantic" color={8}>
          {commit.time}
        </Telem.Text.TimeSpanSince>
        <Dialog.Frame location={location.BOTTOM_RIGHT}>
          <Dialog.Trigger
            hideCaret
            variant="text"
            size="small"
            textColor={9}
            full={false}
            onClick={(e) => e.stopPropagation()}
            aria-label={`Actions for snapshot ${short}`}
          >
            <Icon.KebabMenu />
          </Dialog.Trigger>
          <Dialog.Dialog background={1} pack={false} bordered>
            <ActionsMenu canUpdate={canUpdate} onChange={handleChange} />
          </Dialog.Dialog>
        </Dialog.Frame>
      </Flex.Box>
    </List.Item>
  );
};

interface ActionsMenuProps {
  canUpdate: boolean;
  onChange: (key: string) => void;
}

const ActionsMenu = ({ canUpdate, onChange }: ActionsMenuProps): ReactElement => {
  const { close } = Dialog.useContext();
  const handleChange = (key: string) => {
    close();
    onChange(key);
  };
  return (
    <Menu.Menu level="small" gap="small" onChange={handleChange}>
      <Menu.Item itemKey="compare">
        <Icon.Compare />
        Compare with now
      </Menu.Item>
      {canUpdate && (
        <>
          <Menu.Item itemKey="load">
            <Icon.Pull />
            Load this configuration
          </Menu.Item>
          <Menu.Item itemKey="clone">
            <Icon.Branch />
            New configuration from this snapshot
          </Menu.Item>
        </>
      )}
    </Menu.Menu>
  );
};

export interface RangeSnapshotsProps {
  rangeKey: string;
}

const Snapshots = ({ rangeKey }: RangeSnapshotsProps): ReactElement | null => {
  const client = Synnax.use();
  const status = usePolledStatus();
  const { snapshots: epoch } = useUI();
  const handleError = Status.useErrorHandler();
  const [snapshots, setSnapshots] = useState<vcs.Snapshot[]>([]);
  const enabled = status?.enabled === true;
  useEffect(() => {
    if (client == null || !enabled) return;
    let cancelled = false;
    handleError(async () => {
      const next = await client.vcs.snapshots(rangeKey);
      if (!cancelled) setSnapshots(next);
    }, "Failed to load configuration snapshots");
    return () => {
      cancelled = true;
    };
  }, [client, rangeKey, enabled, epoch]);
  if (!enabled || snapshots.length === 0) return null;
  return (
    <Flex.Box y empty className={CSS.B("vcs-range-snapshots")}>
      {snapshots.map((s) => (
        <SnapshotItem key={s.tag} snapshot={s} />
      ))}
    </Flex.Box>
  );
};

/** Lists the configuration snapshots of a range, newest first. */
export const RangeSnapshots = (props: RangeSnapshotsProps): ReactElement | null =>
  FLAGS.versionControl ? <Snapshots {...props} /> : null;
