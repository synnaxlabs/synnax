import { useEffect, type ReactElement } from "react";

import { type Button } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { type useForm } from "react-hook-form";

import { CSS } from "@/css";

import { SelectVendor } from "./SelectVendor";
import { type Configuration, type Properties, type Vendor } from "./types";

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

export interface PropertiesProps
  extends Pick<
    ReturnType<typeof useForm<Configuration>>,
    "setValue" | "control" | "watch" | "trigger"
  > {}

export const PropertiesForm = ({
  control: c,
  setValue,
  watch,
}: PropertiesProps): ReactElement => {
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
        <Input.ItemControlled<Vendor> control={c} name="vendor" label="Vendor">
          {(props) => <SelectVendor {...props} />}
        </Input.ItemControlled>
        <Input.ItemControlled<string> control={c} name="key" label="Serial Number" />
        <Input.ItemControlled<string> control={c} name="model" label="Model" />
        <Input.ItemControlled<string> control={c} name="name" label="Name" />
        <Input.ItemControlled<string>
          control={c}
          name="identifier"
          label="Identifier"
        />
      </Align.Space>
    </Align.Space>
  );
};
