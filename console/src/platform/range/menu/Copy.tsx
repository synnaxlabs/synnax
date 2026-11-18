// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Core } from "@/platform/core";
import { Session } from "@/session";

const iso = ({ timeRange: { start, end } }: ranger.Range): string =>
  end.equals(TimeStamp.MAX)
    ? start.toString("ISO")
    : `${start.toString("ISO")} - ${end.toString("ISO")}`;

// A range planned with no times holds TimeStamp.MAX as its start.
const isScheduled = ({ timeRange }: ranger.Range): boolean =>
  !timeRange.makeValid().start.equals(TimeStamp.MAX);

export interface CopyProps {
  ranges: ranger.Range[];
}

/** Copies the link, names, times, or code of the ranges. */
export const Copy = ({ ranges }: CopyProps): ReactElement => {
  const copyLink = Core.useCopyLinkToClipboard();
  const single = ranges.length === 1 ? ranges[0] : null;
  const names = ranges.map((r) => r.name).join("\n");
  const name = single?.name ?? "";
  return (
    <Menu.Sub itemKey="copy" textValue="Copy">
      <Menu.SubTrigger>
        <Icon.Copy />
        Copy
      </Menu.SubTrigger>
      <Menu.Dialog aria-label="Copy">
        <Menu.Menu gap="small">
          {single != null && !Session.Runtime.LINKS_DISABLED && (
            <Menu.Item
              itemKey="link"
              onClick={() =>
                copyLink({
                  name: single.name,
                  resourceID: ranger.resourceID(single.key),
                })
              }
            >
              <Icon.Link />
              Link
            </Menu.Item>
          )}
          <Menu.CopyItem
            itemKey="name"
            text={names}
            successMessage={{
              message: single == null ? "Copied names" : "Copied name",
              name,
            }}
          >
            <Icon.Rename />
            {single == null ? "Names" : "Name"}
          </Menu.CopyItem>
          {single != null && isScheduled(single) && (
            <Menu.CopyItem
              itemKey="time"
              text={iso(single)}
              successMessage={{ message: "Copied time range", name }}
            >
              <Icon.Range />
              Time range
            </Menu.CopyItem>
          )}
          {single != null && (
            <>
              <Menu.CopyItem
                itemKey="python"
                text={`rng = client.ranges.retrieve(key="${single.key}")`}
                successMessage={{ message: "Copied Python code", name }}
              >
                <Icon.Python />
                Python code
              </Menu.CopyItem>
              <Menu.CopyItem
                itemKey="typescript"
                text={`const rng = await client.ranges.retrieve("${single.key}");`}
                successMessage={{ message: "Copied TypeScript code", name }}
              >
                <Icon.TypeScript />
                TypeScript code
              </Menu.CopyItem>
            </>
          )}
        </Menu.Menu>
      </Menu.Dialog>
    </Menu.Sub>
  );
};
