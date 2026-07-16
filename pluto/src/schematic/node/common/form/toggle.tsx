// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Form } from "@/form";
import { ColorField } from "@/schematic/node/common/form/Color";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import { StyleForm } from "@/schematic/node/common/form/Style";
import { Wrapper } from "@/schematic/node/common/form/Wrapper";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Toggle } from "@/schematic/node/common/toggle";
import { type FormProps } from "@/schematic/node/spec";
import { Tabs } from "@/tabs";

interface ToggleFormProps extends FormProps {
  hideInnerOrientation?: boolean;
  omit?: string[];
}

export const ToggleForm = ({
  actions,
  hideInnerOrientation,
  omit,
}: ToggleFormProps): ReactElement => (
  <Tabs.Frame initialValue="style">
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="control">Control</Tabs.Tab>
      {actions != null && (
        <>
          <Flex.Box grow />
          <Flex.Box x align="center" empty>
            {actions}
          </Flex.Box>
        </>
      )}
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <StyleForm hideInnerOrientation={hideInnerOrientation} />
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <Toggle.ChannelForm path="" omit={omit} />
    </Tabs.Content>
  </Tabs.Frame>
);

export const DummyToggleForm = (): ReactElement => (
  <Wrapper x align="stretch">
    <Flex.Box y grow>
      <Label.Form path="label" />
      <Flex.Box x grow>
        <ColorField path="color" />
        <ScaleField path="scale" />
        <Form.SwitchField path="clickable" label="Clickable" hideIfNull optional />
      </Flex.Box>
    </Flex.Box>
    <Orientation.Field path="" />
  </Wrapper>
);
