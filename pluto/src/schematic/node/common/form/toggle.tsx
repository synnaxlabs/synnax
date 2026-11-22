// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Tabs as BaseTabs } from "@synnaxlabs/lyra/tabs";
import { deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Channel } from "@/channel";
import { ActivationDelayField } from "@/schematic/node/common/form/ActivationDelay";
import { ColorValueField } from "@/schematic/node/common/form/Color";
import { ControlChipField } from "@/schematic/node/common/form/Control";
import { ScaleField } from "@/schematic/node/common/form/Scale";
import {
  StyleForm,
  type StyleFormProps,
  type StyleValues,
} from "@/schematic/node/common/form/Style";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import type { FormProps } from "@/schematic/node/spec";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";

const path = deep.path<StyleValues & { clickable?: boolean }>();

const togglePath = deep.path<schematic.ToggleConfig>();

interface ToggleChannelFormProps {
  omit?: string[];
}

export const ToggleChannelForm = ({
  omit = [],
}: ToggleChannelFormProps): ReactElement => {
  const { value, onChange } = Form.useField(togglePath);

  const handleSourceChange = (v: channel.Key | null): void =>
    onChange({ ...value, stateChannel: v ?? 0 });

  const handleSinkChange = (v: channel.Key | null): void =>
    onChange({
      ...value,
      commandChannel: v ?? 0,
    });

  return (
    <Form.Sections x>
      <Form.Section title="State">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            allowNone
            value={value.stateChannel}
            onChange={handleSourceChange}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            allowNone
            value={value.commandChannel}
            onChange={handleSinkChange}
          />
        </Input.Item>
        {!omit.includes("onClickDelay") && <ActivationDelayField />}
        <ControlChipField />
      </Form.Section>
      <Form.Section title="Staleness">
        <Staleness.TimeoutField />
      </Form.Section>
    </Form.Sections>
  );
};

export interface ToggleFormProps
  extends FormProps, Pick<StyleFormProps, "showNormallyOpen"> {
  hideInnerOrientation?: boolean;
  omit?: string[];
}

export const ToggleForm = ({
  actions,
  tab,
  onTabChange,
  hideInnerOrientation,
  showNormallyOpen,
  omit,
}: ToggleFormProps): ReactElement => (
  <Properties.Tabs
    tabs={["control", "style"]}
    tab={tab}
    onTabChange={onTabChange}
    actions={actions}
  >
    <BaseTabs.Content itemKey="style">
      <StyleForm
        hideInnerOrientation={hideInnerOrientation}
        showNormallyOpen={showNormallyOpen}
      />
    </BaseTabs.Content>
    <BaseTabs.Content itemKey="control">
      <ToggleChannelForm omit={omit} />
    </BaseTabs.Content>
  </Properties.Tabs>
);

export const DummyToggleForm = (): ReactElement => (
  <Form.Sections x>
    <Form.Section title="Label">
      <Label.Form path={path.label} />
    </Form.Section>
    <Form.Section title="Appearance">
      <ColorValueField path={path.strokeColor} label="Stroke" />
      <ScaleField path={path.scale} />
      <Form.SwitchField path={path.clickable} label="Clickable" />
    </Form.Section>
    <Orientation.Section path={path} />
  </Form.Sections>
);
