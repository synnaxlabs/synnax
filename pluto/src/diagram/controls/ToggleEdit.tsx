// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x/location";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";

import { type ReactElement } from "react";

import { useContext } from "@/diagram/Context";

export interface ToggleEditProps extends Omit<
  Button.ToggleProps,
  "value" | "onChange" | "children"
> {}

export const ToggleEdit = ({ onClick, ...rest }: ToggleEditProps): ReactElement => {
  const { editable, onEditableChange } = useContext();
  return (
    <Button.Toggle
      tooltipLocation={location.BOTTOM_LEFT}
      size="small"
      tooltip={`${editable ? "Disable" : "Enable"} editing`}
      {...rest}
      onChange={() => onEditableChange(!editable)}
      value={editable}
    >
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
    </Button.Toggle>
  );
};
