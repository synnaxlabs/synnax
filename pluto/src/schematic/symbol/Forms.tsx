// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/symbol/Forms.css";

import { type channel } from "@synnaxlabs/client";
import {
  type bounds,
  color,
  type direction,
  type location,
  type xy,
} from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { Button } from "@/button";
import { Channel } from "@/channel";
import { Color } from "@/color";
import { Component } from "@/component";
import { CSS } from "@/css";
import { Direction } from "@/direction";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { StateOverrideControls } from "@/schematic/symbol/Custom";
import { SelectOrientation } from "@/schematic/symbol/SelectOrientation";
import {
  type ControlStateProps,
  type LabelExtensionProps,
} from "@/schematic/symbol/Symbols";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Text } from "@/text";
import { Button as CoreButton } from "@/vis/button";
import { type Input as CoreInput } from "@/vis/input";
import { type Setpoint } from "@/vis/setpoint";
import { type Toggle } from "@/vis/toggle";
import { Value } from "@/vis/value";

export interface SymbolFormProps extends Pick<Tabs.TabsProps, "actions"> {}

interface FormWrapperProps extends Flex.BoxProps {}

const FormWrapper: FC<FormWrapperProps> = ({
  className,
  direction,
  ...rest
}): ReactElement => (
  <Flex.Box
    direction={direction}
    align="stretch"
    className={CSS(CSS.B("symbol-form"), className)}
    gap={direction === "x" ? "large" : "medium"}
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
  showOuterCenter?: boolean;
}

const OrientationControl = ({
  hideOuter,
  hideInner,
  showOuterCenter,
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
          showOuterCenter={showOuterCenter}
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
  <Flex.Box x align="stretch">
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
      {({ value, onChange }) => <Select.Text.Level value={value} onChange={onChange} />}
    </Form.Field>
    <Form.Field<Flex.Alignment>
      visible={!omit.includes("align")}
      path={`${path}.align`}
      label="Label Alignment"
      padHelpText={false}
      hideIfNull
    >
      {({ value, onChange }) => (
        <Select.Flex.Alignment value={value} onChange={onChange} />
      )}
    </Form.Field>
    <Form.Field<direction.Direction>
      visible={!omit.includes("direction")}
      path={`${path}.direction`}
      label="Label Direction"
      padHelpText={false}
      hideIfNull
    >
      {({ value, onChange }) => (
        <Direction.Select value={value} onChange={onChange} yDirection="down" />
      )}
    </Form.Field>
  </Flex.Box>
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

const SCALE_CONTROL_BOUNDS: bounds.Bounds = { lower: 5, upper: 1000 };
const SCALE_CONTROL_DRAG_SCALE: xy.Crude = { x: 0.75, y: 0.5 };

const ScaleControl: Form.FieldT<number> = (props): ReactElement => (
  <Form.Field hideIfNull label="Scale" align="start" padHelpText={false} {...props}>
    {({ value, onChange }) => (
      <Input.Numeric
        dragScale={SCALE_CONTROL_DRAG_SCALE}
        bounds={SCALE_CONTROL_BOUNDS}
        endContent="%"
        value={Math.round(value * 100)}
        onChange={(v) => onChange(parseFloat((v / 100).toFixed(2)))}
      />
    )}
  </Form.Field>
);
interface CommonStyleFormProps extends SymbolFormProps {
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

const ToggleControlForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<
    Omit<Toggle.UseProps, "aetherKey"> & { control: ControlStateProps }
  >(path);
  const sourceP = telem.sourcePipelinePropsZ.parse(value.source?.props);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSourceChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sourcePipeline("boolean", {
      connections: [{ from: "valueStream", to: "threshold" }],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
        threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };

  const handleSinkChange = (v: channel.Key | null): void => {
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
        showChip: true,
        showIndicator: true,
        ...value.control,
        chip: { sink: controlChipSink, source: authSource },
        indicator: { statusSource: authSource },
      },
    });
  };

  return (
    <FormWrapper x grow align="stretch">
      <Input.Item label="State Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
        />
      </Input.Item>
      <Input.Item label="Command Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
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

interface CommonToggleFormProps extends SymbolFormProps {
  hideInnerOrientation?: boolean;
}

export const CommonToggleForm = ({
  actions,
  hideInnerOrientation,
}: CommonToggleFormProps): ReactElement => {
  const content: Tabs.RenderProp = useCallback(
    ({ tabKey }) => {
      switch (tabKey) {
        case "control":
          return <ToggleControlForm path="" />;
        default:
          return <CommonStyleForm hideInnerOrientation={hideInnerOrientation} />;
      }
    },
    [hideInnerOrientation],
  );
  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });
  return <Tabs.Tabs {...props} actions={actions} />;
};

const DIMENSIONS_DRAG_SCALE: xy.Crude = { y: 2, x: 0.25 };
const DIMENSIONS_BOUNDS: bounds.Bounds = { lower: 0, upper: 2000 };
const BORDER_RADIUS_BOUNDS: bounds.Bounds = { lower: 0, upper: 51 };

export interface TankFormProps extends SymbolFormProps {
  includeBorderRadius?: boolean;
  includeStrokeWidth?: boolean;
}

export const TankForm = ({
  includeBorderRadius = false,
  includeStrokeWidth = false,
}: TankFormProps): ReactElement => (
  <FormWrapper x align="stretch">
    <Flex.Box y grow>
      <LabelControls path="label" />
      <Flex.Box x>
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.Field<number>
          path="borderRadius.x"
          hideIfNull
          optional
          label="X Border Radius"
          grow
        >
          {(props) => (
            <Input.Numeric
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={BORDER_RADIUS_BOUNDS}
              endContent="%"
              {...props}
            />
          )}
        </Form.Field>
        <Form.Field<number>
          path="borderRadius.y"
          hideIfNull
          optional
          label="Y Border Radius"
          grow
        >
          {(props) => (
            <Input.Numeric
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={BORDER_RADIUS_BOUNDS}
              endContent="%"
              {...props}
            />
          )}
        </Form.Field>
        {includeBorderRadius && (
          <Form.Field<number>
            path="borderRadius"
            hideIfNull
            optional
            label="Border Radius"
            grow
          >
            {(props) => (
              <Input.Numeric
                dragScale={DIMENSIONS_DRAG_SCALE}
                bounds={DIMENSIONS_BOUNDS}
                endContent="px"
                {...props}
              />
            )}
          </Form.Field>
        )}
        {includeStrokeWidth && (
          <Form.Field<number>
            path="strokeWidth"
            hideIfNull
            optional
            label="Border Width"
            grow
          >
            {(props) => (
              <Input.Numeric
                dragScale={DIMENSIONS_DRAG_SCALE}
                bounds={{ lower: 0, upper: 21 }}
                endContent="px"
                {...props}
              />
            )}
          </Form.Field>
        )}
        <Form.Field<number> path="dimensions.width" label="Width" grow>
          {({ value, ...rest }) => (
            <Input.Numeric
              value={value ?? 200}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
              endContent="px"
              {...rest}
            />
          )}
        </Form.Field>
        <Form.Field<number> path="dimensions.height" label="Height" grow>
          {({ value, ...rest }) => (
            <Input.Numeric
              value={value ?? 200}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
              endContent="px"
              {...rest}
            />
          )}
        </Form.Field>
      </Flex.Box>
    </Flex.Box>
    <OrientationControl path="" hideInner showOuterCenter label="Label Location" />
  </FormWrapper>
);

export interface PolygonFormProps {
  numSides: number;
}

export const CommonPolygonForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Flex.Box direction="y" grow>
      <LabelControls path="label" />
      <Flex.Box direction="x">
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.NumericField
          path="rotation"
          label="Rotation"
          inputProps={{
            dragScale: { x: 1, y: 0.25 },
            bounds: { lower: 0, upper: 360 },
            endContent: "°",
          }}
          grow
        />
        <Form.NumericField
          path="numSides"
          label="Number of Sides"
          inputProps={{
            dragScale: { x: 0.5, y: 0.1 },
            bounds: { lower: 3, upper: 21 },
          }}
          grow
        />
        <Form.NumericField
          path="sideLength"
          label="Side Length"
          inputProps={{
            dragScale: { x: 1, y: 1 },
            bounds: { lower: 10, upper: 500 },
            endContent: "px",
          }}
          grow
        />
        <Form.NumericField
          path="cornerRounding"
          label="Corner Rounding"
          inputProps={{
            dragScale: { x: 1, y: 1 },
            bounds: { lower: 0, upper: 181 }, // internally limited as well to ensure weird things don't happen
            endContent: "px",
          }}
          grow
        />
        <Form.NumericField
          path="strokeWidth"
          label="Border Width"
          inputProps={{
            dragScale: { x: 1, y: 1 },
            bounds: { lower: 1, upper: 21 },
            endContent: "px",
          }}
          grow
        />
      </Flex.Box>
    </Flex.Box>
  </FormWrapper>
);

export const CircleForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Flex.Box direction="y" grow>
      <LabelControls path="label" />
      <Flex.Box direction="x">
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.NumericField
          path="radius"
          label="Radius"
          inputProps={{
            dragScale: { x: 1, y: 1 },
            bounds: { lower: 0, upper: 500 },
            endContent: "px",
          }}
          grow
        />
        <Form.NumericField
          path="strokeWidth"
          label="Border Width"
          inputProps={{
            dragScale: { x: 1, y: 1 },
            bounds: { lower: 1, upper: 21 },
            endContent: "px",
          }}
          grow
        />
      </Flex.Box>
    </Flex.Box>
  </FormWrapper>
);

const VALUE_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telemetry", name: "Telemetry" },
  { tabKey: "redline", name: "Redline" },
];

const valueWidthInputProps = {
  dragScale: { x: 1, y: 0.25 },
  bounds: { lower: 40, upper: 500 },
  endContent: "px",
} as const;

export const ValueForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <FormWrapper y empty>
            <Value.TelemForm path="" />;
          </FormWrapper>
        );
      case "redline":
        return (
          <FormWrapper y empty>
            <Value.RedlineForm path="redline" />
          </FormWrapper>
        );
      default:
        return (
          <FormWrapper x>
            <Flex.Box y grow>
              <LabelControls path="label" />
              <Flex.Box x>
                <ColorControl path="color" />
                <Form.Field<string>
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                >
                  {(p) => <Input.Text {...p} />}
                </Form.Field>
                <Form.NumericField
                  path="inlineSize"
                  label="Value Width"
                  hideIfNull
                  inputProps={valueWidthInputProps}
                />
                <Form.Field<Text.Level>
                  path="level"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Select.Text.Level value={value} onChange={onChange} />
                  )}
                </Form.Field>
              </Flex.Box>
            </Flex.Box>
            <OrientationControl path="" hideInner />
          </FormWrapper>
        );
    }
  }, []);
  const props = Tabs.useStatic({ tabs: VALUE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};

const GAUGE_BAR_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  min: 1,
  max: 50,
  step: 1,
  bounds: { lower: 1, upper: 50 },
  endContent: "px",
  dragScale: { x: 0.1, y: 0.1 },
};

const MIN_VALUE_INPUT_PROPS: Partial<Input.NumericProps> = {
  step: 10,
};

const MAX_VALUE_INPUT_PROPS: Partial<Input.NumericProps> = {
  step: 10,
};

const handleLevelChange = (v: Text.Level, { set }: Form.ContextValue): void => {
  if (v === "small") set("barWidth", 4);
  else if (v === "h5") set("barWidth", 8);
  else set("barWidth", 10);
};

export const GaugeForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <FormWrapper y empty>
            <Value.TelemForm path="" />
          </FormWrapper>
        );
      default:
        return (
          <FormWrapper x>
            <Flex.Box y grow>
              <LabelControls path="label" />
              <Flex.Box x>
                <ColorControl path="color" />
                <Form.TextField
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                />
                <Form.NumericField
                  path="bounds.lower"
                  label="Min Value"
                  hideIfNull
                  inputProps={MIN_VALUE_INPUT_PROPS}
                />
                <Form.NumericField
                  path="bounds.upper"
                  label="Max Value"
                  hideIfNull
                  inputProps={MAX_VALUE_INPUT_PROPS}
                />
                <Form.NumericField
                  path="barWidth"
                  label="Bar Width"
                  hideIfNull
                  inputProps={GAUGE_BAR_WIDTH_INPUT_PROPS}
                />
                <Form.Field<Text.Level>
                  path="level"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                  onChange={handleLevelChange}
                >
                  {({ value, onChange }) => (
                    <Select.Text.Level value={value} onChange={onChange} />
                  )}
                </Form.Field>
              </Flex.Box>
            </Flex.Box>
          </FormWrapper>
        );
    }
  }, []);
  const tabs: Tabs.Spec[] = [
    { tabKey: "properties", name: "Properties" },
    { tabKey: "telemetry", name: "Telemetry" },
  ];
  const props = Tabs.useStatic({ tabs, content });
  return <Tabs.Tabs {...props} />;
};

interface LightTelemFormT extends Omit<Toggle.UseProps, "aetherKey"> {}

const LightTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<LightTelemFormT>(path);
  const sourceP = telem.sourcePipelinePropsZ.parse(value.source?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const threshold = telem.withinBoundsProps.parse(sourceP.segments.threshold.props);

  const handleSourceChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sourcePipeline("boolean", {
      connections: [{ from: "valueStream", to: "threshold" }],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
        threshold: telem.withinBounds({ trueBound: threshold.trueBound }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };

  const handleThresholdChange = (bounds: { lower: number; upper: number }): void => {
    const t = telem.sourcePipeline("boolean", {
      connections: [{ from: "valueStream", to: "threshold" }],
      segments: {
        valueStream: telem.streamChannelValue({ channel: source.channel }),
        threshold: telem.withinBounds({ trueBound: bounds }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };
  if (typeof source.channel !== "number")
    throw new Error("Channel key must be used for light telemetry");

  return (
    <FormWrapper x align="stretch">
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle value={source.channel} onChange={handleSourceChange} />
      </Input.Item>
      <Input.Item label="Lower Threshold">
        <Input.Numeric
          value={threshold.trueBound.lower ?? 0.9}
          onChange={(v) => handleThresholdChange({ ...threshold.trueBound, lower: v })}
        />
      </Input.Item>
      <Input.Item label="Upper Threshold">
        <Input.Numeric
          value={threshold.trueBound.upper ?? 1.1}
          onChange={(v) => handleThresholdChange({ ...threshold.trueBound, upper: v })}
        />
      </Input.Item>
    </FormWrapper>
  );
};

const LIGHT_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telemetry", name: "Telemetry" },
];

export const LightForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return <LightTelemForm path="" />;
      default:
        return <CommonStyleForm />;
    }
  }, []);
  const props = Tabs.useStatic({ tabs: LIGHT_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};

type ButtonTelemFormT = Omit<CoreButton.UseProps, "aetherKey"> & {
  control: ControlStateProps;
};

export const ButtonTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<ButtonTelemFormT>(path);
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
    <FormWrapper y empty>
      <Flex.Box x>
        <Input.Item label="Output Channel" grow padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
        <Form.NumericField
          label="Activation Delay"
          path="onClickDelay"
          inputProps={{ endContent: "ms" }}
          hideIfNull
          padHelpText={false}
        />
        <Form.SwitchField
          path="control.show"
          label="Show Control Chip"
          hideIfNull
          optional
          padHelpText={false}
        />
      </Flex.Box>
      <Form.Field<CoreButton.Mode> path="mode" label="Mode" optional>
        {({ value, onChange }) => (
          <CoreButton.SelectMode value={value} onChange={onChange} />
        )}
      </Form.Field>
    </FormWrapper>
  );
};

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

export const SetpointTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<
    Omit<Setpoint.UseProps, "aetherKey"> & {
      control: ControlStateProps;
      disabled?: boolean;
    }
  >(path);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSinkChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sinkPipeline("number", {
      connections: [],
      segments: { setter: control.setChannelValue({ channel: v }) },
      inlet: "setter",
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
      disabled: v == 0,
    });
  };

  return (
    <FormWrapper x grow align="stretch">
      <Input.Item label="Command Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
    </FormWrapper>
  );
};

export const SetpointForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <SetpointTelemForm path="" />;
      default:
        return (
          <FormWrapper x align="stretch">
            <Flex.Box y align="stretch" grow gap="small">
              <LabelControls path="label" />
              <Flex.Box x>
                <Form.TextField
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                />
                <Form.Field<Component.Size>
                  path="size"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Component.SelectSize value={value} onChange={onChange} />
                  )}
                </Form.Field>
                <ColorControl path="color" />
              </Flex.Box>
            </Flex.Box>
            <OrientationControl path="" hideInner />
          </FormWrapper>
        );
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};

interface InputTelemFormProps {
  path: string;
}

const InputTelemForm = ({ path }: InputTelemFormProps): ReactElement => {
  const { value, onChange } = Form.useField<
    Omit<CoreInput.UseProps, "aetherKey"> & {
      control: ControlStateProps;
      disabled?: boolean;
    }
  >(path);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSinkChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sinkPipeline("string", {
      connections: [],
      segments: { setter: control.setChannelValue({ channel: v }) },
      inlet: "setter",
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
      disabled: v == 0,
    });
  };

  return (
    <FormWrapper x grow align="stretch">
      <Input.Item label="Command Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
    </FormWrapper>
  );
};

export const InputForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <InputTelemForm path="" />;
      default:
        return (
          <FormWrapper x>
            <Flex.Box y align="stretch" grow gap="small">
              <LabelControls path="label" />
              <Flex.Box x>
                <Form.Field<Component.Size>
                  path="size"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Component.SelectSize value={value} onChange={onChange} />
                  )}
                </Form.Field>
                <ColorControl path="color" />
              </Flex.Box>
            </Flex.Box>
          </FormWrapper>
        );
    }
  }, []);
  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};

export const TextBoxForm = (): ReactElement => {
  const autoFit = Form.useField<boolean>("autoFit", { optional: true });
  return (
    <FormWrapper x align="stretch" grow>
      <Flex.Box y grow>
        <Flex.Box x align="stretch">
          <Form.Field<string> path="value" label="Text" padHelpText={false} grow>
            {(p) => <Input.Text {...p} />}
          </Form.Field>
          <Form.Field<Text.Level> path="level" label="Text Size" padHelpText={false}>
            {({ value, onChange }) => (
              <Select.Text.Level value={value} onChange={onChange} />
            )}
          </Form.Field>
          <Form.Field<Flex.Alignment>
            path="align"
            label="Alignment"
            padHelpText={false}
            hideIfNull
          >
            {({ value, onChange }) => (
              <Select.Flex.Alignment value={value} onChange={onChange} />
            )}
          </Form.Field>
        </Flex.Box>
        <Flex.Box x>
          <ColorControl path="color" />
          <Form.Field<number>
            onChange={(_, { set }) => set("autoFit", false)}
            path="width"
            label="Wrap Width"
            padHelpText={false}
          >
            {(p) => (
              <Input.Numeric
                {...p}
                bounds={{ lower: 0, upper: 2000 }}
                dragScale={5}
                endContent="px"
              >
                <Button.Button
                  onClick={() => autoFit?.onChange(true)}
                  disabled={autoFit?.value === true}
                  variant="outlined"
                  style={{ borderLeft: "var(--pluto-border-l5)" }}
                  tooltip={
                    autoFit?.value === true
                      ? "Manually enter value to disable auto fit"
                      : "Enable auto fit"
                  }
                >
                  <Icon.AutoFitWidth />
                </Button.Button>
              </Input.Numeric>
            )}
          </Form.Field>
        </Flex.Box>
      </Flex.Box>
      <OrientationControl path="" />
    </FormWrapper>
  );
};

export const OffPageReferenceForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Flex.Box y grow>
      <LabelControls path="label" omit={["maxInlineSize", "align", "direction"]} />
      <ColorControl path="color" />
    </Flex.Box>
    <OrientationControl path="" hideOuter />
  </FormWrapper>
);

export const CylinderForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Flex.Box y grow>
      <LabelControls path="label" />
      <Flex.Box x>
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.Field<number> path="dimensions.width" label="Width" grow>
          {({ value, ...rest }) => (
            <Input.Numeric
              value={value ?? 200}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
              endContent="px"
              {...rest}
            />
          )}
        </Form.Field>
        <Form.Field<number> path="dimensions.height" label="Height" grow>
          {({ value, ...rest }) => (
            <Input.Numeric
              value={value ?? 200}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
              endContent="px"
              {...rest}
            />
          )}
        </Form.Field>
      </Flex.Box>
    </Flex.Box>
    <OrientationControl path="" hideInner />
  </FormWrapper>
);

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

export const BoxForm = (): ReactElement => (
  <TankForm includeBorderRadius includeStrokeWidth />
);

export const SwitchForm = (): ReactElement => <CommonToggleForm hideInnerOrientation />;
