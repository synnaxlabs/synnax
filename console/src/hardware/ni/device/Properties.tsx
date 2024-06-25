// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/device/Properties.css";

import { Form } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { SelectModel } from "@/hardware/ni/device/enrich/SelectModel";

const MIN_IDENTIFIER_LENGTH = 4;
const MAX_IDENTIFIER_LENGTH = 7;

export const extrapolateIdentifier = (identifier: string): string => {
  const words = identifier.split(" ");
  let toGrabFromFirst = MIN_IDENTIFIER_LENGTH - words.length + 1;
  if (toGrabFromFirst < 1) toGrabFromFirst = 1;
  return words
    .map((word, i) => (i === 0 ? word.slice(0, toGrabFromFirst) : word[0]))
    .join("")
    .toLowerCase()
    .slice(0, MAX_IDENTIFIER_LENGTH);
};

export const PropertiesForm = (): ReactElement => {
  Form.useFieldListener<string>({
    path: "properties.name",
    onChange: (state, { set, get }) => {
      const id = get("properties.identifier");
      if (!id.touched) set("properties.identifier", extrapolateIdentifier(state.value));
    },
  });

  return (
    <Align.Space
      direction="x"
      className={CSS.B("properties")}
      justify="center"
      align="start"
      grow
      size={10}
    >
      <Align.Space className={CSS.B("description")} direction="y">
        <Text.Text level="h1">Let's get started</Text.Text>
        <Text.Text level="p">
          Welcome to the Synnax hardware configuration workflow. We'll walk you through
          the steps needed to set up your device for smooth operation with Synnax.
        </Text.Text>
        <Text.Text level="p">
          To start off, we've automatically detected some information about your device,
          and we'd like to confirm that it's correct.
        </Text.Text>
        <Text.Text level="p">
          We'd also like you to provide a human readable <b>name</b> for your device,
          and give it a short <b>identifier</b> that will be used to label channels and
          other resources related to it.
        </Text.Text>
        <Text.Text level="p">
          If the model for your device doesn't show up, or the information we've
          detected is incorrect, please let us know.
        </Text.Text>
      </Align.Space>
      <Align.Space grow direction="y" align="stretch" className={CSS.B("form")}>
        <Form.Field<string> path="properties.key" label="Serial Number" />
        <Form.Field<string> path="properties.model" label="Model">
          {(props) => <SelectModel {...props} />}
        </Form.Field>
        <Form.Field<string> path="properties.name" label="Name" />
        <Form.Field<string> path="properties.identifier" label="Identifier" />
      </Align.Space>
    </Align.Space>
  );
};
