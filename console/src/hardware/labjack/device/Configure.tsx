// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/labjack/device/Configure.css";

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
import { deep, strings } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useRef, useState } from "react";

import { CSS } from "@/css";
import {
  configurablePropertiesZ,
  MODEL_KEYS,
  type Properties,
  ZERO_PROPERTIES,
} from "@/hardware/labjack/device/types";
import { type Layout } from "@/layout";

export const Configure = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const { data, isPending } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null) return;
      const dev = await client.hardware.devices.retrieve<Properties>(layoutKey);
      console.log(dev);
      return dev;
    },
  });
  if (isPending || data == null) return <div>Loading...</div>;
  return <ConfigureInternal device={data} onClose={onClose} />;
};

interface ConfigureInternalProps extends Pick<Layout.RendererProps, "onClose"> {
  device: device.Device<Properties>;
}

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

const ConfigureInternal = ({
  device,
  onClose,
}: ConfigureInternalProps): ReactElement => {
  const name = device.name;
  console.log("given name", name);
  const methods = Form.use<typeof configurablePropertiesZ>({
    values: {
      name, // TODO: something odd with naming here if i rename and reconfigure through ontology context menu
      identifier: "",
    },
    schema: configurablePropertiesZ,
  });

  const client = Synnax.use();

  const [step, setStep] = useState<"name" | "identifier">("name");
  const [recommendedIds, setRecommendedIds] = useState<string[]>([]);

  const identifierRef = useRef<HTMLInputElement>(null);

  const addStatus = Status.useAggregator();

  const { isPending, mutate } = useMutation<void, Error, void>({
    mutationKey: [client?.key],
    onError: (e) =>
      addStatus({
        variant: "error",
        message: `Failed to configure ${device.name}`,
        description: e.message,
      }),
    mutationFn: async () => {
      if (client == null) throw new Error("Cannot reach cluster");
      if (step === "name") {
        if (methods.validate("name")) {
          setStep("identifier");
          setRecommendedIds(
            strings.generateShortIdentifiers(methods.get<string>("name").value),
          );
          setTimeout(() => identifierRef.current?.focus(), 100); // TODO jank
        }
        return;
      }
      if (!methods.validate("identifier")) return;
      const model = MODEL_KEYS.find((m) => m === device.model);
      if (model == null) throw new Error(`Unknown Model: ${device.model}`);
      await client.hardware.devices.create({
        ...device,
        configured: true,
        name: methods.get<string>("name").value,
        properties: {
          ...deep.copy(ZERO_PROPERTIES),
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
          {step == "identifier" && (
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
            {step === "identifier" ? "To Save" : "To Next"}
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

export const CONFIGURE_LAYOUT_TYPE = "configure_LabJack";
export type LayoutType = typeof CONFIGURE_LAYOUT_TYPE;

export const createConfigureLayout =
  (key: string, initial: Omit<Partial<Layout.State>, "type">) => (): Layout.State => {
    const { name = "LabJack.Device.Configure", location = "modal", ...rest } = initial;
    return {
      key,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: key, //TODO: difference between key and windowKey?
      name,
      icon: "Device",
      window: {
        navTop: true,
        size: { height: 350, width: 800 },
        resizable: true,
      },
      location,
      ...rest,
    };
  };
