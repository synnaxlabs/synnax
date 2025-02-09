// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/device/Configure.css";

import { type device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Form, Nav, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { deep, strings, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { identifierZ, nameZ } from "@/hardware/common/device/types";
import { type Layout } from "@/layout";
import { Triggers } from "@/triggers";

export const CONFIGURE_LAYOUT: Omit<Layout.BaseState, "type" | "key"> = {
  icon: "Device",
  location: "modal",
  name: "Configure",
  window: { resizable: false, size: { height: 350, width: 800 }, navTop: true },
};

interface InternalProps<
  Properties extends UnknownRecord,
  Make extends string,
  Model extends string,
> extends Pick<Layout.RendererProps, "onClose"> {
  device: device.Device<Properties, Make, Model>;
  zeroProperties: Properties;
}

const configurablePropertiesZ = z.object({ name: nameZ, identifier: identifierZ });
type ConfigurablePropertiesZ = typeof configurablePropertiesZ;

const Internal = <
  Properties extends UnknownRecord,
  Make extends string,
  Model extends string,
>({
  device,
  device: { name },
  onClose,
  zeroProperties,
}: InternalProps<Properties, Make, Model>) => {
  const methods = Form.use<ConfigurablePropertiesZ>({
    values: { name, identifier: "" },
    schema: configurablePropertiesZ,
  });
  const client = Synnax.use();
  const [step, setStep] = useState<"name" | "identifier">("name");
  const isNameStep = step === "name";
  const triggerAction = isNameStep ? "Next" : "Save";
  const [recommendedIds, setRecommendedIds] = useState<string[]>([]);
  const identifierRef = useRef<HTMLInputElement>(null);
  const handleException = Status.useExceptionHandler();
  const { isPending, mutate } = useMutation<void, Error, void>({
    onError: (e) => handleException(e, `Failed to configure ${name}`),
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (isNameStep) {
        if (methods.validate("name")) {
          setStep("identifier");
          setRecommendedIds(
            strings.generateShortIdentifiers(methods.get<string>("name").value),
          );
          setTimeout(() => identifierRef.current?.focus(), 100);
        }
        return;
      }
      if (!methods.validate("identifier")) return;
      await client.hardware.devices.create({
        ...device,
        configured: true,
        name: methods.get<string>("name").value,
        properties: {
          ...deep.copy(zeroProperties),
          ...device.properties,
          enriched: true,
          identifier: methods.get<string>("identifier").value,
        },
      });
      onClose();
    },
  });
  return (
    <Align.Space align="stretch" className={CSS.B("configure")} empty>
      <Form.Form {...methods}>
        <Align.Space
          align="stretch"
          justify="center"
          grow
          size="large"
          style={{ padding: "5rem" }}
        >
          {isNameStep ? (
            <>
              <Text.Text level="h4" shade={7}>
                Before you can acquire data from this device, we'll need a few details.
                To start off, enter a name so it's easy to look up later.
              </Text.Text>
              <Form.TextField
                autoFocus
                inputProps={{ autoFocus: true, level: "h2", variant: "natural" }}
                label="Name"
                path="name"
              />
            </>
          ) : (
            <>
              <Text.Text level="h4" shade={7}>
                Next, we'll need a short identifier for{" "}
                {methods.get<string>("name").value}. We'll use this as a prefix for all
                channels associated with this device. We've generated some suggestions
                below.
              </Text.Text>
              <Align.Space size="small">
                <Form.TextField
                  autoFocus
                  label="Identifier"
                  inputProps={{ level: "h2", ref: identifierRef, variant: "natural" }}
                  path="identifier"
                />
                <Align.Space direction="x">
                  <Button.Icon disabled size="small" variant="text">
                    <Icon.Bolt />
                  </Button.Icon>
                  {recommendedIds.map((id) => (
                    <Button.Button
                      key={id}
                      onClick={() => {
                        methods.set("identifier", id);
                        identifierRef.current?.focus();
                      }}
                      size="small"
                      variant="suggestion"
                    >
                      {id}
                    </Button.Button>
                  ))}
                </Align.Space>
              </Align.Space>
            </>
          )}
        </Align.Space>
      </Form.Form>
      <Nav.Bar location="bottom" size={48}>
        <Triggers.SaveHelpText action={triggerAction} />
        <Nav.Bar.End>
          <Button.Button
            disabled={isPending}
            loading={isPending}
            onClick={() => mutate()}
            triggers={Triggers.SAVE}
            type="submit"
          >
            {triggerAction}
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

export interface ConfigureProps<
  Properties extends UnknownRecord,
  Make extends string,
  Model extends string,
> extends Layout.RendererProps,
    Pick<InternalProps<Properties, Make, Model>, "zeroProperties"> {}

export const Configure = <
  Properties extends UnknownRecord,
  Make extends string,
  Model extends string,
>({
  layoutKey,
  ...rest
}: ConfigureProps<Properties, Make, Model>) => {
  const client = Synnax.use();
  const { data, error, isError, isPending } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      return await client.hardware.devices.retrieve<Properties, Make, Model>(layoutKey);
    },
  });
  if (isPending)
    return (
      <Status.Text.Centered level="h4" variant="loading">
        Fetching device from server
      </Status.Text.Centered>
    );
  if (isError) {
    const color = Status.variantColors.error;
    return (
      <Align.Space align="center" grow justify="center">
        <Text.Text color={color} level="h2">
          Failed to load data for device with key {layoutKey}
        </Text.Text>
        <Text.Text color={color} level="p">
          {error.message}
        </Text.Text>
      </Align.Space>
    );
  }
  return <Internal device={data} {...rest} />;
};
