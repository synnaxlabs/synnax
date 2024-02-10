import { type ReactElement, useCallback } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { type hardware } from "@synnaxlabs/client";
import { Button, Nav, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Tabs } from "@synnaxlabs/pluto/tabs";
import { useQuery } from "@tanstack/react-query";
import { FormProvider, useForm } from "react-hook-form";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { enrich } from "@/hardware/configure/ni/enrich";
import { PhysicalPlanForm } from "@/hardware/device/new/PhysicalPlanForm";
import { PropertiesForm } from "@/hardware/device/new/PropertiesForm";
import { configurationZ, type Configuration } from "@/hardware/device/new/types";
import { type Layout } from "@/layout";

import { buildPhysicalDevicePlan } from "./physicalPlan";
import { buildSoftwareTasks } from "./softwareTasks";
import { SoftwareTasksForm } from "./SoftwareTasksForm";

import "@/hardware/device/new/Configure.css";

import { type NITask } from "@/hardware/configure/ni/types";

const makeDefaultValues = (device: hardware.DevicePayload): Configuration => {
  return {
    properties: {
      key: device.key,
      name: device.name,
      vendor: device.make,
      model: device.model,
      identifier: "",
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
      return await client.hardware.retrieveDevice(key as string);
    },
  });
  if (isPending || data == null) return <div>Loading...</div>;
  return <ConfigureInternal device={data} />;
};

interface ConfigureInternalProps {
  device: hardware.DevicePayload;
}

const ConfigureInternal = ({ device }: ConfigureInternalProps): ReactElement => {
  const client = Synnax.use();
  const methods = useForm<Configuration>({
    defaultValues: makeDefaultValues(device),
    mode: "onBlur",
    reValidateMode: "onBlur",
    criteriaMode: "all",
    resolver: async (data, context, options) => {
      console.log(
        "validation result",
        await zodResolver(configurationZ)(data, context, options),
      );
      return await zodResolver(configurationZ)(data, context, options);
    },
  });

  const TABS: Tabs.TabSpec[] = [
    {
      tabKey: "properties",
      name: "Properties",
    },
    {
      tabKey: "physicalPlan",
      name: "Channel Creation",
    },
    {
      tabKey: "softwareTasks",
      name: "Software Tasks",
    },
  ];

  const tabsProps = Tabs.useStatic({ tabs: TABS });

  const content: Tabs.TabRenderProp = useCallback(
    ({ tabKey }) => {
      switch (tabKey) {
        case "properties":
          return <PropertiesForm />;
        case "physicalPlan":
          return <PhysicalPlanForm />;
        default:
          return <SoftwareTasksForm />;
      }
    },
    [tabsProps.onSelect],
  );

  const handleNext = (): void => {
    void (async () => {
      if (tabsProps.selected === "properties") {
        const ok = await methods.trigger("properties");
        if (!ok) return;
        const existingPlan = methods.getValues().physicalPlan;
        if (existingPlan.groups.length === 0) {
          const enriched = enrich(methods.getValues().properties);
          const plan = buildPhysicalDevicePlan(
            enriched,
            methods.getValues().properties.identifier,
          );
          methods.setValue("physicalPlan.groups", plan.groups);
        }
        tabsProps.onSelect?.("physicalPlan");
      } else if (tabsProps.selected === "physicalPlan") {
        const ok = await methods.trigger("physicalPlan");
        if (!ok) return;
        const existingPlan = methods.getValues().softwarePlan;
        if (existingPlan.tasks.length === 0) {
          const { properties, physicalPlan } = methods.getValues();
          const tasks = buildSoftwareTasks(properties, physicalPlan);
          methods.setValue("softwarePlan.tasks", tasks);
        }
        tabsProps.onSelect?.("softwareTasks");
      } else if (tabsProps.selected === "softwareTasks") {
        const ok = await methods.trigger("softwarePlan");
        if (!ok) return;
        const groups = methods.getValues().physicalPlan.groups;
        if (client == null) return;
        const rack = await client.hardware.retrieveRack(device.rack);
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
            data.map((c, i): void => {
              rawDataChannels[i].synnaxChannel = c.key;
            });
            rawIdx.synnaxChannel = idx.key;
            g.channels.forEach((c) => {
              output.set(c.key, c.synnaxChannel!);
            });
          }),
        );

        const tasks = methods.getValues().softwarePlan.tasks as NITask[];
        if (client == null) return;

        tasks.forEach((t) => {
          t.config.channels.forEach((c) => {
            c.channel = output.get(c.key)!;
          });
        });
        console.log(tasks);

        const t = tasks[0];
        await rack.createTask({
          name: t.name,
          type: t.type,
          config: t.config,
        });
      }
    })();
  };

  return (
    <Align.Space className={CSS.B("device-new-configure")} empty>
      <FormProvider {...methods}>
        <Tabs.Tabs
          direction="x"
          {...tabsProps}
          size="large"
          onSelect={() => {}}
          content={content}
        ></Tabs.Tabs>
        <Nav.Bar size={48} location="bottom">
          <Nav.Bar.End>
            <Button.Button variant="outlined">Cancel</Button.Button>
            <Button.Button onClick={handleNext}>Next Step</Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </FormProvider>
    </Align.Space>
  );
};

export type LayoutType = "hardwareConfigureNew";
export const LAYOUT_TYPE = "hardwareConfigureNew";

export const create =
  (device: string, initial: Omit<Partial<Layout.LayoutState>, "type">) =>
  (): Layout.LayoutState => {
    const { name = "Configure Hardware", location = "mosaic", ...rest } = initial;
    return {
      key: initial.key ?? device,
      type: LAYOUT_TYPE,
      windowKey: initial.key ?? device,
      name,
      window: {
        navTop: true,
        size: { height: 800, width: 1200 },
      },
      location,
      ...rest,
    };
  };
