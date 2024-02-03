import { useEffect, type ReactElement } from "react";

import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { useFormContext, useFormState, useWatch } from "react-hook-form";

import { CSS } from "@/css";
import { SelectModel } from "@/hardware/configure/ni/SelectModel";
import { SelectVendor } from "@/hardware/device/new/SelectVendor";
import { type Configuration, type Vendor } from "@/hardware/device/new/types";

import "@/hardware/device/new/PropertiesForm.css";

const MIN_IDENTIFIER_LENGTH = 3;
const MAX_IDENTIFIER_LENGTH = 5;

const extrapolateIdentifier = (identifier: string): string => {
  const words = identifier.split(" ");
  let toGrabFromFirst = MIN_IDENTIFIER_LENGTH - words.length + 1;
  if (toGrabFromFirst < 1) toGrabFromFirst = 1;
  return words
    .map((word, i) => (i === 0 ? word.slice(0, toGrabFromFirst) : word[0]))
    .join("")
    .toUpperCase()
    .slice(0, MAX_IDENTIFIER_LENGTH);
};

export const PropertiesForm = (): ReactElement => {
  const { setValue } = useFormContext<Configuration>();

  const name = useWatch<Configuration>({ name: "properties.name" }) as string;
  const identifier: string = useWatch<Configuration>({
    name: "properties.identifier",
  }) as string;
  const id = useFormState({ name: "properties.identifier" });
  console.log(name);
  if (!id.isDirty && name !== "") {
    console.log("DOG");
    const newIdentifier = extrapolateIdentifier(name);
    if (newIdentifier !== identifier) setValue("properties.identifier", newIdentifier);
  }

  return (
    <Align.Center>
      <Align.Space
        direction="y"
        className={CSS.B("properties")}
        justify="center"
        align="start"
        size="large"
      >
        <Text.Text level="h1">Let's get started</Text.Text>
        <Text.Text level="p">
          Confirm the details of your device and give it a name.
        </Text.Text>
        <Align.Space direction="y" align="stretch" className={CSS.B("fields")}>
          <Input.HFItem<Vendor> name="properties.vendor" label="Vendor">
            {(props) => <SelectVendor {...props} />}
          </Input.HFItem>
          <Input.HFItem<string> name="properties.key" label="Serial Number" />
          <Input.HFItem<string> name="properties.model" label="Model">
            {(props) => <SelectModel {...props} />}
          </Input.HFItem>
          <Input.HFItem<string> name="properties.name" label="Name" />
          <Input.HFItem<string> name="properties.identifier" label="Identifier" />
        </Align.Space>
      </Align.Space>
    </Align.Center>
  );
};
