// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Form, Status } from "@synnaxlabs/pluto";

import { useIsSnapshot } from "@/hardware/common/task/Form";

export interface EnableDisableButtonProps extends Omit<
  Button.ToggleProps,
  "onChange" | "value" | "children"
> {
  path: string;
}

export const EnableDisableButton = ({ path, ...rest }: EnableDisableButtonProps) => {
  const isSnapshot = useIsSnapshot();
  const { get, set } = Form.useContext();
  const fs = get<boolean>(path, { optional: true });
  if (fs == null) return null;
  const { value } = fs;
  return (
    <Button.Toggle
      disabled={isSnapshot}
      onChange={(v) => set(path, v)}
      size="small"
      tooltip={
        isSnapshot ? undefined : `${value ? "Disable" : "Enable"} data acquisition`
      }
      value={value}
      {...rest}
    >
      <Status.Indicator />
    </Button.Toggle>
  );
};
