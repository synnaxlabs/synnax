// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/bus/Connect.css";

import { device, type rack } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Access, Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { caseconv, type record } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { z } from "zod";

import { Analytics } from "@/platform/analytics";
import { Command } from "@/platform/command";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { type Task } from "@/platform/task";

export interface CreateDeviceParams<
  M extends string,
  P extends z.ZodType<record.Unknown>,
> {
  make: M;
  /** The integration's name in sentences, such as CAN. */
  name: string;
  /** Names one device in titles, such as CAN device. */
  noun: string;
  /** What the device picker says the integration does. */
  description: string;
  icon: Icon.ReactElement;
  properties: P;
  /** The integration a rack must run for the device to connect from it. */
  integration: Analytics.Properties<"device_connected">["integration"];
  /** Edits the device's properties, at paths under properties. */
  Properties: FC;
  /** @returns the model of a device with the given properties. */
  getModel: (properties: z.infer<P>) => string;
  /** @returns where a device with the given properties is reached. */
  getLocation: (properties: z.infer<P>) => string;
  /** The error shown on each text property that must not be empty. */
  required?: Partial<Record<keyof z.infer<P> & string, string>>;
}

/**
 * Creates the connect form, device select, commands, and device picker entry of a bus
 * integration. An integration supplies its property fields and how they name a
 * device's model and location. A scanned device opens in the same form to finish its
 * properties.
 */
export const createDevice = <M extends string, P extends z.ZodType<record.Unknown>>({
  make,
  name,
  noun,
  description,
  icon,
  properties,
  integration,
  Properties,
  getModel,
  getLocation,
  required = {},
}: CreateDeviceParams<M, P>) => {
  const SCHEMAS = {
    properties,
    make: z.literal(make),
    model: z.string(),
  } as const satisfies device.DeviceSchemas<P, z.ZodLiteral<M>, z.ZodString>;

  type Schema = PDevice.FormSchema<typeof SCHEMAS>;

  const INITIAL_VALUES: device.Device<P, z.ZodLiteral<M>> = {
    key: "",
    name: noun,
    make,
    model: "",
    location: "",
    properties: properties.parse({}),
    rack: 0,
    configured: true,
  };

  const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

  const beforeValidate = ({
    get,
    set,
    setStatus,
  }: Flux.BeforeValidateParams<PDevice.RetrieveQuery, Schema>): boolean => {
    const props = properties.parse(get("properties").value);
    let valid = true;
    for (const [field, message] of Object.entries(required)) {
      if (message == null || props[field] !== "") continue;
      const path = `properties.${field}`;
      setStatus(path, { key: path, variant: "error", message });
      valid = false;
    }
    if (!valid) return false;
    set("model", getModel(props));
    set("location", getLocation(props));
    set("configured", true);
    return true;
  };

  const NAME_INPUT_PROPS = { level: "h2", variant: "text", placeholder: noun } as const;

  const Connect = ({
    deviceKey,
    rack: defaultRack,
    close,
    onBack,
  }: PlatformDevice.ConnectProps) => {
    const { capture } = Analytics.use();
    const {
      form,
      save,
      status: stat,
      variant,
    } = useForm({
      query: deviceKey == null ? null : { key: deviceKey },
      initialValues: { ...INITIAL_VALUES, rack: defaultRack ?? 0 },
      beforeValidate,
      afterSave: ({ get }) => {
        if (deviceKey == null) capture("device_connected", { integration });
        close(get<device.Key>("key").value);
      },
    });
    const rackKey = Form.useFieldValue<rack.Key, rack.Key, Schema>("rack", {
      ctx: form,
    });
    const reason = PlatformDevice.useBlockedReason({
      integration,
      name,
      rack: rackKey,
    });
    return (
      <Modal.Frame className={CSS.B("bus-connect")}>
        <Modal.Header icon={icon}>{`${noun}.Connect`}</Modal.Header>
        <Modal.Body>
          <Form.Form<Schema> {...form}>
            <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
            <PlatformDevice.RackField integration={integration} />
            <Properties />
          </Form.Form>
        </Modal.Body>
        <PlatformDevice.ConnectFooter
          reason={reason}
          variant={variant}
          message={stat.description}
          onConnect={() => save()}
          onBack={onBack}
        />
      </Modal.Frame>
    );
  };

  const useConnectModal = PlatformDevice.createConnectModal(Connect, integration);

  const Select = () => {
    const openConnect = useConnectModal();
    return (
      <PlatformDevice.Select
        onConfigure={(deviceKey) => void openConnect({ deviceKey })}
        emptyContent={`No ${noun}s connected`}
        label={caseconv.capitalize(noun)}
        make={make}
        icon={icon}
        connect={{ title: noun, open: () => openConnect() }}
      />
    );
  };

  const COMMANDS: Command.Command[] = [
    Command.create({
      key: `${integration}_connect_device`,
      name: `Connect ${noun}`,
      icon,
      useOnSelect: useConnectModal,
      useVisible: () => Access.useCreateGranted(device.TYPE_ONTOLOGY_ID),
    }),
  ];

  const NETWORK: PlatformDevice.Network = {
    key: integration,
    name,
    title: noun,
    description,
    icon,
    Connect,
  };

  return {
    MAKE: make,
    useConnectModal,
    Select,
    COMMANDS,
    NETWORK,
  };
};

export interface CreateContextMenuItemsParams {
  integration: string;
  useCreateRead: Task.UseCreate;
  useCreateWrite: Task.UseCreate;
}

/** Creates a bus device's context menu: edit its connection and create its tasks. */
export const createContextMenuItems = ({
  integration,
  useCreateRead,
  useCreateWrite,
}: CreateContextMenuItemsParams) => {
  const items = [
    PlatformDevice.createTaskContextMenuItem({
      itemKey: `${integration}.readTask`,
      label: "Create read task",
      useCreate: useCreateRead,
    }),
    PlatformDevice.createTaskContextMenuItem({
      itemKey: `${integration}.writeTask`,
      label: "Create write task",
      useCreate: useCreateWrite,
    }),
  ];
  const ContextMenuItems = (
    props: PlatformDevice.ContextMenuItemsProps,
  ): ReactElement => (
    <>
      <PlatformDevice.EditConnectionMenuItem {...props} />
      <Menu.Divider />
      <PlatformDevice.TaskContextMenuItems {...props} items={items} />
    </>
  );
  return ContextMenuItems;
};
