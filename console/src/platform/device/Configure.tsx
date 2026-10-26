// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/Configure.css";

import { type device, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Device } from "@synnaxlabs/pluto";
import { deep, type record, strings } from "@synnaxlabs/x";
import { z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { type Hardware } from "@/platform/device/integration";
import { type ConfigureParams, identifierZ, nameZ } from "@/platform/device/types";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

interface InternalProps<
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
> {
  close: () => void;
  icon: Icon.ReactElement;
  device: device.Device<Properties, Make, Model>;
  initialProperties: z.infer<Properties>;
  /** Opens a new read task for the device. Adds a "Save and create read task" action. */
  onCreateTask?: (deviceKey: device.Key) => void;
  /** Returns to the view that opened the form. Adds a "Back" action. */
  onBack?: () => void;
}

const configurablePropertiesZ = z.object({ name: nameZ, identifier: identifierZ });
type ConfigurablePropertiesZ = typeof configurablePropertiesZ;

const Internal = <
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
>({
  device,
  device: { name },
  close,
  icon,
  initialProperties,
  onCreateTask,
  onBack,
}: InternalProps<Properties, Make, Model>) => {
  const methods = Form.use<ConfigurablePropertiesZ>({
    values: { name, identifier: "" },
    schema: configurablePropertiesZ,
  });
  const currentName = Form.useFieldValue<string, string, ConfigurablePropertiesZ>(
    "name",
    { ctx: methods },
  );
  const { capture } = Analytics.use();
  const { updateAsync, variant } = Device.useCreate();
  const suggestions = strings.createShortIdentifiers(currentName);
  const save = async (createTask: boolean) => {
    if (!methods.validate()) return;
    const saved = await updateAsync({
      ...device,
      configured: true,
      name: methods.get<string>("name").value,
      properties: {
        ...(deep.copy(initialProperties) as record.Unknown),
        ...(device.properties as record.Unknown),
        identifier: methods.get<string>("identifier").value,
      },
    });
    if (!saved) return;
    capture("device_configured", { make: device.make });
    if (createTask) onCreateTask?.(device.key);
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
            path="name"
          />
          <Flex.Box gap="small">
            <Form.TextField
              label="Identifier"
              inputProps={{ level: "h3", variant: "text", placeholder: "dev1" }}
              path="identifier"
            />
            <Text.Text level="small" color={9}>
              A short prefix for every channel on this device.
            </Text.Text>
            <Flex.Box x align="center">
              <Icon.Bolt />
              {suggestions.map((id) => (
                <Button.Button
                  key={id}
                  onClick={() => methods.set("identifier", id)}
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
      <Nav.Bar location="bottom" size={48} bordered>
        <Nav.Bar.Start gap="small">
          {onBack != null && (
            <Button.Button variant="text" onClick={onBack}>
              <Icon.Arrow.Left />
              Back
            </Button.Button>
          )}
          <Triggers.SaveHelpText action="Save" noBar />
        </Nav.Bar.Start>
        <Nav.Bar.End gap="small">
          {onCreateTask != null && (
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
            type="submit"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Modal.Frame>
  );
};

export interface ConfigureProps<
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
> extends Pick<
  InternalProps<Properties, Make, Model>,
  "close" | "icon" | "initialProperties" | "onCreateTask" | "onBack"
> {
  deviceKey: device.Key;
}

export const Configure = <
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
>({
  deviceKey,
  ...rest
}: ConfigureProps<Properties, Make, Model>) => {
  const device = Device.use({ key: deviceKey });
  return <Internal device={device} {...rest} />;
};

export interface ConfigureModalParams extends ConfigureParams {
  hardware: Hardware;
}

/** Opens {@link Configure} for a device of the hardware integration. */
export const useConfigureModal = Modals.create<ConfigureModalParams>(
  ({ deviceKey, hardware, close }) => (
    <Configure
      deviceKey={deviceKey}
      close={close}
      icon={hardware.icon}
      initialProperties={hardware.zeroProperties}
    />
  ),
);
