// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";

import {
  NO_SECURITY_MODE,
  type SecurityMode,
  SIGN_AND_ENCRYPT_SECURITY_MODE,
  SIGN_SECURITY_MODE,
} from "@/hardware/opc/device/types";

const DATA: SecurityMode[] = [
  NO_SECURITY_MODE,
  SIGN_SECURITY_MODE,
  SIGN_AND_ENCRYPT_SECURITY_MODE,
];

export interface SelectSecurityModeProps
  extends Omit<Select.ButtonsProps<SecurityMode>, "keys"> {}

export const SelectSecurityMode = (props: SelectSecurityModeProps) => (
  <Select.Buttons {...props} keys={DATA}>
    <Select.Button itemKey={NO_SECURITY_MODE}>None</Select.Button>
    <Select.Button itemKey={SIGN_SECURITY_MODE}>Sign</Select.Button>
    <Select.Button itemKey={SIGN_AND_ENCRYPT_SECURITY_MODE}>
      Sign And Encrypt
    </Select.Button>
  </Select.Buttons>
);
