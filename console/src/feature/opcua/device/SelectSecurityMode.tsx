// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";

import {
  NO_SECURITY_MODE,
  type SecurityMode,
  SIGN_AND_ENCRYPT_SECURITY_MODE,
  SIGN_SECURITY_MODE,
} from "@/feature/opcua/device/types";

export interface SelectSecurityModeProps extends Select.ButtonsProps<SecurityMode> {}

export const SelectSecurityMode = (props: SelectSecurityModeProps) => (
  <Select.Buttons {...props}>
    <Select.Item itemKey={NO_SECURITY_MODE}>None</Select.Item>
    <Select.Item itemKey={SIGN_SECURITY_MODE}>Sign</Select.Item>
    <Select.Item itemKey={SIGN_AND_ENCRYPT_SECURITY_MODE}>Sign and encrypt</Select.Item>
  </Select.Buttons>
);
