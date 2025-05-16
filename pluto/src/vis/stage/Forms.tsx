// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/stage/Forms.css";

import { type channel } from "@synnaxlabs/client";
import { color, type direction, type location } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { Align } from "@/align";
import { Channel } from "@/channel";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Form } from "@/form";
import { Input } from "@/input";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Text } from "@/text";
import { type Button as CoreButton } from "@/vis/button";
import { SelectOrientation } from "@/vis/stage/SelectOrientation";
import { type ControlStateProps, type LabelExtensionProps } from "@/vis/stage/Symbols";

export interface SymbolFormProps {}

interface FormWrapperProps extends Align.SpaceProps {}

const FormWrapper: FC<FormWrapperProps> = ({
  className,
  direction,
  ...rest
}): ReactElement => (
  <Align.Space
    direction={direction}
    align="stretch"
    className={CSS(CSS.B("symbol-form"), className)}
    size={direction === "x" ? "large" : "medium"}
    {...rest}
  />
);

interface SymbolOrientation {
  label?: LabelExtensionProps;
  orientation?: location.Outer;
}

interface ShowOrientationProps {
  hideOuter?: boolean;
  hideInner?: boolean;
}

const OrientationControl = ({
  hideOuter,
  hideInner,
  ...rest
}: Form.FieldProps<SymbolOrientation> & ShowOrientationProps): ReactElement | null => {
  if (hideInner && hideOuter) return null;
  return (
    <Form.Field<SymbolOrientation> label="Orientation" padHelpText={false} {...rest}>
      {({ value, onChange }) => (
        <SelectOrientation
          value={{
            inner: value.orientation ?? "top",
            outer: value.label?.orientation ?? "top",
          }}
          hideInner={hideInner}
          hideOuter={hideOuter}
          onChange={(v) =>
            onChange({
              ...value,
              orientation: v.inner,
              label: { ...value.label, orientation: v.outer },
            })
          }
        />
      )}
    </Form.Field>
  );
};

interface LabelControlsProps {
  path: string;
  omit?: string[];
}

const LabelControls = ({ path, omit = [] }: LabelControlsProps): ReactElement => (
  <Align.Space x align="stretch">
    <Form.Field<string> path={`${path}.label`} label="Label" padHelpText={false} grow>
      {(p) => <Input.Text selectOnFocus {...p} />}
    </Form.Field>
    <Form.NumericField
      visible={!omit.includes("maxInlineSize")}
      style={{ maxWidth: 125 }}
      path={`${path}.maxInlineSize`}
      hideIfNull
      label="Label Wrap Width"
      inputProps={{ endContent: "px", dragScale: { x: 1, y: 0.5 } }}
      padHelpText={false}
    />
    <Form.Field<Text.Level>
      hideIfNull
      visible={!omit.includes("level")}
      path={`${path}.level`}
      label="Label Size"
      padHelpText={false}
    >
      {(p) => <Select.Text.Level {...p} />}
    </Form.Field>
    <Form.Field<Align.Alignment>
      visible={!omit.includes("align")}
      path={`${path}.align`}
      label="Label Alignment"
      padHelpText={false}
      hideIfNull
    >
      {(p) => <Select.TextAlignment {...p} />}
    </Form.Field>
    <Form.Field<direction.Direction>
      visible={!omit.includes("direction")}
      path={`${path}.direction`}
      label="Label Direction"
      padHelpText={false}
      hideIfNull
    >
      {(p) => <Select.Direction {...p} yDirection="down" />}
    </Form.Field>
  </Align.Space>
);

const ColorControl: Form.FieldT<color.Crude> = (props): ReactElement => (
  <Form.Field hideIfNull label="Color" align="start" padHelpText={false} {...props}>
    {({ value, onChange, variant: _, ...rest }) => (
      <Color.Swatch
        value={value ?? color.setAlpha(color.ZERO, 1)}
        onChange={onChange}
        {...rest}
        bordered
      />
    )}
  </Form.Field>
);

const ScaleControl: Form.FieldT<number> = (props): ReactElement => (
  <Form.Field hideIfNull label="Scale" align="start" padHelpText={false} {...props}>
    {({ value, onChange }) => (
      <Input.Numeric
        dragScale={{
          x: 0.75,
          y: 0.5,
        }}
        bounds={{ lower: 50, upper: 1000 }}
        endContent="%"
        value={Math.round(value * 100)}
        onChange={(v) => onChange(parseFloat((v / 100).toFixed(2)))}
      />
    )}
  </Form.Field>
);

interface CommonStyleFormProps {
  omit?: string[];
  hideInnerOrientation?: boolean;
  hideOuterOrientation?: boolean;
}

export const CommonStyleForm = ({
  omit,
  hideInnerOrientation,
  hideOuterOrientation,
}: CommonStyleFormProps): ReactElement => (
  <FormWrapper x align="stretch">
    <Align.Space y grow>
      <LabelControls omit={omit} path="label" />
      <Align.Space x grow>
        <ColorControl path="color" optional />
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
      </Align.Space>
    </Align.Space>
    <OrientationControl
      path=""
      hideInner={hideInnerOrientation}
      hideOuter={hideOuterOrientation}
    />
  </FormWrapper>
);

type ButtonTelemFormT = Omit<CoreButton.UseProps, "aetherKey"> & {
  control: ControlStateProps;
};

export const ButtonTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<ButtonTelemFormT>({ path });
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSinkChange = (v: channel.Key): void => {
    v ??= 0;
    const t = telem.sinkPipeline("boolean", {
      connections: [{ from: "setpoint", to: "setter" }],
      segments: {
        setter: control.setChannelValue({ channel: v }),
        setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
      },
      inlet: "setpoint",
    });

    const authSource = control.authoritySource({ channel: v });

    const controlChipSink = control.acquireChannelControl({
      channel: v,
      authority: 255,
    });

    onChange({
      ...value,
      sink: t,
      control: {
        ...value.control,
        showChip: true,
        chip: { sink: controlChipSink, source: authSource },
        showIndicator: true,
        indicator: { statusSource: authSource },
      },
    });
  };

  return (
    <FormWrapper x>
      <Input.Item label="Output Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
      <Form.NumericField
        label="Activation Delay"
        path="onClickDelay"
        inputProps={{ endContent: "ms" }}
        hideIfNull
      />
      <Form.SwitchField
        path="control.show"
        label="Show Control Chip"
        hideIfNull
        optional
      />
    </FormWrapper>
  );
};

const COMMON_TOGGLE_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "control", name: "Control" },
];

export const ButtonForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <ButtonTelemForm path="" />;
      default:
        return (
          <CommonStyleForm
            omit={["align", "maxInlineSize"]}
            hideInnerOrientation
            hideOuterOrientation
          />
        );
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};
