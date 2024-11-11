// Copyright 2024 Synnax Labs, Inc.
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

export const infoLayout: Layout.State = {
  type: "versionInfo",
  key: "versionInfo",
  windowKey: "versionInfo",
  name: "Version Info",
  location: "window",
  window: {
    resizable: false,
    navTop: true,
    size: { width: 500, height: 325 },
  },
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
    mutationKey: ["version.update"],
    mutationFn: async () => {
      if (!updateQuery.isFetched || updateQuery.data?.available !== true) return;
      const update = updateQuery.data;
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

  let updateContent: JSX.Element = (
    <Status.Text level="h4" weight={350} variant="loading" size="medium">
      Checking for updates
    </Status.Text>
  );
  if (updateMutation.isPending) 
    if (progressPercent === 100) 
      updateContent = (
        <Status.Text level="h4" variant="loading" size="medium">
          Update downloaded. Restarting
        </Status.Text>
      );
     else 
      updateContent = (
        <Align.Space direction="y" size="medium">
          <Status.Text variant="loading" level="h4" size="medium">
            Downloading update
          </Status.Text>
          <Align.Space direction="x" size="medium" align="center" justify="center">
            <Progress.Progress value={progressPercent} />
            <Text.Text level="p" shade={6} noWrap>
              {Math.ceil(amountDownloaded.megabytes)} /{" "}
              {Math.ceil(updateSize.megabytes)} MB
            </Text.Text>
          </Align.Space>
        </Align.Space>
      );
    
   else if (updateQuery.isFetched) 
    if (updateQuery.data?.available) {
      const version = updateQuery.data.version;
      updateContent = (
        <>
          <Status.Text level="h4" variant="success">
            Version {version} available
          </Status.Text>
          <Button.Button
            variant="outlined"
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
    <Align.Space
      align="center"
      direction="y"
      size="large"
      style={{ paddingTop: "6rem" }}
    >
      <Align.Space direction="y" size="small" justify="center" align="center">
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
        shade={6}
        weight={350}
        style={{ position: "absolute", bottom: "2rem" }}
      >
        © 2022-2024 Synnax Labs, Inc. All rights reserved
      </Text.Text>
    </Align.Space>
  );
};
