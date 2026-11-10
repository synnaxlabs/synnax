// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/task/ArcItem.css";

import { arc, type task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { Access, Arc } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";

export interface ArcItemProps extends List.ItemProps<task.Key> {
  arcKey: arc.Key;
  onRename: (params: { key: arc.Key; name: string }) => void;
  onEdit: (key: arc.Key) => void;
}

/** A row for the task of an Arc, which shows and controls the Arc. */
export const ArcItem = ({
  arcKey,
  onRename,
  onEdit,
  ...rest
}: ArcItemProps): ReactElement => {
  const { data: arcItem } = Arc.useResult({ key: arcKey });
  const hasUpdatePermission = Access.useUpdateGranted(arc.resourceID(arcKey));
  const handleRename = (name: string) => onRename({ key: arcKey, name });
  const handleDoubleClick = () => onEdit(arcKey);
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const {
    running,
    onStartStop,
    taskStatus: status,
  } = Arc.useTaskControls(arcKey, arcItem?.name ?? "", {
    afterSuccess: captureDeploy,
  });
  const drifted = Arc.useDrifted({ arcKey });
  const isLoading = status.variant === "loading";
  return (
    <Select.Item
      {...rest}
      onDoubleClick={handleDoubleClick}
      justify="between"
      align="center"
    >
      <Flex.Box y gap="small" grow className={CSS.BE("arc", "metadata")}>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={status.variant}
            className={CSS.BE("arc-list-item", "status")}
          />
          <Flex.Box x align="center">
            <Icon.Arc />
            <Text.MaybeEditable
              id={`text-${arcKey}`}
              value={arcItem?.name ?? ""}
              onChange={hasUpdatePermission ? handleRename : undefined}
              allowDoubleClick={false}
              overflow="fade"
              weight={500}
            />
            {drifted && (
              <Tooltip.Dialog>
                <Text.Text level="small">Configuration changed since deploy</Text.Text>
                <Text.Text level="small" status="warning">
                  <Icon.Refresh />
                </Text.Text>
              </Tooltip.Dialog>
            )}
          </Flex.Box>
        </Flex.Box>
        <Text.Text level="tiny" color={8}>
          Arc automation task
        </Text.Text>
      </Flex.Box>
      {hasUpdatePermission && (
        <Button.Button
          variant="outlined"
          size="small"
          status={isLoading ? "loading" : running ? "error" : undefined}
          onClick={onStartStop}
          onDoubleClick={stopPropagation}
          tooltip={`${running ? "Stop" : "Start"} ${arcItem?.name ?? ""}`}
        >
          {running ? <Icon.Stop /> : <Icon.Play />}
        </Button.Button>
      )}
    </Select.Item>
  );
};
