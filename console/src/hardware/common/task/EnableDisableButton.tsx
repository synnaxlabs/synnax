// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Status, Text } from "@synnaxlabs/pluto";

export interface EnableDisableButtonProps {
  value: boolean;
  onChange: (v: boolean) => void;
  isDisabled?: boolean;
  isSnapshot: boolean;
}

export const EnableDisableButton = ({
  value,
  onChange,
  isDisabled = false,
  isSnapshot,
}: EnableDisableButtonProps) => (
  <Button.ToggleIcon
    checkedVariant={isSnapshot ? "preview" : undefined}
    uncheckedVariant={isSnapshot ? "preview" : "outlined"}
    disabled={isDisabled}
    value={value}
    size="small"
    stopPropagation
    onChange={onChange}
    tooltip={
      isSnapshot ? undefined : (
        <Text.Text level="small" style={{ maxWidth: 300 }}>
          {value ? "Disable" : "Enable"} data acquisition
        </Text.Text>
      )
    }
  >
    <Status.Circle variant={value ? "success" : "disabled"} />
  </Button.ToggleIcon>
);
