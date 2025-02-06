// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Form as PForm, Status, Text } from "@synnaxlabs/pluto";

export interface EnableDisableButtonProps {
  path: string;
  isSnapshot: boolean;
}

export const EnableDisableButton = ({ path, isSnapshot }: EnableDisableButtonProps) => {
  const { get, set } = PForm.useContext();
  const { value } = get<boolean>(path);
  return (
    <Button.ToggleIcon
      disabled={isSnapshot}
      onChange={(v) => set(path, v)}
      size="small"
      stopPropagation
      tooltip={
        isSnapshot ? undefined : (
          <Text.Text level="small">
            {value ? "Disable" : "Enable"} data acquisition
          </Text.Text>
        )
      }
      value={value}
    >
      <Status.Circle />
    </Button.ToggleIcon>
  );
};
