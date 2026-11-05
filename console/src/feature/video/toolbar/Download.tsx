// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax, Video } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useTimeRange } from "@/feature/video/useTimeRange";
import { Runtime } from "@/platform/runtime";

/** Downloads the range that the view plays as an MP4 file. */
export const Download = (): ReactElement => {
  const name = Video.useName();
  const channel = Video.useChannel();
  const range = useTimeRange();
  const client = Synnax.use();
  const download = Runtime.useDownload();
  const handleError = Status.useErrorHandler();
  const handleClick =
    range == null || channel === 0
      ? undefined
      : () =>
          handleError(
            async () => {
              if (client == null) throw new DisconnectedError();
              const stream = await client.readMP4(range, channel);
              await download({ stream, name, extension: "mp4" });
            },
            { message: "Failed to download video", name },
          );
  return (
    <Button.Button
      tooltip={handleClick == null ? "Set a range to download" : "Download as MP4"}
      size="medium"
      variant="text"
      disabled={handleClick == null}
      onClick={handleClick}
    >
      <Icon.Download />
    </Button.Button>
  );
};
