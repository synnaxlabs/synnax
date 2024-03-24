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
import { Button, Form, Nav, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { useQuery } from "@tanstack/react-query";

import { CSS } from "@/css";
import { enrich } from "@/hardware/configure/ni/enrich";
import { type NITask } from "@/hardware/configure/ni/types";
import { buildPhysicalDevicePlan } from "@/hardware/device/new/physicalPlan/physicalPlan";
import { PhysicalPlanForm } from "@/hardware/device/new/physicalPlan/PhysicalPlanForm";
import {
  extrapolateIdentifier,
  PropertiesForm,
} from "@/hardware/device/new/properties/PropertiesForm";
import { buildSoftwareTasks } from "@/hardware/device/new/softwareTasks/softwareTasks";
import { SoftwareTasksForm } from "@/hardware/device/new/softwareTasks/SoftwareTasksForm";
import { Steps } from "@/hardware/device/new/Steps";
import {
  configurationZ,
  type Configuration,
  type EnrichedProperties,
  type SoftwarePlan,
  type PhysicalPlan,
} from "@/hardware/device/new/types";
import { type Layout } from "@/layout";

import "@/hardware/device/new/Configure.css";

const makeDefaultValues = (device: device.Device): Configuration => {
  return {
    properties: {
      key: device.key,
      name: device.name,
      vendor: device.make,
      model: device.model,
      identifier: extrapolateIdentifier(device.name),
      location: "Dev1",
      analogInput: { portCount: 0 },
      analogOutput: { portCount: 0 },
      digitalInput: { portCount: 0, lineCounts: [] },
      digitalOutput: { portCount: 0, lineCounts: [] },
      digitalInputOutput: { portCount: 0, lineCounts: [] },
    },
    physicalPlan: {
      groups: [],
    },
    softwarePlan: {
      tasks: [],
    },
  };
};

export const Configure = ({ layoutKey }: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const { data, isPending } = useQuery({
    queryKey: [layoutKey, { client }],
    queryFn: async ({ queryKey }) => {
      const [key] = queryKey;
      if (client == null) return;
      return await client.hardware.devices.retrieve(key as string);
    },
  });
  if (isPending || data == null) return <div>Loading...</div>;
  return <ConfigureInternal device={data} />;
};

interface ConfigureInternalProps {
  device: device.Device;
}

const ConfigureInternal = ({ device }: ConfigureInternalProps): ReactElement => {
  const client = Synnax.use();

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
        const existingPlan = methods.get<PhysicalPlan>({ path: "physicalPlan" }).value;
        if (existingPlan.groups.length === 0) {
          const enriched = enrich(
            methods.get<EnrichedProperties>({ path: "properties" }).value,
          );
          const plan = buildPhysicalDevicePlan(
            enriched,
            methods.get<string>({ path: "properties.identifier" }).value,
          );
          methods.set({ path: "physicalPlan.groups", value: plan.groups });
        }
        setStep("physicalPlan");
      } else if (step === "physicalPlan") {
        const ok = methods.validate("physicalPlan");
        if (!ok) return;
        const existingPlan = methods.get<SoftwarePlan>({ path: "softwarePlan" }).value;
        if (existingPlan.tasks.length === 0) {
          const { value: properties } = methods.get<EnrichedProperties>({
            path: "properties",
          });
          const physicalPlan = methods.get<PhysicalPlan>({
            path: "physicalPlan",
          }).value;
          const tasks = buildSoftwareTasks(properties, physicalPlan);
          methods.set({ path: "softwarePlan.tasks", value: tasks });
        }
        setStep("softwareTasks");
      } else if (step === "softwareTasks") {
        const ok = methods.validate("softwarePlan");
        if (!ok) return;
        const groups = methods.get<PhysicalPlan>({ path: "physicalPlan" }).value.groups;
        if (client == null) return;

        const rack = await client.hardware.racks.retrieve(device.rack);
        const output = new Map<string, number>();
        await Promise.all(
          groups.map(async (g) => {
            const rawIdx = g.channels.find((c) => c.isIndex);
            if (rawIdx == null) return;
            const idx = await client.channels.create({
              name: rawIdx.name,
              isIndex: true,
              dataType: rawIdx?.dataType,
            });
            const rawDataChannels = g.channels.filter(
              (c) => !c.isIndex && c.synnaxChannel == null,
            );
            const data = await client.channels.create(
              rawDataChannels.map((c) => ({
                name: c.name,
                dataType: c.dataType,
                index: idx.key,
              })),
            );
            data.forEach((c, i): void => {
              rawDataChannels[i].synnaxChannel = c.key;
            });
            rawIdx.synnaxChannel = idx.key;
            g.channels.forEach((c) => {
              output.set(c.key, c.synnaxChannel);
            });
          }),
        );

        const tasks = methods.get<NITask[]>({ path: "softwarePlan.tasks" }).value;
        if (client == null) return;

        tasks.forEach((t) => {
          t.config.channels.forEach((c) => {
            c.channel = output.get(c.key) as string;
          });
        });

        const t = tasks[0];
        await rack.createTask({
          name: t.name,
          type: t.type,
          config: t.config,
        });
      }
    })();
  };

  let content: ReactElement;
  if (step === "properties") {
    content = <PropertiesForm value={step} onChange={setStep} />;
  } else if (step === "physicalPlan") {
    content = <PhysicalPlanForm />;
  }

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" empty>
      <Form.Form {...methods}>
        <Align.Space className={CSS.B("content")}>{content}</Align.Space>
        <Nav.Bar size={48} location="bottom">
          <Nav.Bar.Start>
            <Steps value={step} onChange={setStep} />
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

export type LayoutType = "hardwareConfigureNew";
export const LAYOUT_TYPE = "hardwareConfigureNew";

export const create =
  (device: string, initial: Omit<Partial<Layout.LayoutState>, "type">) =>
  (): Layout.LayoutState => {
    const { name = "Configure Hardware", location = "window", ...rest } = initial;
    return {
      key: initial.key ?? device,
      type: LAYOUT_TYPE,
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
