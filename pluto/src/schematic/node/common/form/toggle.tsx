// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Tabs as BaseTabs } from "@synnaxlabs/lyra/tabs";
import { type ReactElement } from "react";

import { ColorField } from "@/schematic/node/common/form/Color";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import { StyleForm } from "@/schematic/node/common/form/Style";
import { Tabs } from "@/schematic/node/common/form/Tabs";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Toggle } from "@/schematic/node/common/toggle";
import { type FormProps } from "@/schematic/node/spec";

interface ToggleFormProps extends FormProps {
  hideInnerOrientation?: boolean;
  omit?: string[];
}

export const ToggleForm = ({
  actions,
  hideInnerOrientation,
  omit,
}: ToggleFormProps): ReactElement => (
  <Tabs tabs={["style", "control"]} actions={actions}>
    <BaseTabs.Content itemKey="style">
      <StyleForm hideInnerOrientation={hideInnerOrientation} />
    </BaseTabs.Content>
    <BaseTabs.Content itemKey="control">
      <Toggle.ChannelForm path="" omit={omit} />
    </BaseTabs.Content>
  </Tabs>
);

export const DummyToggleForm = (): ReactElement => (
  <Form.Sections x>
    <Form.Section title="Label">
      <Label.Form path="label" />
    </Form.Section>
    <Form.Section title="Appearance">
      <ColorField path="color" />
      <ScaleField path="scale" />
      <Form.SwitchField path="clickable" label="Clickable" hideIfNull optional />
    </Form.Section>
    <Orientation.Section path="" />
  </Form.Sections>
);
