// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { FLAGS } from "@/flags";
import { changesOf } from "@/platform/vcs/changes";
import { useFocusCommit, usePolledStatus } from "@/platform/vcs/Provider";
import { RESOURCE_ID } from "@/platform/vcs/resource";

export interface CommitMenuItemProps {
  resources: resource.ID[];
}

const Item = ({ resources }: CommitMenuItemProps): ReactElement | null => {
  const status = usePolledStatus();
  const focusCommit = useFocusCommit();
  const canCommit = Access.useUpdateGranted(RESOURCE_ID);
  if (status?.enabled !== true || !canCommit) return null;
  const changed = changesOf(status.changes, resources);
  if (changed.length === 0) return null;
  return (
    <Menu.Item
      itemKey="vcsCommit"
      onClick={() => focusCommit(changed.map(({ resource }) => resource))}
    >
      <Icon.Commit />
      Commit...
    </Menu.Item>
  );
};

/**
 * A context menu item that opens the version control changes with only the changes of
 * the given resources checked. Renders nothing when none has changes.
 */
export const CommitMenuItem = (props: CommitMenuItemProps): ReactElement | null =>
  FLAGS.versionControl ? <Item {...props} /> : null;

/** The part of a tree context menu's props {@link CommitContextMenuItem} reads. */
export interface CommitContextMenuItemProps {
  selection: { ids: resource.ID[] };
}

/** A {@link CommitMenuItem} for the selection of a tree context menu. */
export const CommitContextMenuItem = ({
  selection: { ids },
}: CommitContextMenuItemProps): ReactElement | null => (
  <CommitMenuItem resources={ids} />
);
