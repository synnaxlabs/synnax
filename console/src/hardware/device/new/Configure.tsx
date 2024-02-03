import { type ReactElement, useCallback } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Nav } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Tabs } from "@synnaxlabs/pluto/tabs";
import { FormProvider, useForm } from "react-hook-form";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { enrich } from "@/hardware/configure/ni/enrich";
import { PhysicalPlanForm } from "@/hardware/device/new/PhysicalPlanForm";
import { PropertiesForm } from "@/hardware/device/new/PropertiesForm";
import { configurationZ, type Configuration } from "@/hardware/device/new/types";
import { type Layout } from "@/layout";

import { buildPhysicalDevicePlan } from "./physicalPlan";

import "@/hardware/device/new/Configure.css";

import { SoftwareTasksForm } from "./SoftwareTasksForm";
import { buildSoftwareTasks } from "./softwareTasks";

const DEFAULT_VALUES: Configuration = {
  properties: {
    key: "",
    name: "",
    vendor: "other",
    model: "",
    identifier: "",
    location: "",
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

export const Configure = (): ReactElement => {
  const methods = useForm<Configuration>({
    defaultValues: DEFAULT_VALUES,
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
          const enriched = enrich(methods.getValues().properties);
          const physicalPlan = buildPhysicalDevicePlan(
            enriched,
            methods.getValues().properties.identifier,
          );
          console.log(physicalPlan.groups);
          methods.setValue("physicalPlan.groups", physicalPlan.groups);
          return <PhysicalPlanForm />;
        default:
          const softwarePlan = buildSoftwareTasks(
            methods.getValues().properties,
            methods.getValues().physicalPlan,
          );
          methods.setValue("softwarePlan.tasks", softwarePlan);
          console.log(softwarePlan);
          return <SoftwareTasksForm />;
      }
    },
    [tabsProps.onSelect],
  );

  return (
    <Align.Space className={CSS.B("device-new-configure")} empty>
      <FormProvider {...methods}>
        <Tabs.Tabs
          direction="x"
          {...tabsProps}
          size="large"
          content={content}
        ></Tabs.Tabs>
        <Nav.Bar size={48} location="bottom">
          <Nav.Bar.End>
            <Button.Button variant="outlined">Cancel</Button.Button>
            <Button.Button onClick={async () => await methods.trigger()}>
              Next Step
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </FormProvider>
    </Align.Space>
  );
};

export type LayoutType = "hardwareConfigureNew";
export const LAYOUT_TYPE = "hardwareConfigureNew";

export const create =
  (initial: Omit<Partial<Layout.LayoutState>, "type">) => (): Layout.LayoutState => {
    const { name = "Configure Hardware", location = "mosaic", ...rest } = initial;
    const k = uuidv4();
    return {
      key: initial.key ?? k,
      type: LAYOUT_TYPE,
      windowKey: initial.key ?? k,
      name,
      window: {
        navTop: true,
        size: { height: 800, width: 1200 },
      },
      location,
      ...rest,
    };
  };
