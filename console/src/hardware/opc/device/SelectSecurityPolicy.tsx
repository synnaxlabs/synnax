// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { type SecurityPolicy } from "@/hardware/opc/device/types";

interface SecurityPolicyInfo {
  key: SecurityPolicy;
  name: string;
}

const SECURITY_POLICIES: SecurityPolicyInfo[] = [
  { key: "None", name: "None" },
  { key: "Basic128Rsa15", name: "Basic 128-bit (SHA-1)" },
  { key: "Basic256", name: "Basic 256-bit (SHA-1)" },
  { key: "Basic256Sha256", name: "Basic 256-bit (SHA-256)" },
  { key: "Aes128_Sha256_RsaOaep", name: "AES 128-bit (SHA-256, RSA-OAEP)" },
  { key: "Aes256_Sha256_RsaPss", name: "AES 256-bit (SHA-256, RSA-PSS)" },
];

export interface SelectSecurityPolicyProps
  extends Omit<
    Select.ButtonProps<SecurityPolicy, SecurityPolicyInfo>,
    "data" | "entryRenderKey"
  > {}

export const SelectSecurityPolicy = (
  props: SelectSecurityPolicyProps,
): ReactElement => (
  <Select.Button<SecurityPolicy, SecurityPolicyInfo>
    data={SECURITY_POLICIES}
    entryRenderKey="name"
    {...props}
  />
);
