// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Progress, Status, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { relaunch } from "@tauri-apps/plugin-process";
import { check } from "@tauri-apps/plugin-updater";
import { useState } from "react";

import { type Layout } from "@/layout";
import { Runtime } from "@/runtime";
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
      if (Runtime.ENGINE !== "tauri") return null;
      await new Promise((resolve) => setTimeout(resolve, 500));
      return await check();
    },
  });

  const [updateSize, setUpdateSize] = useState<Size>(Size.bytes(1));
  const [amountDownloaded, setAmountDownloaded] = useState(Size.bytes(0));
  const progressPercent = (amountDownloaded.valueOf() / updateSize.valueOf()) * 100;

  const handleError = Status.useErrorHandler();

  const updateMutation = useMutation({
    onError: (e) => handleError(e, "Failed to update Synnax Console"),
    mutationFn: async () => {
      if (!updateQuery.isSuccess) return;
      const update = updateQuery.data;
      if (update == null) return;
      await update.downloadAndInstall((progress) => {
        switch (progress.event) {
          case "Started":
            setUpdateSize(Size.bytes(progress.data.contentLength ?? 0));
            setAmountDownloaded(Size.bytes(0));
            break;
          case "Progress":
            setAmountDownloaded((prev) =>
              prev.add(Size.bytes(progress.data.chunkLength)),
            );
            break;
          case "Finished":
            setAmountDownloaded(updateSize);
            break;
        }
      });
      if (Runtime.ENGINE === "tauri") await relaunch();
    },
  });

  let updateContent = (
    <Status.Summary level="h4" weight={350} variant="loading" gap="medium">
      Checking for updates
    </Status.Summary>
  );
  if (updateMutation.isPending)
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
              {Math.ceil(amountDownloaded.megabytes)} /{" "}
              {Math.ceil(updateSize.megabytes)} MB
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      );
  else if (updateQuery.isSuccess)
    if (updateQuery.data != null) {
      const version = updateQuery.data.version;
      updateContent = (
        <>
          <Status.Summary level="h4" variant="success">
            Version {version} available
          </Status.Summary>
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
        <Status.Summary level="h4" variant="success">
          Up to date
        </Status.Summary>
      );
  else if (updateQuery.isError)
    updateContent = (
      <Status.Summary level="h4" variant="error">
        Error checking for update: {updateQuery.error.message}
      </Status.Summary>
    );
  else if (updateMutation.isError)
    updateContent = (
      <Status.Summary level="h4" variant="error">
        Error updating: {updateMutation.error.message}
      </Status.Summary>
    );

  return (
    <Flex.Box align="center" y gap="large" style={{ paddingTop: "6rem" }}>
      <Flex.Box y gap="small" justify="center" align="center">
        <a href="https://synnaxlabs.com" target="_blank" rel="noreferrer">
          <Logo variant="title" style={{ height: "10rem" }} />
        </a>
        <Text.Text level="h3" weight={350}>
          Console v{version}
        </Text.Text>
      </Flex.Box>
      {updateContent}
      <Text.Text
        level="small"
        color={10}
        weight={350}
        style={{ position: "absolute", bottom: "3rem" }}
      >
        © 2022-2025 Synnax Labs, Inc. All rights reserved
      </Text.Text>
    </Flex.Box>
  );
};
