import { type ReactElement, useCallback } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Nav } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Tabs } from "@synnaxlabs/pluto/tabs";
import { nanoid } from "nanoid";
import { FormProvider, useForm } from "react-hook-form";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { type Configuration, configurationZ } from "@/hardware/device/new/types";
import { type Layout } from "@/layout";

import { Hardware } from "./Hardware";
import { PropertiesForm } from "./Properties";

import "@/hardware/device/new/Configure.css";

const TABS: Tabs.TabSpec[] = [
  {
    tabKey: "hardwareModules",
    name: "Hardware Modules",
  },
  {
    tabKey: "properties",
    name: "Properties",
  },

  {
    tabKey: "softwareTasks",
    name: "Software Tasks",
  },
];

const DEFAULT_VALUES: Configuration = {
  name: "GSE DAQ",
  vendor: "other",
  model: "PXIe-1082",
  identifier: "GSE",
  key: "123259",
  isChassis: true,
  slotCount: 2,
  modules: [
    {
      key: "123259-2",
      model: "PXI-6225",
      analogInCount: 80,
      busType: "PXI",
      category: "voltage",
      slot: 1,
      groups: [
        {
          name: "Analog In",
          channelPrefix: "gsdq_s1_ai_",
          key: "123259-2-0",
          channels: Array.from({ length: 80 }, (_, i) => ({
            key: nanoid(),
            dataType: "float32",
            name: `gsdq_s1_ai_${i}`,
            port: i + 1,
            line: 0,
            channel: i,
            group: 0,
          })),
        },
      ],
    },
    {
      key: "123259-1",
      model: "PXI-6514 ",
      analogInCount: 0,
      digitalInCount: 32,
      digitalOutCount: 32,
      slot: 2,
      groups: [
        {
          key: "123259-1-0",
          name: "Digital In",
          channelPrefix: "gsdq_s1_di_",
          channelSuffix: "",
          channels: [
            {
              key: nanoid(),
              name: "gsdq_s1_di_time",
              port: 0,
              line: 0,
              isIndex: true,
              dataType: "timestamp",
            },
            ...Array.from({ length: 32 }, (_, i) => ({
              key: nanoid(),
              name: `gsdq_s1_di_${i}`,
              port: i + 1,
              line: 0,
              dataType: "float32",
              group: 0,
              isIndex: false,
            })),
          ],
        },
        ...Array.from({ length: 32 }, (_, i) => ({
          key: `123259-1-1-${i}`,
          name: `Digital Out Command ${i}`,
          channelPrefix: `gsdq_s1_do_${i}`,
          channelSuffix: "_cmd",
          channels: [
            {
              key: nanoid(),
              name: `gsdq_s1_do_${i}_time`,
              group: i,
              port: 0,
              line: 0,
              isIndex: true,
              dataType: "timestamp",
            },
            {
              key: nanoid(),
              name: `gsdq_s1_do_${i}_cmd`,
              group: i,
              port: 0,
              line: 0,
              isIndex: true,
              dataType: "uint8",
            },
          ],
        })),
        {
          name: "Digital Output States",
          key: "123259-1-2",
          channelPrefix: "gsdq_s1_do_",
          channelSuffix: "_state",
          channels: [
            {
              key: nanoid(),
              name: "gsdq_s1_do_state_time",
              port: 0,
              line: 0,
              isIndex: true,
              dataType: "timestamp",
            },
            ...Array.from({ length: 32 }, (_, i) => ({
              key: nanoid(),
              name: `gsdq_s1_do_${i}_state`,
              channel: i,
              port: 1,
              line: i,
              dataType: "uint8",
              isIndex: false,
            })),
          ],
        },
      ],
    },
  ],
};

export const Configure = (): ReactElement => {
  const methods = useForm<Configuration>({
    defaultValues: DEFAULT_VALUES,
    resolver: zodResolver(configurationZ),
  });

  const tabs = Tabs.useStatic({ tabs: TABS });

  const content: Tabs.TabRenderProp = useCallback(
    ({ tabKey }) => {
      switch (tabKey) {
        case "properties":
          return <PropertiesForm />;
        case "hardwareModules":
          return <Hardware />;
        default:
          return <div>Software Modules</div>;
      }
    },
    [tabs.onSelect],
  );

  return (
    <Align.Space className={CSS.B("device-new-configure")} empty>
      <FormProvider {...methods}>
        <Tabs.Tabs direction="x" {...tabs} content={content}></Tabs.Tabs>
        <Nav.Bar size={48} location="bottom">
          <Nav.Bar.End>
            <Button.Button variant="outlined">Cancel</Button.Button>
            <Button.Button>Next Step</Button.Button>
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
    const { name = "Configure Hardware", location = "window", ...rest } = initial;
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
