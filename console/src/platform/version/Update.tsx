// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/version/Update.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { type Available } from "@/platform/version/check";
import { useCheck, useDownload } from "@/platform/version/Updater";

export interface UpdateProps {
  /** The text level of the update status. Defaults to h4. */
  level?: text.Level;
}

/**
 * Shows whether a Console update is available, and installs it with a restart. Asks
 * first before a downgrade or a breaking update. Renders a fragment, so the parent sets
 * the direction.
 */
export const Update = ({ level = "h4" }: UpdateProps): ReactElement => {
  const check = useCheck();
  const { download, start } = useDownload();
  const confirm = Modals.useConfirm();
  const confirmed = async (found: Available): Promise<boolean> => {
    switch (found.variant) {
      case "upgrade":
        return true;
      case "downgrade":
        return (
          (await confirm({
            message: `Downgrade Console ${found.update.currentVersion} to ${found.update.version}`,
            description: "Downgrade only to match a Core that runs an older version.",
            confirm: { label: "Downgrade", variant: "warning" },
            title: "Downgrade Console",
          })) === true
        );
      case "breaking":
        return (
          (await confirm({
            message: `Console ${found.update.version} cannot work with Core ${found.coreVersion}`,
            description:
              "The Console cannot open the workspace of this Core until you update the Core to the same minor version.",
            confirm: { label: "Update", variant: "warning" },
            title: "Update Console",
          })) === true
        );
    }
  };
  const startAfterConfirm = async (found: Available): Promise<void> => {
    if (await confirmed(found)) start(found.update);
  };
  const available = check.variant === "success" ? check.available : null;
  const updateButton = available != null && (
    <Button.Button variant="filled" onClick={() => void startAfterConfirm(available)}>
      Update and restart
    </Button.Button>
  );
  const total = download.total.valueOf();
  const progressPercent = total === 0 ? 0 : (download.progress.valueOf() / total) * 100;

  let updateContent = (
    <Status.Summary level={level} weight={350} variant="loading" gap="medium">
      Checking for updates
    </Status.Summary>
  );
  if (download.variant === "error" && download.status != null)
    updateContent = (
      <>
        <Status.Summary level={level} status={download.status} />
        {updateButton}
      </>
    );
  else if (download.variant === "loading")
    if (progressPercent === 100)
      updateContent = (
        <Status.Summary level={level} variant="loading" gap="medium">
          Update downloaded. Restarting
        </Status.Summary>
      );
    else
      updateContent = (
        <Flex.Box y gap="medium">
          <Status.Summary variant="loading" level={level} gap="medium">
            Downloading update
          </Status.Summary>
          <Flex.Box
            className={CSS.BE("version-update", "download")}
            x
            gap="medium"
            align="center"
          >
            <Progress.Progress value={progressPercent} />
            <Text.Text color={10} overflow="nowrap" shrink={0}>
              {Math.ceil(download.progress.megabytes)} /{" "}
              {Math.ceil(download.total.megabytes)} MB
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      );
  else if (check.variant === "error" || check.variant === "blocked")
    updateContent = <Status.Summary level={level} status={check.status} />;
  else if (check.variant === "success")
    if (available != null)
      updateContent = (
        <>
          <Status.Summary level={level} variant="success">
            Version {available.update.version} available
          </Status.Summary>
          {updateButton}
        </>
      );
    else
      updateContent = (
        <Status.Summary level={level} variant="success">
          Up to date
        </Status.Summary>
      );

  return updateContent;
};
