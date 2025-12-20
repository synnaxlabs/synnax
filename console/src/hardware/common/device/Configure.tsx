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
import {
  Button,
  Device as Core,
  Device,
  Flex,
  Form,
  Icon,
  Nav,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { deep, type record, status, strings } from "@synnaxlabs/x";
import { useCallback, useRef, useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { identifierZ, nameZ } from "@/hardware/common/device/types";
import { type Layout } from "@/layout";
import { Triggers } from "@/triggers";

export const CONFIGURE_LAYOUT: Omit<Layout.BaseState, "type"> = {
  icon: "Device",
  location: "modal",
  name: "Configure",
  window: { resizable: false, size: { height: 325, width: 800 }, navTop: true },
};

interface InternalProps<
  Properties extends record.Unknown,
  Make extends string,
  Model extends string,
> extends Pick<Layout.RendererProps, "onClose"> {
  device: device.Device<Properties, Make, Model>;
  initialProperties: Properties;
}

const configurablePropertiesZ = z.object({ name: nameZ, identifier: identifierZ });
type ConfigurablePropertiesZ = typeof configurablePropertiesZ;

const Internal = <
  Properties extends record.Unknown,
  Make extends string,
  Model extends string,
>({
  device,
  device: { name },
  onClose,
  initialProperties,
}: InternalProps<Properties, Make, Model>) => {
  const methods = Form.use<ConfigurablePropertiesZ>({
    values: { name, identifier: "" },
    schema: configurablePropertiesZ,
  });
  const [step, setStep] = useState<"name" | "identifier">("name");
  const isNameStep = step === "name";
  const triggerAction = isNameStep ? "Next" : "Save";
  const [recommendedIds, setRecommendedIds] = useState<string[]>([]);
  const identifierRef = useRef<HTMLInputElement>(null);
  const deviceToCreate = () => ({
    ...device,
    configured: true,
    name: methods.get<string>("name").value,
    properties: {
      ...deep.copy(initialProperties),
      ...device.properties,
      identifier: methods.get<string>("identifier").value,
    },
  });
  const { update, variant } = Core.useCreate({
    beforeUpdate: useCallback(async () => {
      if (isNameStep) {
        if (methods.validate("name")) {
          setStep("identifier");
          setRecommendedIds(
            strings.createShortIdentifiers(methods.get<string>("name").value),
          );
          setTimeout(() => identifierRef.current?.focus(), 100);
        }
        return false;
      }
      if (!methods.validate("identifier")) return false;
      return deviceToCreate();
    }, [isNameStep, methods, setStep, setRecommendedIds, identifierRef]),
    afterSuccess: useCallback(() => onClose(), [onClose]),
  });

  return (
    <Flex.Box align="stretch" className={CSS.B("configure")} empty>
      <Form.Form<typeof configurablePropertiesZ> {...methods}>
        <Flex.Box
          align="stretch"
          justify="center"
          grow
          gap="large"
          style={{ padding: "5rem" }}
        >
          {isNameStep ? (
            <>
              <Text.Text>
                Before you can acquire data from this device, we'll need a few details.
                To start off, enter a name so it's easy to look up later.
              </Text.Text>
              <Form.TextField
                autoFocus
                inputProps={{ autoFocus: true, level: "h2", variant: "text" }}
                label="Name"
                path="name"
              />
            </>
          ) : (
            <>
              <Text.Text>
                Next, we'll need a short identifier for{" "}
                {methods.get<string>("name").value}. We'll use this as a prefix for all
                channels associated with this device. We've given you some suggestions
                below.
              </Text.Text>
              <Flex.Box gap="small">
                <Form.TextField
                  autoFocus
                  label="Identifier"
                  inputProps={{ level: "h2", ref: identifierRef, variant: "text" }}
                  path="identifier"
                />
                <Flex.Box x>
                  <Button.Button disabled size="small" variant="text">
                    <Icon.Bolt />
                  </Button.Button>
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
                </Flex.Box>
              </Flex.Box>
            </>
          )}
        </Flex.Box>
      </Form.Form>
      <Nav.Bar location="bottom" size={48} bordered>
        <Triggers.SaveHelpText action={triggerAction} />
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={() => update(deviceToCreate())}
            variant="filled"
            trigger={Triggers.SAVE}
            type="submit"
          >
            {triggerAction}
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Flex.Box>
  );
};

export interface ConfigureProps<
  Properties extends record.Unknown,
  Make extends string,
  Model extends string,
>
  extends
    Layout.RendererProps,
    Pick<InternalProps<Properties, Make, Model>, "initialProperties"> {}

export const Configure = <
  Properties extends record.Unknown,
  Make extends string,
  Model extends string,
>({
  layoutKey,
  ...rest
}: ConfigureProps<Properties, Make, Model>) => {
  const { data, status, variant } = Device.useRetrieve({ key: layoutKey });
  if (variant !== "success") return <Status.Summary status={status} />;
  return <Internal device={data} {...rest} />;
};
