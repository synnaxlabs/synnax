// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Channel as PChannel } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { Channel } from "@/platform/channel";
import { ContextMenu as Base } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Errors } from "@/platform/errors";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";

const Internal = ({ keys, target }: Menu.ContextMenuMenuProps): ReactElement => {
  const channelKeys = keys.map(Number);
  const channels = PChannel.useMultiple({ where: { key: { in: channelKeys } } });
  const ids = channelKeys.map((key) => channel.resourceID(key));
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const openCalculated = Channel.useCalculatedModal();
  const copyLink = Core.useCopyLinkToClipboard();
  const confirm = Modals.useConfirmDelete({ type: "Channel" });
  const { update: del } = PChannel.useDelete();
  const handleError = Status.useErrorHandler();
  const single = channels.length === 1 ? channels[0] : undefined;
  return (
    <Base.Menu>
      {single?.isCalculated === true && hasUpdatePermission && (
        <Menu.Item
          itemKey="openCalculated"
          onClick={() => openCalculated({ channelKey: single.key })}
        >
          <Icon.Edit />
          Edit calculation
        </Menu.Item>
      )}
      <Menu.Divider />
      {single != null && hasUpdatePermission && !single.internal && (
        <Base.RenameItem
          onClick={() => Text.edit(List.itemNameID(single.key), target)}
        />
      )}
      <Menu.Divider />
      {single != null && (
        <Link.CopyContextMenuItem
          onClick={() => copyLink({ name: single.name, resourceID: single.resourceID })}
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && channels.length > 0 && (
        <Base.DeleteItem
          onClick={() =>
            handleError(async () => {
              if (await confirm(channels)) del(channelKeys);
            }, "Failed to delete channels")
          }
        />
      )}
      <Menu.Divider />
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};

const ListContextMenu = (props: Menu.ContextMenuMenuProps): ReactElement => (
  <Errors.SuspenseBoundary loading={null}>
    <Internal {...props} />
  </Errors.SuspenseBoundary>
);

/** The menu of the channels selected in the List layout of the explorer. */
export const contextMenu = Component.renderProp(ListContextMenu);
