// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/bus/Connect.css";

import { device, type rack, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Device as PDevice, type Flux, Rack } from "@synnaxlabs/pluto";
import { caseconv, type record } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";
import { z } from "zod";

import { Command } from "@/platform/command";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Empty } from "@/platform/empty";
import { Modals } from "@/platform/modals";
import { type Task } from "@/platform/task";
import { type Tree } from "@/platform/tree";
import { Triggers } from "@/platform/triggers";

type FormParams = Flux.BeforeValidateParams<
  PDevice.RetrieveQuery,
  typeof PDevice.formSchema
>;

export interface CreateDeviceParams<
  M extends string,
  P extends z.ZodType<record.Unknown>,
> {
  make: M;
  /** Names one device in titles, such as CAN device. */
  noun: string;
  icon: Icon.ReactElement;
  properties: P;
  /** The integration a rack must run for the device to connect from it. */
  integration: string;
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
 * Creates the connect modal, device select, and commands of a bus integration. An
 * integration supplies its property fields and how they name a device's model and
 * location. A scanned device opens in the same modal to finish its properties.
 */
export const createDevice = <M extends string, P extends z.ZodType<record.Unknown>>({
  make,
  noun,
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

  const useForm = PDevice.createForm(SCHEMAS);

  const INITIAL_VALUES: device.Device = {
    key: "",
    name: noun,
    make,
    model: "",
    location: "",
    properties: properties.parse({}),
    rack: 0,
    configured: true,
  };

  const beforeValidate = ({ get, set, setStatus }: FormParams): boolean => {
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

  const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration };

  const renderRack = Component.renderProp(
    (props: Pick<Rack.SelectSingleProps, "value" | "onChange">) => (
      <Rack.SelectSingle {...props} initialQuery={INITIAL_RACK_QUERY} />
    ),
  );

  const NAME_INPUT_PROPS = { level: "h2", variant: "text", placeholder: noun } as const;

  const useConnectModal = Modals.create<PlatformDevice.ConnectParams>(
    ({ deviceKey, close }) => {
      const {
        form,
        save,
        status: stat,
        variant,
      } = useForm({
        query: deviceKey == null ? null : { key: deviceKey },
        initialValues: INITIAL_VALUES,
        beforeValidate,
        afterSave: useCallback(() => close(), [close]),
      });
      return (
        <Modals.Frame className={CSS.B("bus-connect")}>
          <Modals.Header icon={icon}>{`${noun}.Connect`}</Modals.Header>
          <Flex.Box className={CSS.B("content")} grow size="small">
            <Form.Form<typeof PDevice.formSchema> {...form}>
              <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
              <Form.Field<rack.Key> path="rack" label="Connect from" required>
                {renderRack}
              </Form.Field>
              <Properties />
            </Form.Form>
          </Flex.Box>
          <Modals.Footer>
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
          </Modals.Footer>
        </Modals.Frame>
      );
    },
  );

  const EmptyContent = () => {
    const connect = useConnectModal();
    const { close } = Dialog.useContext();
    return (
      <Empty.Action
        message={`No ${noun}s connected`}
        action={`Connect ${noun}`}
        onClick={() => {
          connect();
          close();
        }}
      />
    );
  };

  const Select = () => {
    const connect = useConnectModal();
    return (
      <PlatformDevice.Select
        onConfigure={(deviceKey) => connect({ deviceKey })}
        emptyContent={<EmptyContent />}
        label={caseconv.capitalize(noun)}
        make={make}
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

  return {
    MAKE: make,
    useConnectModal,
    Select,
    COMMANDS,
  };
};

export interface CreateContextMenuItemsParams {
  integration: string;
  useConnectModal: () => (params?: PlatformDevice.ConnectParams) => void;
  useCreateRead: Task.UseCreate;
  useCreateWrite: Task.UseCreate;
}

/** Creates a bus device's context menu: edit its connection and create its tasks. */
export const createContextMenuItems = ({
  integration,
  useConnectModal,
  useCreateRead,
  useCreateWrite,
}: CreateContextMenuItemsParams) => {
  const configs: PlatformDevice.TaskContextMenuItemConfig[] = [
    {
      itemKey: `${integration}.readTask`,
      label: "Create read task",
      useCreate: useCreateRead,
    },
    {
      itemKey: `${integration}.writeTask`,
      label: "Create write task",
      useCreate: useCreateWrite,
    },
  ];
  const ContextMenuItems = (props: Tree.ContextMenuProps): ReactElement => {
    const connect = useConnectModal();
    const onConfigure = (deviceKey: device.Key) => connect({ deviceKey });
    return (
      <>
        <PlatformDevice.EditConnectionMenuItem {...props} onConfigure={onConfigure} />
        <Menu.Divider />
        <PlatformDevice.TaskContextMenuItems
          {...props}
          onConfigure={onConfigure}
          taskContextMenuItemConfigs={configs}
        />
      </>
    );
  };
  return ContextMenuItems;
};
