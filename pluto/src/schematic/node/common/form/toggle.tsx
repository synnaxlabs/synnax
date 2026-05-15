// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { type ReactElement, useCallback } from "react";

import { Form as Base } from "@synnaxlabs/lyra/form";
import { ColorField } from "@/schematic/node/common/form/Color";
import { COMMON_TOGGLE_FORM_TABS } from "@/schematic/node/common/form/input";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import { StyleForm } from "@/schematic/node/common/form/Style";
import { Wrapper } from "@/schematic/node/common/form/Wrapper";
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
}: ToggleFormProps): ReactElement => {
  const content: Tabs.RenderProp = useCallback(
    ({ tabKey }) => {
      switch (tabKey) {
        case "control":
          return <Toggle.ChannelForm path="" omit={omit} />;
        default:
          return <StyleForm hideInnerOrientation={hideInnerOrientation} />;
      }
    },
    [hideInnerOrientation, omit],
  );
  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });
  return <Tabs.Tabs {...props} actions={actions} />;
};

export const DummyToggleForm = (): ReactElement => (
  <Wrapper x align="stretch">
    <Flex.Box y grow>
      <Label.Form path="label" />
      <Flex.Box x grow>
        <ColorField path="color" />
        <ScaleField path="scale" />
        <Base.SwitchField path="clickable" label="Clickable" hideIfNull optional />
      </Flex.Box>
    </Flex.Box>
    <Orientation.Field path="" />
  </Wrapper>
);
