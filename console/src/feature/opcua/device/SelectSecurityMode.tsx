// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { record } from "@synnaxlabs/x";

import {
  NO_SECURITY_MODE,
  type SecurityMode,
  SIGN_AND_ENCRYPT_SECURITY_MODE,
  SIGN_SECURITY_MODE,
} from "@/feature/opcua/device/types";

/** The names of the security modes. */
export const SECURITY_MODE_NAMES: Record<SecurityMode, string> = {
  [NO_SECURITY_MODE]: "None",
  [SIGN_SECURITY_MODE]: "Sign",
  [SIGN_AND_ENCRYPT_SECURITY_MODE]: "Sign and encrypt",
};

export interface SelectSecurityModeProps extends Select.ButtonsProps<SecurityMode> {}

export const SelectSecurityMode = (props: SelectSecurityModeProps) => (
  <Select.Buttons {...props}>
    {record.entries(SECURITY_MODE_NAMES).map(([mode, name]) => (
      <Select.Item key={mode} itemKey={mode}>
        {name}
      </Select.Item>
    ))}
  </Select.Buttons>
);
