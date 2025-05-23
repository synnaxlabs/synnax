// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Align, Button, Progress, Status, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { relaunch } from "@tauri-apps/plugin-process";
import { check, type DownloadEvent } from "@tauri-apps/plugin-updater";
import { useState } from "react";

import { type Layout } from "@/layout";
import { useSelectVersion } from "@/version/selectors";

export const INFO_LAYOUT_TYPE = "versionInfo";

export const INFO_LAYOUT: Layout.BaseState = {
  type: INFO_LAYOUT_TYPE,
  key: INFO_LAYOUT_TYPE,
  name: "About.Version",
  icon: "Info",
  location: "modal",
  window: { resizable: false, navTop: true, size: { width: 500, height: 325 } },
  excludeFromWorkspace: true,
};

export const Info: Layout.Renderer = () => {
  const version = useSelectVersion();
  const updateQuery = useQuery({
    queryKey: ["version.update"],
    queryFn: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500));
      return await check();
    },
  });

  const [updateSize, setUpdateSize] = useState<Size>(Size.bytes(1));
  const [amountDownloaded, setAmountDownloaded] = useState(Size.bytes(0));
  const progressPercent = (amountDownloaded.valueOf() / updateSize.valueOf()) * 100;

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!updateQuery.isSuccess) return;
      const update = updateQuery.data;
      if (update == null) return;
      await update.downloadAndInstall((progress: DownloadEvent) => {
        if (progress.event === "Started")
          setUpdateSize(Size.bytes(progress.data.contentLength ?? 0));
        else if (progress.event === "Progress")
          setAmountDownloaded((prev) =>
            prev.add(Size.bytes(progress.data.chunkLength)),
          );
      });
      await new Promise((resolve) => setTimeout(resolve, 100));
      setAmountDownloaded(updateSize);
      await new Promise((resolve) => setTimeout(resolve, 750));
      await relaunch();
    },
  });

  let updateContent = (
    <Status.Text level="h4" weight={350} loading size="medium">
      Checking for updates
    </Status.Text>
  );
  if (updateMutation.isPending)
    if (progressPercent === 100)
      updateContent = (
        <Status.Text level="h4" loading size="medium">
          Update downloaded. Restarting
        </Status.Text>
      );
    else
      updateContent = (
        <Align.Space y size="medium">
          <Status.Text loading level="h4" size="medium">
            Downloading update
          </Status.Text>
          <Align.Space x size="medium" align="center" justify="center">
            <Progress.Progress value={progressPercent} />
            <Text.Text level="p" shade={10} noWrap>
              {Math.ceil(amountDownloaded.megabytes)} /{" "}
              {Math.ceil(updateSize.megabytes)} MB
            </Text.Text>
          </Align.Space>
        </Align.Space>
      );
  else if (updateQuery.isSuccess)
    if (updateQuery.data != null) {
      const version = updateQuery.data.version;
      updateContent = (
        <>
          <Status.Text level="h4" variant="success">
            Version {version} available
          </Status.Text>
          <Button.Button
            variant="filled"
            disabled={updateMutation.isPending}
            onClick={() => updateMutation.mutate()}
          >
            Update & Restart
          </Button.Button>
        </>
      );
    } else
      updateContent = (
        <Status.Text level="h4" variant="success">
          Up to date
        </Status.Text>
      );
  else if (updateQuery.isError)
    updateContent = (
      <Status.Text level="h4" variant="error">
        Error checking for update: {updateQuery.error.message}
      </Status.Text>
    );
  else if (updateMutation.isError)
    updateContent = (
      <Status.Text level="h4" variant="error">
        Error updating: {updateMutation.error.message}
      </Status.Text>
    );

  return (
    <Align.Space align="center" y size="large" style={{ paddingTop: "6rem" }}>
      <Align.Space y size="small" justify="center" align="center">
        <a href="https://synnaxlabs.com" target="_blank" rel="noreferrer">
          <Logo variant="title" style={{ height: "10rem" }} />
        </a>
        <Text.Text level="h3" weight={350}>
          Console v{version}
        </Text.Text>
      </Align.Space>
      {updateContent}
      <Text.Text
        level="small"
        shade={10}
        weight={350}
        style={{ position: "absolute", bottom: "3rem" }}
      >
        © 2022-2025 Synnax Labs, Inc. All rights reserved
      </Text.Text>
    </Align.Space>
  );
};
