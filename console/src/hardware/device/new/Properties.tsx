import { useEffect, type ReactElement } from "react";

import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { useFormContext } from "react-hook-form";

import { CSS } from "@/css";

import { SelectVendor } from "./SelectVendor";
import { type Vendor } from "./types";

import "@/hardware/device/new/Properties.css";

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
  const { watch, setValue, control: c } = useFormContext();

  useEffect(
    () =>
      watch((value, { name }) => {
        if (
          name !== "name" ||
          value.name == null ||
          c.getFieldState("identifier")?.isTouched
        )
          return;
        setValue("identifier", extrapolateIdentifier(value.name));
      }).unsubscribe,
    [watch, setValue],
  );

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
          <Input.HFItem<Vendor> control={c} name="vendor" label="Vendor">
            {(props) => <SelectVendor {...props} />}
          </Input.HFItem>
          <Input.HFItem<string> control={c} name="key" label="Serial Number" />
          <Input.HFItem<string> control={c} name="model" label="Model" />
          <Input.HFItem<string> control={c} name="name" label="Name" />
          <Input.HFItem<string> control={c} name="identifier" label="Identifier" />
        </Align.Space>
      </Align.Space>
    </Align.Center>
  );
};
