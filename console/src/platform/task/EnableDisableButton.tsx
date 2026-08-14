// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Form, Status } from "@synnaxlabs/pluto";

import { useIsSnapshot } from "@/platform/task/Form";
import { type Polarity } from "@/platform/task/types";

export interface EnableDisableButtonProps extends Omit<
  Button.ToggleProps,
  "onChange" | "value" | "children"
> {
  path: string;
  polarity?: Polarity;
}

export const EnableDisableButton = ({
  path,
  polarity = "enabled",
  ...rest
}: EnableDisableButtonProps) => {
  const isSnapshot = useIsSnapshot();
  const { set } = Form.useContext();
  const raw = Form.useFieldValue<boolean>(path, { optional: true });
  if (raw == null) return null;
  const value = polarity === "enabled" ? raw : !raw;
  return (
    <Button.Toggle
      disabled={isSnapshot}
      onChange={(v) => set(path, polarity === "enabled" ? v : !v)}
      size="small"
      tooltip={isSnapshot ? undefined : `${value ? "Disable" : "Enable"}`}
      value={value}
      {...rest}
    >
      <Status.Indicator />
    </Button.Toggle>
  );
};
