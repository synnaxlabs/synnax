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

import { useTare, useTareable } from "@/platform/task/useTare";

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
  const tareable = useTareable([channel]);
  const tare = useTare();
  if (tareable == null || tareable.floats.length === 0) return null;
  const { floats, running } = tareable;
  return (
    <Button.Button
      variant="outlined"
      disabled={disabled || !running}
      onClick={() => tare(floats)}
      size="small"
      tooltip="Tare"
    >
      <Icon.Tare />
    </Button.Button>
  );
};
