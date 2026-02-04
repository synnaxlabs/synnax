// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Text, Triggers } from "@synnaxlabs/pluto";
import { type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export interface ConfigureButtonProps extends Omit<Button.ButtonProps, "onClick"> {
  /** Click handler */
  onClick: () => void;
  /** Whether to show keyboard trigger hint */
  showTrigger?: boolean;
  /** Button status variant */
  statusVariant?: status.Variant;
}

export const ConfigureButton = ({
  onClick,
  showTrigger = false,
  statusVariant,
  ...props
}: ConfigureButtonProps): ReactElement => (
  <Button.Button
    onClick={onClick}
    status={statusVariant}
    size="medium"
    tooltip={
      showTrigger ? (
        <Flex.Box x align="center" gap="small">
          <Triggers.Text level="small" trigger={CONFIGURE_TRIGGER} />
          <Text.Text level="small">To Configure</Text.Text>
        </Flex.Box>
      ) : undefined
    }
    trigger={showTrigger ? CONFIGURE_TRIGGER : undefined}
    variant="outlined"
    {...props}
  >
    Configure
  </Button.Button>
);
