// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Target, useActions } from "@/feature/vcs/diff/Provider";
import { Toolbar } from "@/platform/toolbar";
import { Vcs } from "@/platform/vcs";

const KIND_LABELS: Record<vcs.ChangeKind, string> = {
  added: "Added",
  modified: "Modified",
  deleted: "Deleted",
};

const KIND_STATUSES: Record<vcs.ChangeKind, Status.Variant> = {
  added: "success",
  modified: "warning",
  deleted: "error",
};

export interface HeaderProps {
  target: Target;
  /** kind is null while the files load, and when the revs hold the same files. */
  kind: vcs.ChangeKind | null;
  json: boolean;
  onJsonChange: (json: boolean) => void;
}

/** The header of the diff tab: what changed, between which revs, and the actions. */
export const Header = ({
  target,
  kind,
  json,
  onJsonChange,
}: HeaderProps): ReactElement => {
  const Actions = useActions();
  const TypeIcon = Vcs.typeIcon(target.type);
  return (
    <Toolbar.Header>
      <Flex.Box x align="center" gap="medium" shrink grow>
        <Toolbar.Title>
          <TypeIcon />
          {target.name}
        </Toolbar.Title>
        {kind != null && (
          <Badge.Badge status={KIND_STATUSES[kind]}>{KIND_LABELS[kind]}</Badge.Badge>
        )}
      </Flex.Box>
      <Flex.Box x align="center" gap="medium" shrink={false}>
        <Text.Text level="small" color={9} gap="small">
          {target.beforeLabel}
          <Icon.Arrow.Right />
          {target.afterLabel}
        </Text.Text>
        <Toolbar.Actions>
          <Toolbar.ToggleAction
            value={json}
            onChange={onJsonChange}
            tooltip={json ? "Show the changes" : "Show the raw files"}
            tooltipLocation={location.BOTTOM_LEFT}
          >
            <Icon.JSON />
            JSON
          </Toolbar.ToggleAction>
          {Actions != null && kind != null && <Actions target={target} />}
        </Toolbar.Actions>
      </Flex.Box>
    </Toolbar.Header>
  );
};
