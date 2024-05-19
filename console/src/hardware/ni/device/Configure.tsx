// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { type device } from "@synnaxlabs/client";
import { Button, Form, Nav, Synnax, Steps } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { useQuery } from "@tanstack/react-query";

import { CSS } from "@/css";
import { Confirm } from "@/hardware/ni/device/Confirm";
import { buildPhysicalDevicePlan } from "@/hardware/ni/device/buildGroups";
import { CreateChannels } from "@/hardware/ni/device/CreateChannels";
import { extrapolateIdentifier, PropertiesForm } from "@/hardware/ni/device/Properties";
import {
  configurationZ,
  GroupConfig,
  type Configuration,
  type EnrichedProperties,
} from "@/hardware/ni/device/types";
import { type Layout } from "@/layout";

import { enrich } from "@/hardware/ni/device/enrich/enrich";

import "@/hardware/ni/device/Configure.css";

const makeDefaultValues = (device: device.Device): Configuration => {
  return {
    properties: {
      key: device.key,
      name: device.name,
      vendor: device.make as "NI",
      model: device.model,
      identifier: extrapolateIdentifier(device.name),
      location: "Dev1",
      analogInput: { portCount: 0 },
      analogOutput: { portCount: 0 },
      digitalInput: { portCount: 0, lineCounts: [] },
      digitalOutput: { portCount: 0, lineCounts: [] },
      digitalInputOutput: { portCount: 0, lineCounts: [] },
    },
    groups: [],
  };
};

const STEPS: Steps.Step[] = [
  {
    key: "properties",
    title: "Define Properties",
  },
  {
    key: "createChannels",
    title: "Create Channels",
  },
  {
    key: "confirm",
    title: "Confirm",
  },
  {
    key: "nextSteps",
    title: "Next Steps",
  },
];

export const Configure = ({ layoutKey }: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const { data, isPending } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async ({ queryKey }) => {
      const [key] = queryKey;
      if (client == null) return;
      return await client.hardware.devices.retrieve(layoutKey);
    },
  });
  if (isPending || data == null) return <div>Loading...</div>;
  return <ConfigureInternal device={data} />;
};

interface ConfigureInternalProps {
  device: device.Device;
}

const ConfigureInternal = ({ device }: ConfigureInternalProps): ReactElement => {
  const [step, setStep] = useState("properties");

  const methods = Form.use<typeof configurationZ>({
    values: makeDefaultValues(device),
    schema: configurationZ,
  });

  const handleNext = (): void => {
    void (async () => {
      if (step === "properties") {
        const ok = methods.validate("properties");
        if (!ok) return;
        const existingGroups = methods.get<GroupConfig[]>({ path: "groups" }).value;
        if (existingGroups.length === 0) {
          const enriched = enrich(
            methods.get<EnrichedProperties>({ path: "properties" }).value,
          );
          const groups = buildPhysicalDevicePlan(
            enriched,
            methods.get<string>({ path: "properties.identifier" }).value,
          );
          methods.set({ path: "groups", value: groups });
        }
        setStep("createChannels");
      } else if (step === "createChannels") {
        const ok = methods.validate("groups");
        if (!ok) return;
        setStep("confirm");
      }
    })();
  };

  let content: ReactElement;
  if (step === "properties") content = <PropertiesForm />;
  else if (step === "createChannels") content = <CreateChannels />;
  else if (step === "confirm") content = <Confirm />;

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" empty>
      <Form.Form {...methods}>
        <Align.Space className={CSS.B("content")}>{content}</Align.Space>
        <Nav.Bar size={48} location="bottom">
          <Nav.Bar.Start>
            <Steps.Steps steps={STEPS} value={step} onChange={setStep} />
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <Button.Button variant="outlined">Cancel</Button.Button>
            <Button.Button onClick={handleNext}>Next Step</Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </Form.Form>
    </Align.Space>
  );
};

export const CONFIGURE_LAYOUT_TYPE = "configure_NI";
export type LayoutType = typeof CONFIGURE_LAYOUT_TYPE;

export const createConfigureLayout =
  (device: string, initial: Omit<Partial<Layout.State>, "type">) =>
  (): Layout.State => {
    const { name = "Configure Hardware", location = "window", ...rest } = initial;
    return {
      key: initial.key ?? device,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: initial.key ?? device,
      name,
      window: {
        navTop: true,
        size: { height: 900, width: 1200 },
        resizable: false,
      },
      location,
      ...rest,
    };
  };
