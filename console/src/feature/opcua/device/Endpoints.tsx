// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  type rack,
  status,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { commandTimeout } from "@/feature/opcua/device/commandTimeout";
import { sendScanCommand } from "@/feature/opcua/device/scanCommand";
import { SECURITY_MODE_NAMES } from "@/feature/opcua/device/SelectSecurityMode";
import { SECURITY_POLICY_NAMES } from "@/feature/opcua/device/SelectSecurityPolicy";
import {
  type ConnectionConfig,
  NO_SECURITY_POLICY,
  type SecurityMode,
  type SecurityPolicy,
} from "@/feature/opcua/device/types";
import {
  type Endpoint,
  endpointsResponseZ,
  GET_ENDPOINTS_COMMAND_TYPE,
} from "@/feature/opcua/task/types";

const LOGIN_NAMES: Record<string, string> = {
  anonymous: "Anonymous",
  username: "Username",
  certificate: "Certificate",
  issued_token: "Issued token",
};

const isMode = (mode: string): mode is SecurityMode =>
  Object.hasOwn(SECURITY_MODE_NAMES, mode);

const isPolicy = (policy: string): policy is SecurityPolicy =>
  policy === NO_SECURITY_POLICY || Object.hasOwn(SECURITY_POLICY_NAMES, policy);

/** One security mode and policy that the server offers. */
interface Option {
  key: string;
  mode: string;
  policy: string;
  logins: string[];
  level: number;
}

const optionKey = (mode: string, policy: string): string => `${mode}/${policy}`;

/**
 * Merges the endpoints that share a mode and policy, since servers list each one for
 * every address they listen on, and sorts them most secure first.
 */
const toOptions = (endpoints: Endpoint[]): Option[] => {
  const options = new Map<string, Option>();
  for (const {
    securityMode,
    securityPolicy,
    userTokenTypes,
    securityLevel,
  } of endpoints) {
    const key = optionKey(securityMode, securityPolicy);
    const existing = options.get(key);
    if (existing == null) {
      options.set(key, {
        key,
        mode: securityMode,
        policy: securityPolicy,
        logins: [...userTokenTypes],
        level: securityLevel,
      });
      continue;
    }
    existing.level = Math.max(existing.level, securityLevel);
    for (const login of userTokenTypes)
      if (!existing.logins.includes(login)) existing.logins.push(login);
  }
  return [...options.values()].sort((a, b) => b.level - a.level);
};

const optionName = ({ mode, policy }: Option): string => {
  const modeName = isMode(mode) ? SECURITY_MODE_NAMES[mode] : mode;
  if (!isPolicy(policy)) return `${modeName}, ${policy} (not supported)`;
  if (policy === NO_SECURITY_POLICY) return modeName;
  return `${modeName}, ${SECURITY_POLICY_NAMES[policy]}`;
};

type Result =
  | { variant: "success"; options: Option[] }
  | { variant: "error"; status: status.Status };

/** Asks the Driver on the rack for the endpoints of the server at connection. */
const findEndpoints = async (
  client: Client | null,
  rack: rack.Key,
  connection: ConnectionConfig,
): Promise<Result> => {
  if (client == null) throw new DisconnectedError();
  const res = await sendScanCommand(client, rack, {
    type: GET_ENDPOINTS_COMMAND_TYPE,
    timeout: commandTimeout(connection),
    args: { connection },
  });
  if (res.variant === "error") return { variant: "error", status: res };
  const { endpoints } = endpointsResponseZ.parse(res.details?.data);
  return { variant: "success", options: toOptions(endpoints) };
};

type State = { rack: rack.Key; endpoint: string } & (Result | { variant: "loading" });

export interface EndpointsProps {
  rack: rack.Key;
  onSelect: (mode: SecurityMode, policy: SecurityPolicy) => void;
}

/**
 * Asks the server for its endpoints through the Driver on the rack, and sets the
 * security mode and policy to the endpoint the user selects.
 */
export const Endpoints = ({ rack, onSelect }: EndpointsProps): ReactElement => {
  const client = Synnax.use();
  const [state, setState] = useState<State | null>(null);
  const { get } = Form.useContext();
  const endpoint = Form.useFieldValue<string>("properties.connection.endpoint");
  const securityMode = Form.useFieldValue<string>("properties.connection.securityMode");
  const securityPolicy = Form.useFieldValue<string>(
    "properties.connection.securityPolicy",
  );
  const handleFind = async () => {
    setState({ rack, endpoint, variant: "loading" });
    const connection = get<ConnectionConfig>("properties.connection").value;
    let next: State;
    try {
      next = { rack, endpoint, ...(await findEndpoints(client, rack, connection)) };
    } catch (error) {
      next = {
        rack,
        endpoint,
        variant: "error",
        status: status.fromException(error, "Failed to get endpoints"),
      };
    }
    setState(next);
  };
  const current = state?.rack === rack && state.endpoint === endpoint ? state : null;
  const options = current?.variant === "success" ? current.options : null;
  const handleChange = (key: string | null) => {
    const option = options?.find((o) => o.key === key);
    if (option == null || !isMode(option.mode) || !isPolicy(option.policy)) return;
    onSelect(option.mode, option.policy);
  };
  return (
    <Flex.Box y gap="small">
      <Flex.Box x gap="small" align="center">
        {options != null && options.length > 0 && (
          <Select.Simple<string>
            resourceName="endpoint"
            allowNone
            value={optionKey(securityMode, securityPolicy)}
            onChange={handleChange}
          >
            {options.map((option) => {
              const name = optionName(option);
              return (
                <Select.Item
                  key={option.key}
                  itemKey={option.key}
                  textValue={name}
                  disabled={!isMode(option.mode) || !isPolicy(option.policy)}
                >
                  <Text.Text>{name}</Text.Text>
                  <Text.Text level="small" color={9}>
                    {option.logins
                      .map((login) => LOGIN_NAMES[login] ?? login)
                      .join(", ")}
                  </Text.Text>
                </Select.Item>
              );
            })}
          </Select.Simple>
        )}
        <Button.Button
          size="small"
          variant="text"
          disabled={rack === 0 || endpoint === ""}
          status={current?.variant === "loading" ? "loading" : undefined}
          onClick={() => void handleFind()}
        >
          <Icon.Search />
          Find endpoints
        </Button.Button>
      </Flex.Box>
      {current?.variant === "error" && <Status.Summary status={current.status} />}
      {options?.length === 0 && (
        <Input.HelpText>The server lists no endpoints.</Input.HelpText>
      )}
    </Flex.Box>
  );
};
