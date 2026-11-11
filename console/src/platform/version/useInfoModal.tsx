// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/version/Info.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Logo } from "@synnaxlabs/lyra/logo";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Size } from "@synnaxlabs/x";
import { relaunch } from "@tauri-apps/plugin-process";
import { type Update } from "@tauri-apps/plugin-updater";
import { useState } from "react";

import { CSS } from "@/platform/css";
import { License } from "@/platform/license";
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
  start: (update: Update) => void;
}

const useDownload = (): UseDownloadReturn => {
  const [download, setDownload] = useState<Download>(ZERO_DOWNLOAD);
  const addStatus = Status.useAdder();
  const install = useInstall();
  const start = (update: Update): void =>
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

const Info = () => {
  const version = Session.Version.use();
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
    <Status.Summary level="h4" weight={350} variant="loading" gap="medium">
      Checking for updates
    </Status.Summary>
  );
  if (download.variant === "error" && download.status != null)
    updateContent = <Status.Summary level="h4" status={download.status} />;
  else if (download.variant === "loading")
    if (progressPercent === 100)
      updateContent = (
        <Status.Summary level="h4" variant="loading" gap="medium">
          Update downloaded. Restarting
        </Status.Summary>
      );
    else
      updateContent = (
        <Flex.Box y gap="medium">
          <Status.Summary variant="loading" level="h4" gap="medium">
            Downloading update
          </Status.Summary>
          <Flex.Box
            className={CSS.BE("version-info", "download")}
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
    updateContent = <Status.Summary level="h4" status={check.status} />;
  else if (check.variant === "success")
    if (check.available != null) {
      const found = check.available;
      updateContent = (
        <>
          <Status.Summary level="h4" variant="success">
            Version {found.update.version} available
          </Status.Summary>
          <Button.Button variant="filled" onClick={() => void startAfterConfirm(found)}>
            Update and restart
          </Button.Button>
        </>
      );
    } else
      updateContent = (
        <Status.Summary level="h4" variant="success">
          Up to date
        </Status.Summary>
      );

  return (
    <Modal.Frame className={CSS.B("version-info")}>
      <Modal.Header icon={<Icon.Info />}>About.Version</Modal.Header>
      <Flex.Box
        className={CSS.BE("version-info", "content")}
        align="center"
        y
        gap="large"
      >
        <Flex.Box y gap="small" justify="center" align="center">
          <a href="https://synnaxlabs.com" target="_blank" rel="noreferrer">
            <Logo.Logo variant="title" className={CSS.BE("version-info", "logo")} />
          </a>
          <Text.Text level="h3" weight={350}>
            Console v{version}
          </Text.Text>
        </Flex.Box>
        {updateContent}
        <License.Details />
        <Text.Text
          className={CSS.BE("version-info", "footer-note")}
          level="small"
          color={10}
          weight={350}
        >
          © 2022-2026 Synnax Labs, Inc. All rights reserved
        </Text.Text>
      </Flex.Box>
    </Modal.Frame>
  );
};

export const useInfoModal = Modals.create(Info);
