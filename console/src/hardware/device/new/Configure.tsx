import { type ReactElement, useCallback } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { Align } from "@synnaxlabs/pluto/align";
import { Tabs } from "@synnaxlabs/pluto/tabs";
import { useForm } from "react-hook-form";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

import { Properties, defaultProperties, propertiesZ } from "./Properties";

import "@/hardware/device/new/Configure.css";

const TABS: Tabs.TabSpec[] = [
  {
    tabKey: "properties",
    name: "Properties",
  },
  {
    tabKey: "hardwareModules",
    name: "Hardware Modules",
  },
  {
    tabKey: "softwareModules",
    name: "Software Modules",
  },
];

export const Configure = (): ReactElement => {
  const { control, watch, setValue } = useForm({
    defaultValues: defaultProperties,
    resolver: zodResolver(propertiesZ),
  });

  const content: Tabs.TabRenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "properties":
        return <Properties control={control} watch={watch} setValue={setValue} />;
      case "hardwareModules":
        return <div>Hardware Modules</div>;
      default:
        return <div>Software Modules</div>;
    }
  }, []);

  const tabs = Tabs.useStatic({ tabs: TABS, content });

  return (
    <Align.Space className={CSS.B("device-new-configure")} direction="x">
      <Tabs.Tabs direction="x" {...tabs}></Tabs.Tabs>
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
      },
      location,
      ...rest,
    };
  };
