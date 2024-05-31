import { SecurityMode, SecurityPolicy } from "@/hardware/opc/device/types";
import { Select } from "@synnaxlabs/pluto";

interface SecurityPolicyInfo {
  key: SecurityPolicy;
  name: string;
}

const SECURITY_POLICIES: SecurityPolicyInfo[] = [
  {
    key: "None",
    name: "None",
  },
  {
    key: "Basic128Rsa15",
    name: "Basic128Rsa15",
  },
  {
    key: "Basic256",
    name: "Basic256",
  },
  {
    key: "Basic256Sha256",
    name: "Basic256Sha256",
  },
  {
    key: "Aes128_Sha256_RsaOaep",
    name: "Aes128_Sha256_RsaOaep",
  },
  {
    key: "Aes256_Sha256_RsaPss",
    name: "Aes256_Sha256_RsaPss",
  },
];

export interface SelectSecurityPolicyProps
  extends Omit<Select.ButtonProps<SecurityPolicy, SecurityPolicyInfo>, "data"> {}

export const SelectSecurityPolicy = (props: SelectSecurityPolicyProps) => (
  <Select.Button<SecurityPolicy, SecurityPolicyInfo>
    data={SECURITY_POLICIES}
    {...props}
  />
);

interface SecurityModeInfo {
  key: SecurityMode;
  name: string;
}

const SECURITY_MODES: SecurityModeInfo[] = [
  {
    key: "None",
    name: "None",
  },
  {
    key: "Sign",
    name: "Sign",
  },
  {
    key: "SignAndEncrypt",
    name: "Sign And Encrypt",
  },
];

export interface SelectSecurityModeProps
  extends Omit<Select.ButtonProps<SecurityMode, SecurityModeInfo>, "data"> {}

export const SelectSecurityMode = (props: SelectSecurityModeProps) => (
  <Select.Button<SecurityMode, SecurityModeInfo> data={SECURITY_MODES} {...props} />
);
