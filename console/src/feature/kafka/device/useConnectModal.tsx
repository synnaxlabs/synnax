// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/device/Connect.css";

import { type device, type rack, status, TimeSpan } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Device as PDevice, type Flux, Rack } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import {
  type Device,
  type SASL,
  type SASLType,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/kafka/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/feature/kafka/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { type Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(15);

const INITIAL_VALUES: Device = {
  key: "",
  name: "",
  make: "kafka",
  model: "cluster",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const brokers = get<string[]>("properties.brokers").value;
  set("location", brokers.filter((b) => b !== "").join(","));
};

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: get<rack.Key>("rack").value,
    includeStatus: true,
    schemas: SCAN_SCHEMAS,
  });
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { connection: get("properties").value },
  });
  if (state.variant === "error") throw new Error(state.message);
  // The scan just succeeded, so the device starts healthy. The scanner overwrites
  // this status when the cluster goes away.
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    message: "Device connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { notifyOnChange: false, markTouched: false });
  return true;
};

const BrokersField = () => (
  <Form.Field<string[]> path="properties.brokers" label="Brokers" required>
    {({ value, onChange }) => (
      <Flex.Box y gap="small">
        {value.map((broker, i) => (
          <Flex.Box x key={i} gap="small" align="center">
            <Flex.Box grow>
              <Input.Text
                value={broker}
                onChange={(next) => onChange(value.map((b, j) => (j === i ? next : b)))}
                placeholder="localhost:9092"
              />
            </Flex.Box>
            {value.length > 1 && (
              <Button.Button
                variant="text"
                size="small"
                tooltip="Remove broker"
                onClick={() => onChange(value.filter((_, j) => j !== i))}
              >
                <Icon.Close />
              </Button.Button>
            )}
          </Flex.Box>
        ))}
        <Flex.Box x>
          <Button.Button
            variant="text"
            size="small"
            onClick={() => onChange([...value, ""])}
          >
            <Icon.Add />
            Add broker
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    )}
  </Form.Field>
);

const SASL_ITEMS = (
  <>
    <Select.Item itemKey="none">None</Select.Item>
    <Select.Item itemKey="plain">PLAIN</Select.Item>
    <Select.Item itemKey="scram_sha_256">SCRAM-SHA-256</Select.Item>
    <Select.Item itemKey="scram_sha_512">SCRAM-SHA-512</Select.Item>
  </>
);

const SASL_TRIGGER_PROPS = { className: CSS.BE("kafka-connect", "sasl") };

/** Clears the CA certificate when TLS turns off. The Core rejects one without TLS. */
const handleTLS = (tls: boolean, { set }: Form.ContextValue) => {
  if (!tls) set("properties.caCertificate", "");
};

/** Takes a PEM certificate authority for brokers whose certificates the system roots
 * do not trust. Shown only with TLS on. */
const CAField = () => {
  const tls = Form.useFieldValue<boolean>("properties.tls");
  if (!tls) return null;
  return (
    <Form.TextField
      path="properties.caCertificate"
      label="CA certificate"
      inputProps={CA_INPUT_PROPS}
    />
  );
};

/** Selects the SASL mechanism and, for the ones that carry them, the credentials. */
const SASLFields = () => {
  const { set } = Form.useContext();
  const sasl = Form.useFieldValue<SASL>("properties.sasl");
  const handleMechanism = useCallback(
    (mechanism: SASLType) => {
      if (mechanism === "none") set("properties.sasl", { mechanism });
      else
        set("properties.sasl", {
          mechanism,
          username: "username" in sasl ? sasl.username : "",
          password: "password" in sasl ? sasl.password : "",
        });
    },
    [sasl, set],
  );
  return (
    <>
      <Flex.Box x justify="between">
        <Input.Item label="Authentication">
          <Select.Simple<SASLType>
            value={sasl.mechanism}
            onChange={handleMechanism}
            resourceName="mechanism"
            triggerProps={SASL_TRIGGER_PROPS}
          >
            {SASL_ITEMS}
          </Select.Simple>
        </Input.Item>
        <Form.SwitchField path="properties.tls" label="TLS" onChange={handleTLS} />
      </Flex.Box>
      {sasl.mechanism !== "none" && (
        <Flex.Box x justify="between">
          <Form.TextField
            grow
            path="properties.sasl.username"
            inputProps={USERNAME_INPUT_PROPS}
          />
          <Form.TextField
            grow
            path="properties.sasl.password"
            inputProps={PASSWORD_INPUT_PROPS}
          />
        </Flex.Box>
      )}
    </>
  );
};

export const useConnectModal = Modals.create<PlatformDevice.ConnectParams>(
  ({ deviceKey, close }) => {
    const { capture } = Analytics.use();
    const {
      form,
      save,
      status: stat,
      variant,
    } = useForm({
      query: deviceKey == null ? null : { key: deviceKey },
      beforeValidate,
      beforeSave,
      afterSave: useCallback(() => {
        if (deviceKey == null) capture("device_connected", { integration: "kafka" });
        close();
      }, [capture, close, deviceKey]),
    });

    return (
      <Modal.Frame className={CSS.B("kafka-connect")}>
        <Modal.Header icon={<Icon.Logo.Kafka />}>Cluster.Connect</Modal.Header>
        <Flex.Box className={CSS.B("content")} grow size="small">
          <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
            <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
            <Form.Field<rack.Key> path="rack" label="Connect from" required>
              {selectRackRenderProp}
            </Form.Field>
            <BrokersField />
            <Divider.Divider x padded="bottom" />
            <SASLFields />
            <CAField />
          </Form.Form>
        </Flex.Box>
        <Modal.Footer>
          <Nav.Bar.Start gap="small">
            {variant == "success" ? (
              <Triggers.SaveHelpText action="Connect" noBar />
            ) : (
              <Status.Summary variant={variant} message={stat.description} />
            )}
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              onClick={() => save()}
              variant="filled"
              trigger={Triggers.SAVE}
            >
              Connect
            </Button.Button>
          </Nav.Bar.End>
        </Modal.Footer>
      </Modal.Frame>
    );
  },
);

const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration: "kafka" };

const selectRackRenderProp = Component.renderProp(
  (props: Pick<Rack.SelectSingleProps, "value" | "onChange">) => (
    <Rack.SelectSingle {...props} initialQuery={INITIAL_RACK_QUERY} />
  ),
);

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Kafka cluster",
  autoFocus: true,
} as const;

const CA_INPUT_PROPS = {
  placeholder: "-----BEGIN CERTIFICATE-----",
  area: true,
  className: CSS.BE("kafka-connect", "ca"),
} as const;

const USERNAME_INPUT_PROPS = { placeholder: "username" } as const;

const PASSWORD_INPUT_PROPS = { type: "password" } as const;
