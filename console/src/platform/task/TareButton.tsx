// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Channel } from "@synnaxlabs/pluto";

import { useIsSnapshot } from "@/platform/task/Form";
import { useIsRunning } from "@/platform/task/useStatus";
import { useTare } from "@/platform/task/useTare";

export interface TareButtonProps {
  /** The Synnax channel to tare, or 0 when the task has not created it. */
  channel: channel.Key;
  /** The task skips the channel. */
  disabled?: boolean;
}

/**
 * Zeroes a float channel at its present value while the task runs. Renders nothing for
 * any other channel or for a snapshot.
 */
export const TareButton = ({ channel, disabled = false }: TareButtonProps) => {
  const { data: dataType } = Channel.useResultDataType(
    channel === 0 ? null : { key: channel },
  );
  const isSnapshot = useIsSnapshot();
  const isRunning = useIsRunning();
  const tare = useTare();
  if (isSnapshot || dataType?.isFloat !== true) return null;
  return (
    <Button.Button
      variant="outlined"
      disabled={disabled || !isRunning}
      onClick={() => tare([channel])}
      size="small"
      tooltip="Tare"
    >
      <Icon.Tare />
    </Button.Button>
  );
};
