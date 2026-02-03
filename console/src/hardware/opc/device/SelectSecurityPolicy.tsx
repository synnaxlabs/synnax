// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";

import {
  AES128_SHA256_RSAOAEP_SECURITY_POLICY,
  AES256_SHA256_RSAPSS_SECURITY_POLICY,
  BASIC128_RSA15_SECURITY_POLICY,
  BASIC256_SECURITY_POLICY,
  BASIC256_SHA256_SECURITY_POLICY,
  NO_SECURITY_POLICY,
  type SecurityPolicy,
} from "@/hardware/opc/device/types";

const DATA: SecurityPolicy[] = [
  NO_SECURITY_POLICY,
  BASIC128_RSA15_SECURITY_POLICY,
  BASIC256_SECURITY_POLICY,
  BASIC256_SHA256_SECURITY_POLICY,
  AES128_SHA256_RSAOAEP_SECURITY_POLICY,
  AES256_SHA256_RSAPSS_SECURITY_POLICY,
];

export interface SelectSecurityPolicyProps extends Omit<
  Select.ButtonsProps<SecurityPolicy>,
  "keys"
> {}

export const SelectSecurityPolicy = (props: SelectSecurityPolicyProps) => (
  <Select.Buttons {...props} keys={DATA}>
    <Select.Button itemKey={NO_SECURITY_POLICY}>None</Select.Button>
    <Select.Button itemKey={BASIC128_RSA15_SECURITY_POLICY}>
      Basic 128-bit RSA
    </Select.Button>
    <Select.Button itemKey={BASIC256_SECURITY_POLICY}>Basic 256-bit</Select.Button>
    <Select.Button itemKey={BASIC256_SHA256_SECURITY_POLICY}>
      Basic 256-bit with SHA-256
    </Select.Button>
    <Select.Button itemKey={AES128_SHA256_RSAOAEP_SECURITY_POLICY}>
      AES 128-bit with SHA-256
    </Select.Button>
    <Select.Button itemKey={AES256_SHA256_RSAPSS_SECURITY_POLICY}>
      AES 256-bit with SHA-256
    </Select.Button>
  </Select.Buttons>
);
