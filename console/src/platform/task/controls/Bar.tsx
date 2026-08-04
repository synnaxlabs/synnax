// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { type Flex } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useCallback, useState } from "react";

import { Actions } from "@/platform/task/controls/Actions";
import { Frame } from "@/platform/task/controls/Frame";
import { RedeployButton } from "@/platform/task/controls/RedeployButton";
import { StartStopButton } from "@/platform/task/controls/StartStopButton";
import { Status } from "@/platform/task/controls/Status";

export interface BarProps extends Flex.BoxProps {
  status: status.Status;
  running: boolean;
  /** Shows the redeploy button when the deployed config is stale. */
  drifted?: boolean;
  /** Hides the action row entirely. */
  snapshot?: boolean;
  /** Disables the deploy and start/stop actions. */
  disabled?: boolean;
  /** Shown when the status message is empty. */
  fallbackMessage?: string;
  /** Rendered inside the action row, before the deploy controls. */
  extraActions?: ReactNode;
  onDeploy: () => void;
  onStop: () => void;
}

/**
 * Presentational task controls bar: an expandable status next to the deploy,
 * redeploy, and start/stop actions. Play deploys; pause stops.
 */
export const Bar = ({
  status: stat,
  running,
  drifted = false,
  snapshot = false,
  disabled = false,
  fallbackMessage,
  extraActions,
  onDeploy,
  onStop,
  ...props
}: BarProps): ReactElement => {
  const [expanded, setExpanded] = useState(false);
  const handleToggle = useCallback(() => setExpanded((prev) => !prev), []);
  const handleContract = useCallback(() => setExpanded(false), []);
  const handleStartStop = useCallback(
    () => (running ? onStop() : onDeploy()),
    [running, onStop, onDeploy],
  );
  return (
    <Frame expanded={expanded} onContract={handleContract} {...props}>
      <Status
        status={stat}
        expanded={expanded}
        onToggle={handleToggle}
        fallbackMessage={fallbackMessage}
      />
      {!snapshot && (
        <Actions>
          {extraActions}
          {drifted && <RedeployButton onClick={onDeploy} disabled={disabled} />}
          <StartStopButton
            running={running}
            onClick={handleStartStop}
            disabled={disabled}
            statusVariant={status.keepVariants(stat.variant, "loading")}
          />
        </Actions>
      )}
    </Frame>
  );
};
