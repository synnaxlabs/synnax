// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { useController, useForm } from "react-hook-form";
import { z } from "zod";

import { PIDElementSpec, PIDElementFormProps } from "../pid/PIDElements";

import { valueCoreProps } from "./ValueCore";

import { Telem } from "@/telem";

export const coreValuePIDElementProps = valueCoreProps.extend({
  label: z.string().optional(),
});

export type ValuePIDElementProps = z.infer<typeof coreValuePIDElementProps>;

export const ValuePIDElementForm = ({
  value,
  onChange,
}: PIDElementFormProps<ValuePIDElementProps>): ReactElement => {
  const { handleSubmit, watch, getValues } = useForm<ValuePIDElementProps>({
    resolver: zodResolver(coreValuePIDElementProps),
  });
};

export const ValuePIDElementPreview = (): ReactElement => {
  const telem = Telem.Static.usePoint(12.52);
  return <ValuePIDElement label="dog" telem={telem} />;
};

export const ValuePIDElementSpec: PIDElementSpec = {
  title: "Value",
  form: ValuePIDElementForm,
  schema: coreValuePIDElementProps,
  element: ValuePIDElement,
};
