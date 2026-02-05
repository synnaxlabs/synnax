// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon } from "@synnaxlabs/pluto";
import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface StartStopButtonProps extends Omit<Button.ButtonProps, "onClick"> {
  /** Whether the task is currently running */
  running: boolean;
  /** Click handler */
  onClick: () => void;
  /** Button status variant */
  statusVariant?: status.Variant;
}

export const StartStopButton = ({
  running,
  onClick,
  statusVariant,
  ...props
}: StartStopButtonProps): ReactElement => (
  <Button.Button
    status={statusVariant}
    onClick={onClick}
    size="medium"
    variant="filled"
    {...props}
  >
    {running ? <Icon.Pause /> : <Icon.Play />}
  </Button.Button>
);
