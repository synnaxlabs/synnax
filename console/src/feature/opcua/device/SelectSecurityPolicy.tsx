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
  AES128_SHA256_RSAOAEP_SECURITY_POLICY,
  AES256_SHA256_RSAPSS_SECURITY_POLICY,
  BASIC128_RSA15_SECURITY_POLICY,
  BASIC256_SECURITY_POLICY,
  BASIC256_SHA256_SECURITY_POLICY,
  type SecurityPolicy,
} from "@/feature/opcua/device/types";

/** The names of the security policies other than None. */
export const SECURITY_POLICY_NAMES: Record<Exclude<SecurityPolicy, "None">, string> = {
  [BASIC256_SHA256_SECURITY_POLICY]: "Basic 256-bit with SHA-256",
  [AES128_SHA256_RSAOAEP_SECURITY_POLICY]: "AES 128-bit with SHA-256",
  [AES256_SHA256_RSAPSS_SECURITY_POLICY]: "AES 256-bit with SHA-256",
  [BASIC256_SECURITY_POLICY]: "Basic 256-bit (deprecated)",
  [BASIC128_RSA15_SECURITY_POLICY]: "Basic 128-bit RSA (deprecated)",
};

export interface SelectSecurityPolicyProps extends Omit<
  Select.SingleSimpleProps<SecurityPolicy>,
  "resourceName" | "children"
> {}

/** Selects a security policy other than None. */
export const SelectSecurityPolicy = (props: SelectSecurityPolicyProps) => (
  <Select.Simple<SecurityPolicy> resourceName="security policy" {...props}>
    {record.entries(SECURITY_POLICY_NAMES).map(([policy, name]) => (
      <Select.Item key={policy} itemKey={policy}>
        {name}
      </Select.Item>
    ))}
  </Select.Simple>
);
