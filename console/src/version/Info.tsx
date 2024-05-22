import { Layout } from "@/layout";
import { useSelect } from "@/version/selectors";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Progress, Status, Text } from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { relaunch } from "@tauri-apps/plugin-process";
import { DownloadEvent, check, type Update } from "@tauri-apps/plugin-updater";
import { useState } from "react";

export const Info: Layout.Renderer = ({ layoutKey }) => {
  const version = useSelect();
  const updateQuery = useQuery({
    queryKey: ["version.update"],
    queryFn: async () => await check(),
  });

  const [progressPercent, setProgressPercent] = useState(0);

  const updateMutation = useMutation({
    mutationKey: ["version.update"],
    mutationFn: async () => {
      if (!updateQuery.isFetched || !updateQuery.data?.available) return;
      const update = updateQuery.data;
      let updateSize: number = 0;
      await update.downloadAndInstall((progress: DownloadEvent) => {
        if (progress.event === "Started") {
          updateSize = progress.data.contentLength ?? 0;
        } else if (progress.event === "Progress") {
          const percent = (progress.data.chunkLength / updateSize) * 100;
          setProgressPercent(percent);
        } else if (progress.event === "Finished") {
          setProgressPercent(1);
        }
      });
      await relaunch();
    },
  });

  return (
    <Align.Space direction="y">
      <Text.Text level="h2">Synnax Console {version}</Text.Text>
      {updateQuery.isPending && (
        <Text.WithIcon startIcon={<Icon.Loading />} level="h3">
          Checking for updates
        </Text.WithIcon>
      )}
      {updateQuery.isError && (
        <Status.Text variant="error">
          Error checking for update: {updateQuery.error.message}
        </Status.Text>
      )}
      {updateQuery.isFetched &&
        (updateQuery.data?.available ? (
          <>
            <Status.Text variant="info">Update available</Status.Text>
            <Button.Button
              variant="outlined"
              size="small"
              disabled={updateMutation.isPending}
              onClick={() => updateMutation.mutate()}
            >
              Update & Restart
            </Button.Button>
          </>
        ) : (
          <Status.Text variant="success">Up to date</Status.Text>
        ))}
      {updateMutation.isPending ||
        (updateMutation.isSuccess && (
          <Align.Space direction="y">
            <Status.Text variant="info">Downloading update</Status.Text>
            <Progress.Progress value={progressPercent} />
          </Align.Space>
        ))}
    </Align.Space>
  );
};
