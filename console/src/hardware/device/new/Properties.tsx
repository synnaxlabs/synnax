import { useEffect, type ReactElement, useCallback } from "react";

import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { type useForm } from "react-hook-form";
import { z } from "zod";

import { CSS } from "@/css";

import { SelectVendor, type Vendor, vendorsZ } from "./SelectVendor";

export type DeviceProperties = z.infer<typeof propertiesZ>;

export const propertiesZ = z.object({
  vendor: vendorsZ,
  name: z.string(),
  model: z.string(),
  key: z.string(),
  identifier: z
    .string()
    .max(6)
    .refine((s) => !s.includes(" ") && /^[a-zA-Z0-9]+$/.test(s), {
      message: "Only alphanumeric characters allowed",
    }),
});

export const defaultProperties: DeviceProperties = {
  vendor: "ni",
  name: "",
  model: "NI USB 6000",
  key: "01A100CE",
  identifier: "",
};

const TARGET_IDENTIFIER_LENGTH = 3;

const extrapolateIdentifier = (identifier: string): string => {
  const words = identifier.split(" ");
  let toGrabFromFirst = TARGET_IDENTIFIER_LENGTH - words.length + 1;
  if (toGrabFromFirst < 1) toGrabFromFirst = 1;
  return words
    .map((word, i) => (i === 0 ? word.slice(0, toGrabFromFirst) : word[0]))
    .join("");
};

export interface PropertiesProps
  extends Pick<
    ReturnType<typeof useForm<DeviceProperties>>,
    "setValue" | "control" | "watch"
  > {}

export const Properties = ({
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
      align="stretch"
      justify="center"
    >
      <Text.Text level="h1">Let's configure your hardware</Text.Text>
      <Text.Text level="p">
        Confirm the details of your device and give it a name.
      </Text.Text>
      <Input.ItemControlled<Vendor> control={c} name="vendor" label="Vendor">
        {(props) => <SelectVendor {...props} />}
      </Input.ItemControlled>
      <Input.ItemControlled<string> control={c} name="key" label="Serial Number" />
      <Input.ItemControlled<string> control={c} name="model" label="Model" />
      <Input.ItemControlled<string> control={c} name="name" label="Name" />
      <Input.ItemControlled<string> control={c} name="identifier" label="Identifier" />
    </Align.Space>
  );
};
