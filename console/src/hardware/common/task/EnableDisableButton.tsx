// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Status, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/css";

export interface EnableDisableButtonProps {
  value: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
  snapshot?: boolean;
}

export const EnableDisableButton = ({
  value,
  onChange,
  disabled,
  snapshot = false,
}: EnableDisableButtonProps) => (
  <Button.ToggleIcon
    checkedVariant={snapshot ? "preview" : undefined}
    uncheckedVariant={snapshot ? "preview" : "outlined"}
    className={CSS.B("enable-disable-button")}
    disabled={disabled}
    value={value}
    size="small"
    onClick={(e) => e.stopPropagation()}
    onChange={onChange}
    tooltip={
      snapshot ? undefined : (
        <Text.Text level="small" style={{ maxWidth: 300 }}>
          {value ? "Disable" : "Enable"} data acquisition
        </Text.Text>
      )
    }
  >
    <Status.Circle variant={value ? "success" : "disabled"} />
  </Button.ToggleIcon>
);
