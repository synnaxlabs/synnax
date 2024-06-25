// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/device/Configure.css";

import { type device } from "@synnaxlabs/client";
import { Button, Form, Nav, Steps, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";

import { CSS } from "@/css";
import { buildPhysicalDevicePlan as buildGroups } from "@/hardware/ni/device/buildGroups";
import { Confirm } from "@/hardware/ni/device/Confirm";
import { CreateChannels } from "@/hardware/ni/device/CreateChannels";
import { enrich } from "@/hardware/ni/device/enrich/enrich";
import { extrapolateIdentifier, PropertiesForm } from "@/hardware/ni/device/Properties";
import {
  type Configuration,
  configurationZ,
  type EnrichedProperties,
  GroupConfig,
} from "@/hardware/ni/device/types";
import { type Layout } from "@/layout";

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
];

export const Configure = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const { data, isPending } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null) return;
      return await client.hardware.devices.retrieve(layoutKey);
    },
  });
  if (isPending || data == null) return <div>Loading...</div>;
  return <ConfigureInternal device={data} onClose={onClose} />;
};

interface ConfigureInternalProps extends Pick<Layout.RendererProps, "onClose"> {
  device: device.Device;
}

const ConfigureInternal = ({
  device,
  onClose,
}: ConfigureInternalProps): ReactElement => {
  const [step, setStep] = useState("properties");

  const methods = Form.use<typeof configurationZ>({
    values: makeDefaultValues(device),
    schema: configurationZ,
  });

  const handleNext = useMutation({
    mutationKey: [step],
    mutationFn: async () => {
      if (step === "properties") {
        const ok = methods.validate("properties");
        if (!ok) return;
        const existingGroups = methods.get<GroupConfig[]>("groups").value;
        if (existingGroups.length === 0) {
          const enriched = enrich(methods.get<EnrichedProperties>("properties").value);
          const groups = buildGroups(
            enriched,
            methods.get<string>("properties.identifier").value,
          );
          methods.set("groups", groups);
        }
        setStep("createChannels");
      } else if (step === "createChannels") {
        const ok = methods.validate("groups");
        if (!ok) return;
        setStep("confirm");
      } else onClose();
    },
  });

  const client = Synnax.use();

  const [progress, setProgress] = useState<string | undefined>("");

  const confirm = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (client == null) return;
      const channelsGroup = await client.channels.retrieveGroup();
      const deviceOtgGroup = await client.ontology.groups.create(
        channelsGroup.ontologyID,
        device.name,
      );
      const groups = methods.get<GroupConfig[]>("groups").value;
      for (const group of groups) {
        const otgGroup = await client.ontology.groups.create(
          deviceOtgGroup.ontologyID,
          group.name,
        );
        const rawIdx = group.channels.find((c) => c.isIndex);
        setProgress(`Creating index for ${group.name}`);
        if (rawIdx == null) return;
        const idx = await client.channels.create({
          name: rawIdx.name,
          isIndex: true,
          dataType: rawIdx.dataType,
        });

        const rawDataChannels = group.channels.filter(
          (c) => !c.isIndex && c.synnaxChannel == null,
        );
        setProgress(`Creating data channels for ${group.name}`);
        const created = await client.channels.create(
          rawDataChannels.map((c) => ({
            name: c.name,
            dataType: c.dataType,
            index: idx.key,
          })),
        );
        await client.ontology.moveChildren(
          channelsGroup.ontologyID,
          otgGroup.ontologyID,
          idx.ontologyID,
          ...created.map((c) => c.ontologyID),
        );
      }
    },
  });

  let content: ReactElement;
  if (step === "properties") content = <PropertiesForm />;
  else if (step === "createChannels") content = <CreateChannels />;
  else if (step === "confirm")
    content = <Confirm confirm={confirm} progress={progress} />;
  else content = <h1>Unknown step: {step}</h1>;

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
            <Button.Button
              onClick={() => handleNext.mutate()}
              disabled={confirm.isPending || (confirm.isIdle && step === "confirm")}
            >
              {confirm.isSuccess ? "Done" : "Next"}
            </Button.Button>
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
    const { name = "Configure NI Device", location = "window", ...rest } = initial;
    return {
      key: initial.key ?? device,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: initial.key ?? device,
      name,
      window: {
        navTop: true,
        size: { height: 900, width: 1200 },
        resizable: true,
      },
      location,
      ...rest,
    };
  };
