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
import { Logo } from "@synnaxlabs/media";
import {
  Button,
  Flex,
  Icon,
  Progress,
  Status,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { relaunch } from "@tauri-apps/plugin-process";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { useEffect, useRef, useState } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

type UpdateCheck =
  | { variant: "loading" }
  | { variant: "error"; status: status.Status }
  | { variant: "success"; update: Update | null };

const useUpdateCheck = (): UpdateCheck => {
  const [result, setResult] = useState<UpdateCheck>({ variant: "loading" });
  useAsyncEffect(async (signal) => {
    try {
      let update: Update | null = null;
      if (Session.Runtime.ENGINE === "tauri") {
        await new Promise((resolve) => setTimeout(resolve, 500));
        update = await check();
      }
      if (!signal.aborted) setResult({ variant: "success", update });
    } catch (error) {
      if (signal.aborted) return;
      setResult({
        variant: "error",
        status: status.fromException(error, "Failed to check for updates"),
      });
    }
  }, []);
  return result;
};

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
  const start = (update: Update): void =>
    void (async () => {
      setDownload({ ...ZERO_DOWNLOAD, variant: "loading" });
      try {
        await update.downloadAndInstall((prog) => {
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

export interface InfoModalParams {
  /** Starts the download and install as soon as the check finds an update. */
  autoUpdate?: boolean;
}

export const useInfoModal = Modals.create<InfoModalParams>(({ autoUpdate = false }) => {
  const version = Session.Version.use();
  const available = useUpdateCheck();
  const { download, start } = useDownload();
  const autoStarted = useRef(false);
  useEffect(() => {
    if (!autoUpdate || autoStarted.current || available.variant !== "success") return;
    if (available.update == null) return;
    autoStarted.current = true;
    start(available.update);
  }, [autoUpdate, available, start]);
  const progressPercent =
    (download.progress.valueOf() / download.total.valueOf()) * 100;

  let updateContent = (
    <Status.Summary level="h4" weight={350} variant="loading" gap="medium">
      Checking for updates
    </Status.Summary>
  );
  if (available.variant === "error")
    updateContent = <Status.Summary level="h4" status={available.status} />;
  else if (download.variant === "error" && download.status != null)
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
          <Flex.Box x gap="medium" align="center" justify="center">
            <Progress.Progress value={progressPercent} />
            <Text.Text color={10} overflow="ellipsis">
              {Math.ceil(download.progress.megabytes)} /{" "}
              {Math.ceil(download.total.megabytes)} MB
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      );
  else if (available.variant === "success")
    if (available.update != null) {
      const { update } = available;
      updateContent = (
        <>
          <Status.Summary level="h4" variant="success">
            Version {update.version} available
          </Status.Summary>
          <Button.Button variant="filled" onClick={() => start(update)}>
            Update & Restart
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
    <Modals.Frame className={CSS.B("version-info")}>
      <Modals.Header icon={<Icon.Info />}>About.Version</Modals.Header>
      <Flex.Box
        className={CSS.BE("version-info", "content")}
        align="center"
        y
        gap="large"
      >
        <Flex.Box y gap="small" justify="center" align="center">
          <a href="https://synnaxlabs.com" target="_blank" rel="noreferrer">
            <Logo variant="title" className={CSS.BE("version-info", "logo")} />
          </a>
          <Text.Text level="h3" weight={350}>
            Console v{version}
          </Text.Text>
        </Flex.Box>
        {updateContent}
        <Text.Text
          className={CSS.BE("version-info", "footer-note")}
          level="small"
          color={10}
          weight={350}
        >
          © 2022-2026 Synnax Labs, Inc. All rights reserved
        </Text.Text>
      </Flex.Box>
    </Modals.Frame>
  );
});
