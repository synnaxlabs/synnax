// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/Badge.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Access } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import { Vcs } from "@/platform/vcs";
import { Session } from "@/session";

const Internal = (): ReactElement | null => {
  const { status } = Vcs.useStatus();
  const dispatch = Session.useDispatch();
  const canView = Access.useRetrieveGranted(Vcs.RESOURCE_ID);
  if (status == null || !canView) return null;
  const toggle = () => dispatch(Session.Nav.selectLeft({ key: Vcs.TOOLBAR_KEY }));
  if (!status.enabled)
    return (
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        tooltip="Start version control"
        tooltipLocation={location.BOTTOM_LEFT}
        aria-label="Start version control"
        onClick={toggle}
      >
        <Icon.SourceControl />
      </Button.Button>
    );
  const { configuration, changes, incoming, waiting } = status;
  const parts: string[] = [];
  if (changes.length > 0) parts.push(Vcs.count(changes.length, "change"));
  if (incoming.length > 0) parts.push(`${incoming.length} behind`);
  if (waiting > 0) parts.push(`${waiting} to push`);
  return (
    <Button.Button
      variant="outlined"
      size="medium"
      rounded="small"
      gap="small"
      className={CSS.B("vcs-badge")}
      tooltip="Version control"
      tooltipLocation={location.BOTTOM_LEFT}
      aria-label={`Version control: ${[configuration, ...parts].join(", ")}`}
      onClick={toggle}
    >
      <Icon.Branch />
      <Text.Text el="span" weight={450} overflow="fade">
        {configuration}
      </Text.Text>
      {parts.map((part) => (
        <Text.Text
          key={part}
          el="span"
          color={9}
          className={CSS.BE("vcs-badge", "part")}
        >
          {part}
        </Text.Text>
      ))}
    </Button.Button>
  );
};

/**
 * The version control badge in the top bar: the configuration, its uncommitted
 * changes, and how far it is from the remote. A click toggles the toolbar.
 */
export const Badge = (): ReactElement | null =>
  FLAGS.versionControl ? <Internal /> : null;
