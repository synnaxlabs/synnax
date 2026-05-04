// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import { StateOverrideControls } from "@/schematic/node/common/custom/Custom";
import {
  ColorControl,
  COMMON_TOGGLE_FORM_TABS,
  FormWrapper,
  LabelControls,
  type NodeFormProps,
  OrientationControl,
  ScaleControl,
  ToggleControlForm,
} from "@/schematic/node/common/forms/helpers";
import { Tabs } from "@/tabs";

interface CommonStyleFormProps extends NodeFormProps {
  omit?: string[];
  hideInnerOrientation?: boolean;
  hideOuterOrientation?: boolean;
  showStateOverrides?: boolean;
}

export const CommonStyleForm = ({
  omit,
  hideInnerOrientation,
  hideOuterOrientation,
}: CommonStyleFormProps): ReactElement => {
  const hasStateOverrides =
    Form.useFieldValue<string>("stateOverrides", {
      optional: true,
    }) != null;
  return (
    <FormWrapper x align="stretch" empty>
      <Flex.Box y grow>
        <LabelControls omit={omit} path="label" />
        <Flex.Box x grow>
          {!hasStateOverrides && <ColorControl path="color" optional />}
          <Form.Field<boolean>
            path="normallyOpen"
            label="Normally Open"
            padHelpText={false}
            hideIfNull
            optional
          >
            {(p) => <Input.Switch {...p} />}
          </Form.Field>
          <ScaleControl path="scale" />
        </Flex.Box>
      </Flex.Box>
      {hasStateOverrides && <StateOverrideControls />}
      <OrientationControl
        path=""
        hideInner={hideInnerOrientation}
        hideOuter={hideOuterOrientation}
      />
    </FormWrapper>
  );
};

interface CommonToggleFormProps extends NodeFormProps {
  hideInnerOrientation?: boolean;
  omit?: string[];
}

export const CommonToggleForm = ({
  actions,
  hideInnerOrientation,
  omit,
}: CommonToggleFormProps): ReactElement => {
  const content: Tabs.RenderProp = useCallback(
    ({ tabKey }) => {
      switch (tabKey) {
        case "control":
          return <ToggleControlForm path="" omit={omit} />;
        default:
          return <CommonStyleForm hideInnerOrientation={hideInnerOrientation} />;
      }
    },
    [hideInnerOrientation, omit],
  );
  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });
  return <Tabs.Tabs {...props} actions={actions} />;
};

export const CommonDummyToggleForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Flex.Box y grow>
      <LabelControls path="label" />
      <Flex.Box x grow>
        <ColorControl path="color" />
        <ScaleControl path="scale" />
        <Form.SwitchField path="clickable" label="Clickable" hideIfNull optional />
      </Flex.Box>
    </Flex.Box>
    <OrientationControl path="" />
  </FormWrapper>
);
