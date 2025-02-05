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
import {
  Align,
  Button,
  Form,
  Nav,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { deep, strings, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useRef, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { identifierZ, nameZ } from "@/hardware/common/device/types";
import { type Layout } from "@/layout";

export const CONFIGURE_LAYOUT: Omit<Layout.BaseState, "type" | "key"> = {
  name: "Configure",
  icon: "Device",
  location: "modal",
  window: { resizable: false, size: { height: 350, width: 800 }, navTop: true },
};

export interface ConfigureProps<P extends UnknownRecord> extends Layout.RendererProps {
  zeroProperties: P;
}

export const Configure = <P extends UnknownRecord>({
  layoutKey,
  ...rest
}: ConfigureProps<P>): ReactElement => {
  const client = Synnax.use();
  const {
    isPending,
    isError,
    error,
    data: device,
  } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null) throw new Error("Cannot reach server");
      return await client.hardware.devices.retrieve<P>(layoutKey);
    },
  });
  if (isPending)
    return (
      <Status.Text.Centered variant="loading" level="h2">
        Fetching device from server
      </Status.Text.Centered>
    );
  if (isError)
    return (
      <Align.Space direction="y" grow align="center" justify="center">
        <Text.Text level="h2" color={Status.variantColors.error}>
          Failed to load data for device with key {layoutKey}
        </Text.Text>
        <Text.Text level="p" color={Status.variantColors.error}>
          {error.message}
        </Text.Text>
      </Align.Space>
    );
  return <Internal device={device} {...rest} />;
};

interface InternalProps<P extends UnknownRecord>
  extends Pick<Layout.RendererProps, "onClose"> {
  device: device.Device<P>;
  zeroProperties: P;
}

const configurablePropertiesZ = z.object({ name: nameZ, identifier: identifierZ });

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

const Internal = <P extends UnknownRecord>({
  device,
  device: { name },
  onClose,
  zeroProperties,
}: InternalProps<P>): ReactElement => {
  const methods = Form.use<typeof configurablePropertiesZ>({
    values: { name, identifier: "" },
    schema: configurablePropertiesZ,
  });
  const client = Synnax.use();
  const [step, setStep] = useState<"name" | "identifier">("name");
  const [recommendedIds, setRecommendedIds] = useState<string[]>([]);
  const identifierRef = useRef<HTMLInputElement>(null);
  const handleException = Status.useExceptionHandler();
  const { isPending, mutate } = useMutation<void, Error, void>({
    onError: (e) => handleException(e, `Failed to configure ${name}`),
    mutationFn: async () => {
      if (client == null) throw new Error("Cannot reach server");
      if (step === "name") {
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
    <Align.Space className={CSS.B("configure")} align="stretch" empty>
      <Form.Form {...methods}>
        <Align.Space
          align="stretch"
          style={{ padding: "5rem" }}
          justify="center"
          grow
          size="large"
        >
          {step === "name" && (
            <>
              <Text.Text level="h4" shade={7}>
                Before you can acquire data from this device, we'll need a few details.
                To start off, enter a name so it's easy to look up later.
              </Text.Text>
              <Form.TextField
                inputProps={{ variant: "natural", level: "h2", autoFocus: true }}
                path="name"
                label="Name"
                autoFocus
              />
            </>
          )}
          {step === "identifier" && (
            <>
              <Text.Text level="h4" shade={7}>
                Next, we'll need a short identifier for{" "}
                {methods.get<string>("name").value}. We'll use this as a prefix for all
                channels associated with this device. We've generated some suggestions
                below.
              </Text.Text>
              <Align.Space direction="y" size="small">
                <Form.TextField
                  inputProps={{ variant: "natural", level: "h2", ref: identifierRef }}
                  path="identifier"
                  label="Identifier"
                  autoFocus
                />
                <Align.Space direction="x">
                  <Button.Icon variant="text" size="small" disabled>
                    <Icon.Bolt />
                  </Button.Icon>
                  {recommendedIds.map((id) => (
                    <Button.Button
                      key={id}
                      variant="suggestion"
                      size="small"
                      onClick={() => {
                        methods.set("identifier", id);
                        identifierRef.current?.focus();
                      }}
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
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            {step === "identifier" ? "Save" : "Next"}
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button
            type="submit"
            loading={isPending}
            disabled={isPending}
            onClick={() => mutate()}
            triggers={[SAVE_TRIGGER]}
          >
            {step === "identifier" ? "Save" : "Next"}
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
