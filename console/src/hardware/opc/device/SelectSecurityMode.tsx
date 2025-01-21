// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import {
  NO_SECURITY_MODE,
  type SecurityMode,
  SIGN_AND_ENCRYPT_SECURITY_MODE,
  SIGN_SECURITY_MODE,
} from "@/hardware/opc/device/types";

interface SecurityModeInfo extends KeyedNamed<SecurityMode> {}

const SECURITY_MODES: SecurityModeInfo[] = [
  { key: NO_SECURITY_MODE, name: "None" },
  { key: SIGN_SECURITY_MODE, name: "Sign" },
  { key: SIGN_AND_ENCRYPT_SECURITY_MODE, name: "Sign And Encrypt" },
];

export interface SelectSecurityModeProps
  extends Omit<
    Select.ButtonProps<SecurityMode, SecurityModeInfo>,
    "data" | "entryRenderKey"
  > {}

export const SelectSecurityMode = (props: SelectSecurityModeProps) => (
  <Select.Button<SecurityMode, SecurityModeInfo>
    data={SECURITY_MODES}
    entryRenderKey="name"
    {...props}
  />
);
