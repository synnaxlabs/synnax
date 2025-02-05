// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import {
  AES128_SHA256_RSAOAEP_SECURITY_POLICY,
  AES256_SHA256_RSAPSS_SECURITY_POLICY,
  BASIC128_RSA15_SECURITY_POLICY,
  BASIC256_SECURITY_POLICY,
  BASIC256_SHA256_SECURITY_POLICY,
  NO_SECURITY_POLICY,
  type SecurityPolicy,
} from "@/hardware/opc/device/types";

interface SecurityPolicyInfo extends KeyedNamed<SecurityPolicy> {}

const SECURITY_POLICIES: SecurityPolicyInfo[] = [
  { key: NO_SECURITY_POLICY, name: "None" },
  { key: BASIC128_RSA15_SECURITY_POLICY, name: "Basic 128-bit RSA" },
  { key: BASIC256_SECURITY_POLICY, name: "Basic 256-bit" },
  { key: BASIC256_SHA256_SECURITY_POLICY, name: "Basic 256-bit with SHA-256" },
  { key: AES128_SHA256_RSAOAEP_SECURITY_POLICY, name: "AES 128-bit with SHA-256" },
  { key: AES256_SHA256_RSAPSS_SECURITY_POLICY, name: "AES 256-bit with SHA-256" },
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
