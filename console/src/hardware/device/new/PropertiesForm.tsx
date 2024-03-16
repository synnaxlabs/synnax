// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Form } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Text } from "@synnaxlabs/pluto/text";

import { CSS } from "@/css";
import { SelectModel } from "@/hardware/configure/ni/SelectModel";
import { SelectVendor } from "@/hardware/device/new/SelectVendor";
import { type Vendor } from "@/hardware/device/new/types";

import "@/hardware/device/new/PropertiesForm.css";

const MIN_IDENTIFIER_LENGTH = 3;
const MAX_IDENTIFIER_LENGTH = 5;

export const extrapolateIdentifier = (identifier: string): string => {
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
  Form.useFieldListener<string>("properties.name", (state, { set, get }) => {
    const id = get("properties.identifier");
    if (!id.touched) set("properties.identifier", extrapolateIdentifier(state.value));
  });

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
          <Form.Field<Vendor> path="properties.vendor" label="Vendor">
            {(p) => <SelectVendor {...p} />}
          </Form.Field>
          <Form.Field<string> path="properties.key" label="Serial Number" />
          <Form.Field<string> path="properties.model" label="Model">
            {(props) => <SelectModel {...props} />}
          </Form.Field>
          <Form.Field<string> path="properties.name" label="Name" />
          <Form.Field<string> path="properties.identifier" label="Identifier" />
        </Align.Space>
      </Align.Space>
    </Align.Center>
  );
};
