// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/version/Update.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Size, type text } from "@synnaxlabs/x";
import { relaunch } from "@tauri-apps/plugin-process";
import { type Update as TauriUpdate } from "@tauri-apps/plugin-updater";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { type Available } from "@/platform/version/check";
import { useCheck, useInstall } from "@/platform/version/Updater";
import { Session } from "@/session";

interface Download {
  variant: "idle" | "loading" | "error";
  total: Size;
  progress: Size;
  status?: status.Status;
}

const ZERO_DOWNLOAD: Download = {
  variant: "idle",
  total: Size.bytes(0),
  progress: Size.bytes(0),
};

interface UseDownloadReturn {
  download: Download;
  start: (update: TauriUpdate) => void;
}

const useDownload = (): UseDownloadReturn => {
  const [download, setDownload] = useState<Download>(ZERO_DOWNLOAD);
  const addStatus = Status.useAdder();
  const install = useInstall();
  const start = (update: TauriUpdate): void =>
    void (async () => {
      setDownload({ ...ZERO_DOWNLOAD, variant: "loading" });
      try {
        await install(update, (prog) => {
          switch (prog.event) {
            case "Started":
              setDownload((p) => ({
                ...p,
                total: Size.bytes(prog.data.contentLength ?? 0),
              }));
              break;
            case "Progress":
              setDownload((p) => ({
                ...p,
                progress: p.progress.add(Size.bytes(prog.data.chunkLength)),
              }));
              break;
            case "Finished":
              setDownload((p) => ({ ...p, progress: p.total }));
              break;
          }
        });
        if (Session.Runtime.ENGINE === "tauri") await relaunch();
      } catch (error) {
        const st = status.fromException(error, "Failed to update Console");
        addStatus(st);
        setDownload((p) => ({ ...p, variant: "error", status: st }));
      }
    })();
  return { download, start };
};

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
  const total = download.total.valueOf();
  const progressPercent = total === 0 ? 0 : (download.progress.valueOf() / total) * 100;

  let updateContent = (
    <Status.Summary level={level} weight={350} variant="loading" gap="medium">
      Checking for updates
    </Status.Summary>
  );
  if (download.variant === "error" && download.status != null)
    updateContent = <Status.Summary level={level} status={download.status} />;
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
            <Text.Text color={10} overflow="nowrap">
              {Math.ceil(download.progress.megabytes)} /{" "}
              {Math.ceil(download.total.megabytes)} MB
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      );
  else if (check.variant === "error" || check.variant === "blocked")
    updateContent = <Status.Summary level={level} status={check.status} />;
  else if (check.variant === "success")
    if (check.available != null) {
      const found = check.available;
      updateContent = (
        <>
          <Status.Summary level={level} variant="success">
            Version {found.update.version} available
          </Status.Summary>
          <Button.Button variant="filled" onClick={() => void startAfterConfirm(found)}>
            Update and restart
          </Button.Button>
        </>
      );
    } else
      updateContent = (
        <Status.Summary level={level} variant="success">
          Up to date
        </Status.Summary>
      );

  return updateContent;
};
