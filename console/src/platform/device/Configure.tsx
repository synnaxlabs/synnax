// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/Configure.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, strings } from "@synnaxlabs/x";
import { type ComponentType, type ReactElement } from "react";
import { z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import {
  type Adapter,
  type Configurable,
  type SaveParams,
} from "@/platform/device/adapter";
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
  device: Configurable;
  make: string;
  save: (params: SaveParams) => Promise<boolean>;
  variant: status.Variant;
}

const configurablePropertiesZ = z.object({ name: nameZ, identifier: identifierZ });
type ConfigurablePropertiesZ = typeof configurablePropertiesZ;

const Internal = ({
  device,
  device: { name },
  make,
  save: saveDevice,
  variant,
  close,
  icon,
  onCreateReadTask,
  onBack,
}: InternalProps) => {
  const methods = Form.use<ConfigurablePropertiesZ>({
    values: { name, identifier: "" },
    schema: configurablePropertiesZ,
  });
  const currentName = Form.useFieldValue(path.name, { ctx: methods });
  const { capture } = Analytics.use();
  const suggestions = strings.createShortIdentifiers(currentName);
  const save = async (createTask: boolean) => {
    if (!methods.validate()) return;
    const saved = await saveDevice({
      key: device.key,
      name: methods.get(path.name).value,
      identifier: methods.get(path.identifier).value,
    });
    if (!saved) return;
    capture("device_configured", { make });
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
  adapter: Adapter;
}

/** Creates the form that names a found device and sets its identifier. */
export const createConfigure = ({
  icon,
  adapter,
}: CreateConfigureArgs): ComponentType<ConfigureProps> => {
  const { use: useConfigurable, useSave, make } = adapter;
  const Configure = ({ deviceKey, ...rest }: ConfigureProps): ReactElement => {
    const device = useConfigurable(deviceKey);
    const { updateAsync, variant } = useSave();
    return (
      <Internal
        device={device}
        make={make}
        save={updateAsync}
        variant={variant}
        icon={icon}
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
