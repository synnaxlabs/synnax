// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/Configure.css";

import { actions, device, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Device, Flux } from "@synnaxlabs/pluto";
import { deep, type record, strings, verbs } from "@synnaxlabs/x";
import { type ComponentType, type ReactElement } from "react";
import { z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { type Hardware } from "@/platform/device/integration";
import {
  type ConfigureParams,
  type ConfigureProps,
  identifierZ,
  nameZ,
} from "@/platform/device/types";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = deep.path<z.infer<ConfigurablePropertiesZ>>();

interface InternalProps extends Omit<ConfigureProps, "deviceKey"> {
  icon: Icon.ReactElement;
  device: device.Device;
  initialProperties: record.Unknown;
}

const configurablePropertiesZ = z.object({ name: nameZ, identifier: identifierZ });
type ConfigurablePropertiesZ = typeof configurablePropertiesZ;

interface SaveParams {
  prev: device.Device;
  next: device.Device;
}

const { useUpdate: useSave } = Flux.createUpdate<SaveParams>({
  name: "device",
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { prev, next } = data;
    const edits = actions.diffFields(device.deviceZ(), prev, next, [
      "name",
      "configured",
      "properties",
    ]);
    if (edits.length > 0) await client.devices.dispatch(prev.key, edits);
    return data;
  },
});

const Internal = ({
  device,
  device: { name },
  close,
  icon,
  initialProperties,
  onCreateReadTask,
  onBack,
}: InternalProps) => {
  const methods = Form.use<ConfigurablePropertiesZ>({
    values: { name, identifier: "" },
    schema: configurablePropertiesZ,
  });
  const currentName = Form.useFieldValue(path.name, { ctx: methods });
  const { capture } = Analytics.use();
  const { updateAsync, variant } = useSave();
  const suggestions = strings.createShortIdentifiers(currentName);
  const save = async (createTask: boolean) => {
    if (!methods.validate()) return;
    const saved = await updateAsync({
      prev: device,
      next: {
        ...device,
        configured: true,
        name: methods.get(path.name).value,
        properties: {
          ...deep.copy(initialProperties),
          ...device.properties,
          identifier: methods.get(path.identifier).value,
        },
      },
    });
    if (!saved) return;
    capture("device_configured", { make: device.make });
    if (createTask) onCreateReadTask?.(device.key);
    close();
  };

  return (
    <Modal.Frame className={CSS.B("configure")}>
      <Modal.Header icon={icon}>{name ? [name] : "Device.Configure"}</Modal.Header>
      <Form.Form<typeof configurablePropertiesZ> {...methods}>
        <Modal.Body align="stretch" gap="large">
          <Form.TextField
            inputProps={{ autoFocus: true, level: "h2", variant: "text" }}
            label="Name"
            path={path.name}
          />
          <Flex.Box gap="small">
            <Form.TextField
              label="Identifier"
              inputProps={{ level: "h3", variant: "text", placeholder: "dev1" }}
              path={path.identifier}
            />
            <Text.Text level="small" color={9}>
              A short prefix for every channel on this device.
            </Text.Text>
            <Flex.Box x align="center">
              <Icon.Bolt />
              {suggestions.map((id) => (
                <Button.Button
                  key={id}
                  onClick={() => methods.set(path.identifier, id)}
                  size="small"
                  variant="text"
                >
                  {id}
                </Button.Button>
              ))}
            </Flex.Box>
          </Flex.Box>
        </Modal.Body>
      </Form.Form>
      <Modal.Footer>
        <Nav.Bar.Start gap="small">
          {onBack != null && (
            <Button.Button variant="text" onClick={onBack}>
              <Icon.Arrow.Left />
              Back
            </Button.Button>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End gap="small">
          {onCreateReadTask != null && (
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              onClick={() => void save(true)}
            >
              Save and create read task
            </Button.Button>
          )}
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={() => void save(false)}
            variant="filled"
            trigger={Triggers.SAVE}
            triggerIndicator
            type="submit"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export interface CreateConfigureArgs {
  icon: Icon.ReactElement;
  /** The properties a device starts with when the user configures it. */
  zeroProperties: record.Unknown;
}

/** Creates the form that names a found device and sets its identifier. */
export const createConfigure = ({
  icon,
  zeroProperties,
}: CreateConfigureArgs): ComponentType<ConfigureProps> => {
  const Configure = ({ deviceKey, ...rest }: ConfigureProps): ReactElement => {
    const device = Device.use({ key: deviceKey });
    return (
      <Internal
        device={device}
        icon={icon}
        initialProperties={zeroProperties}
        {...rest}
      />
    );
  };
  return Configure;
};

export interface ConfigureModalParams extends ConfigureParams {
  hardware: Hardware;
}

/** Opens the configure form of a device of the hardware integration. */
export const useConfigureModal = Modals.create<ConfigureModalParams>(
  ({ deviceKey, hardware: { Configure }, close }) => (
    <Configure deviceKey={deviceKey} close={close} />
  ),
);
