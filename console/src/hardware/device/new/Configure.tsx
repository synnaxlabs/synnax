import { useEffect, type ReactElement, useCallback } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { Icon } from "@synnaxlabs/media";
import { Nav, Select } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Button } from "@synnaxlabs/pluto/button";
import { Input } from "@synnaxlabs/pluto/input";
import { Tabs } from "@synnaxlabs/pluto/tabs";
import { Text } from "@synnaxlabs/pluto/text";
import { useForm } from "react-hook-form";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

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
  const content = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "properties":
        return <Properties />;
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

const VENDORS = ["ni", "other"] as const;

const vendorsZ = z.enum(VENDORS);

type Vendor = z.infer<typeof vendorsZ>;

interface VendorListItem {
  key: Vendor;
  logo: ReactElement;
  name: string;
}

const DEVICE_VENDORS: VendorListItem[] = [
  {
    key: "ni",
    logo: <Icon.Logo.NI className="vendor-logo " />,
    name: "National Instruments",
  },
  {
    key: "other",
    logo: <Icon.Hardware className="vendor-logo " />,
    name: "Other",
  },
];

const propertiesFormZ = z.object({
  vendor: vendorsZ,
  name: z.string(),
  key: z.string(),
  identifier: z.string().refine((s) => !s.includes(" ") && /^[a-zA-Z0-9]+$/.test(s), {
    message: "Only alphanumeric characters allowed",
  }),
});

const defaultProperties = {
  vendor: "ni",
  name: "",
  model: "NI USB 6000",
  key: "01A100CE",
  identifier: "",
};

const TARGET_IDENTIFIER_LENGTH = 3;

const extrapolateIdentifier = (identifier: string): string => {
  const words = identifier.split(" ");
  let toGrabFromFirst = TARGET_IDENTIFIER_LENGTH - words.length;
  if (toGrabFromFirst < 1) toGrabFromFirst = 1;
  return words
    .map((word, i) => (i === 0 ? word.slice(0, toGrabFromFirst) : word[0]))
    .join("");
};

export const Properties = (): ReactElement => {
  const {
    control: c,
    watch,
    formState,
    setValue,
  } = useForm({
    defaultValues: defaultProperties,
    resolver: zodResolver(propertiesFormZ),
  });

  useEffect(
    () =>
      watch((value, { name }) => {
        if (
          name !== "name" ||
          value.name == null ||
          formState.touchedFields.identifier === true
        )
          return;
        setValue("identifier", extrapolateIdentifier(value.name));
      }).unsubscribe,
    [watch, setValue],
  );

  return (
    <Align.Space
      direction="y"
      className={CSS.B("properties")}
      align="stretch"
      justify="center"
    >
      <Text.Text level="h1">Let's configure your device</Text.Text>
      <Text.Text level="p">
        Confirm the details of your hardware and give it a name.
      </Text.Text>
      <Input.ItemControlled<Vendor> control={c} name="vendor" label="Vendor">
        {(props) => {
          return (
            <Select.DropdownButton<Vendor, VendorListItem>
              className={CSS.B("vendor")}
              data={DEVICE_VENDORS}
              allowNone={false}
              columns={[
                {
                  key: "logo",
                  render: ({ entry }) => entry.logo,
                },
                {
                  key: "name",
                },
              ]}
              {...props}
            >
              {({ selected: s, toggle }) => (
                <Button.Button
                  iconSpacing="small"
                  onClick={toggle}
                  variant="outlined"
                  startIcon={s?.logo}
                >
                  {s?.name}
                </Button.Button>
              )}
            </Select.DropdownButton>
          );
        }}
      </Input.ItemControlled>
      <Input.ItemControlled<string> control={c} name="key" label="Serial Number" />
      <Input.ItemControlled<string> control={c} name="model" label="Model" />
      <Input.ItemControlled<string> control={c} name="name" label="Name" />
      <Input.ItemControlled<string> control={c} name="identifier" label="Identifier" />
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
