// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Flux, Progress, Status, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { relaunch } from "@tauri-apps/plugin-process";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { z } from "zod";

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

const { useRetrieve: useRetrieveUpdateAvailable } = Flux.createRetrieve<
  {},
  Update | null,
  {},
  true
>({
  name: "Version",
  allowDisconnected: true,
  retrieve: async () => {
    if (Runtime.ENGINE !== "tauri") return null;
    await new Promise((resolve) => setTimeout(resolve, 500));
    return await check();
  },
});

export const statusDetailsSchema = z.object({
  total: Size.z,
  progress: Size.z,
});

interface StatusDetails extends z.infer<typeof statusDetailsSchema> {}

const { useUpdate } = Flux.createUpdate<
  Update,
  {},
  Update,
  typeof statusDetailsSchema,
  true
>({
  name: "Console",
  verbs: Flux.UPDATE_VERBS,
  allowDisconnected: true,
  initialStatusDetails: {
    total: Size.bytes(0),
    progress: Size.bytes(0),
  },
  update: async ({ data: update, setStatus }) => {
    await update.downloadAndInstall((prog) => {
      const updateStatus = (v: (p: StatusDetails) => StatusDetails) =>
        setStatus((p) => {
          if (p.variant === "error") return p;
          return {
            ...p,
            variant: "loading",
            details: v(p.details),
          };
        });
      switch (prog.event) {
        case "Started":
          updateStatus((p) => ({
            ...p,
            total: Size.bytes(prog.data.contentLength ?? 0),
          }));
          break;
        case "Progress":
          updateStatus((p) => ({
            ...p,
            progress: p.progress.add(Size.bytes(prog.data.chunkLength)),
          }));
          break;
        case "Finished":
          updateStatus((p) => ({
            ...p,
            variant: "success",
            details: { ...p, progress: p.total },
          }));
          break;
      }
    });
    if (Runtime.ENGINE === "tauri") await relaunch();
    return update;
  },
});

export const Info: Layout.Renderer = () => {
  const version = useSelectVersion();
  const availableQuery = useRetrieveUpdateAvailable({});
  const updateQuery = useUpdate();
  let totalSize: Size = Size.bytes(0);
  let amountDownloaded: Size = Size.bytes(0);
  if (updateQuery.status.variant !== "error") {
    totalSize = updateQuery.status.details.total;
    amountDownloaded = updateQuery.status.details.progress;
  }

  const progressPercent = (amountDownloaded.valueOf() / totalSize.valueOf()) * 100;

  let updateContent = (
    <Status.Summary level="h4" weight={350} variant="loading" gap="medium">
      Checking for updates
    </Status.Summary>
  );
  if (availableQuery.variant === "error")
    updateContent = <Status.Summary level="h4" status={availableQuery.status} />;
  else if (updateQuery.variant === "error")
    updateContent = <Status.Summary level="h4" status={updateQuery.status} />;
  else if (updateQuery.variant === "loading")
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
              {Math.ceil(amountDownloaded.megabytes)} / {Math.ceil(totalSize.megabytes)}{" "}
              MB
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
      );
  else if (availableQuery.variant == "success")
    if (availableQuery.data != null) {
      const update = availableQuery.data;
      updateContent = (
        <>
          <Status.Summary level="h4" variant="success">
            Version {update.version} available
          </Status.Summary>
          <Button.Button variant="filled" onClick={() => updateQuery.update(update)}>
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
