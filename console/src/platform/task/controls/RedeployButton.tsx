// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface RedeployButtonProps extends Omit<Button.ButtonProps, "onClick"> {
  /** Click handler */
  onClick: () => void;
  /** Button status variant */
  statusVariant?: status.Variant;
}

export const RedeployButton = ({
  onClick,
  statusVariant,
  ...props
}: RedeployButtonProps): ReactElement => (
  <Button.Button
    onClick={onClick}
    status={statusVariant}
    size="medium"
    tooltip="Deploy the latest configuration"
    variant="outlined"
    {...props}
  >
    Redeploy
  </Button.Button>
);
