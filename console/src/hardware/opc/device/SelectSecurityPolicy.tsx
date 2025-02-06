// Copyright 2025 Synnax Labs, Inc.
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
  { key: "Basic128Rsa15", name: "Basic128Rsa15" },
  { key: "Basic256", name: "Basic256" },
  { key: "Basic256Sha256", name: "Basic256Sha256" },
  { key: "Aes128_Sha256_RsaOaep", name: "Aes128_Sha256_RsaOaep" },
  { key: "Aes256_Sha256_RsaPss", name: "Aes256_Sha256_RsaPss" },
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
