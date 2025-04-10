// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/Forms.css";

import { type channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { type bounds, type direction, type location, type xy } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useEffect } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
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
import { type ComponentSize } from "@/util/component";
import { type Button as CoreButton } from "@/vis/button";
import { SelectOrientation } from "@/vis/schematic/SelectOrientation";
import {
  type ControlStateProps,
  type LabelExtensionProps,
} from "@/vis/schematic/Symbols";
import { type Setpoint } from "@/vis/setpoint";
import { type Toggle } from "@/vis/toggle";
import { Value } from "@/vis/value";

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

const ColorControl: Form.FieldT<Color.Crude> = (props): ReactElement => (
  <Form.Field hideIfNull label="Color" align="start" padHelpText={false} {...props}>
    {({ value, onChange, variant: _, ...rest }) => (
      <Color.Swatch
        value={value ?? Color.ZERO.setAlpha(1).rgba255}
        onChange={(v) => onChange(v.rgba255)}
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

const ToggleControlForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<
    Omit<Toggle.UseProps, "aetherKey"> & { control: ControlStateProps }
  >({ path });
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

interface CommonToggleFormProps {
  hideInnerOrientation?: boolean;
}

export const CommonToggleForm = ({
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
  return <Tabs.Tabs {...props} />;
};

const DIMENSIONS_DRAG_SCALE: xy.Crude = { y: 2, x: 0.25 };
const DIMENSIONS_BOUNDS: bounds.Bounds = { lower: 0, upper: 2000 };
const BORDER_RADIUS_BOUNDS: bounds.Bounds = { lower: 0, upper: 51 };

export interface TankFormProps {
  includeBorderRadius?: boolean;
  includeStrokeWidth?: boolean;
}

export const TankForm = ({
  includeBorderRadius = false,
  includeStrokeWidth = false,
}: TankFormProps): ReactElement => (
  <FormWrapper x align="stretch">
    <Align.Space y grow>
      <LabelControls path="label" />
      <Align.Space x>
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
      </Align.Space>
    </Align.Space>
    <OrientationControl path="" hideInner />
  </FormWrapper>
);

export interface PolygonFormProps {
  numSides: number;
}

export const CommonPolygonForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Align.Space direction="y" grow>
      <LabelControls path="label" />
      <Align.Space direction="x">
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
      </Align.Space>
    </Align.Space>
  </FormWrapper>
);

export const CircleForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Align.Space direction="y" grow>
      <LabelControls path="label" />
      <Align.Space direction="x">
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
      </Align.Space>
    </Align.Space>
  </FormWrapper>
);

const VALUE_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telemetry", name: "Telemetry" },
];

export const ValueForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return (
          <FormWrapper y empty>
            <Value.TelemForm path="" />;
          </FormWrapper>
        );

      default:
        return (
          <FormWrapper x>
            <Align.Space y grow>
              <LabelControls path="label" />
              <Align.Space x>
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
                  inputProps={{
                    dragScale: { x: 1, y: 0.25 },
                    bounds: { lower: 40, upper: 500 },
                    endContent: "px",
                  }}
                />
                <Form.Field<Text.Level>
                  path="level"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {(p) => <Select.Text.Level {...p} />}
                </Form.Field>
              </Align.Space>
            </Align.Space>
            <OrientationControl path="" hideInner />
          </FormWrapper>
        );
    }
  }, []);
  const props = Tabs.useStatic({ tabs: VALUE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};

interface LightTelemFormT extends Omit<Toggle.UseProps, "aetherKey"> {}

const LightTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<LightTelemFormT>({ path });
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
        threshold: telem.withinBounds({
          trueBound: {
            lower: threshold.trueBound.lower ?? 0.9,
            upper: threshold.trueBound.upper ?? 1.1,
          },
        }),
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

  const c = Channel.useName(source.channel as number);

  useEffect(() => onChange({ ...value }), [c]);

  return (
    <FormWrapper x align="stretch">
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
        />
      </Input.Item>
      <Input.Item label="Lower Threshold">
        <Input.Numeric
          value={threshold.trueBound.lower ?? 0.9}
          onChange={(v) =>
            handleThresholdChange({
              ...threshold.trueBound,
              lower: v,
            })
          }
        />
      </Input.Item>
      <Input.Item label="Upper Threshold">
        <Input.Numeric
          value={threshold.trueBound.upper ?? 1.1}
          onChange={(v) =>
            handleThresholdChange({
              ...threshold.trueBound,
              upper: v,
            })
          }
        />
      </Input.Item>
    </FormWrapper>
  );
};

export const LightForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return <LightTelemForm path="" />;
      default:
        return <CommonStyleForm />;
    }
  }, []);
  const props = Tabs.useStatic({ tabs: VALUE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};

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
  >({ path });
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
            <Align.Space y align="stretch" grow size="small">
              <LabelControls path="label" />
              <Align.Space x>
                <Form.TextField
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                />
                <Form.Field<ComponentSize>
                  path="size"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {(p) => <Select.ComponentSize {...p} />}
                </Form.Field>
                <ColorControl path="color" />
              </Align.Space>
            </Align.Space>
            <OrientationControl path="" hideInner />
          </FormWrapper>
        );
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};

export const TextBoxForm = (): ReactElement => {
  const autoFit = Form.useField<boolean>({
    path: "autoFit",
    optional: true,
  });
  return (
    <FormWrapper x align="stretch" grow>
      <Align.Space y grow>
        <Align.Space x align="stretch">
          <Form.Field<string> path="text" label="Text" padHelpText={false} grow>
            {(p) => <Input.Text {...p} />}
          </Form.Field>
          <Form.Field<Text.Level> path="level" label="Text Size" padHelpText={false}>
            {(p) => <Select.Text.Level {...p} />}
          </Form.Field>
          <Form.Field<Align.Alignment>
            path="align"
            label="Alignment"
            padHelpText={false}
            hideIfNull
          >
            {(p) => <Select.TextAlignment {...p} />}
          </Form.Field>
        </Align.Space>
        <Align.Space x>
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
                <Button.Icon
                  onClick={() => autoFit?.onChange(true)}
                  disabled={autoFit?.value === true}
                  variant="outlined"
                  tooltip={
                    autoFit?.value === true
                      ? "Manually enter value to disable auto fit"
                      : "Enable auto fit"
                  }
                >
                  <Icon.AutoFitWidth />
                </Button.Icon>
              </Input.Numeric>
            )}
          </Form.Field>
        </Align.Space>
      </Align.Space>
      <OrientationControl path="" />
    </FormWrapper>
  );
};

export const OffPageReferenceForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Align.Space y grow>
      <LabelControls path="label" omit={["maxInlineSize", "align", "direction"]} />
      <ColorControl path="color" />
    </Align.Space>
    <OrientationControl path="" hideOuter />
  </FormWrapper>
);

export const CylinderForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Align.Space y grow>
      <LabelControls path="label" />
      <Align.Space x>
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
      </Align.Space>
    </Align.Space>
    <OrientationControl path="" hideInner />
  </FormWrapper>
);

export const CommonDummyToggleForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Align.Space y grow>
      <LabelControls path="label" />
      <Align.Space x grow>
        <ColorControl path="color" />
        <ScaleControl path="scale" />
        <Form.SwitchField path="clickable" label="Clickable" hideIfNull optional />
      </Align.Space>
    </Align.Space>
    <OrientationControl path="" />
  </FormWrapper>
);

export const BoxForm = (): ReactElement => (
  <TankForm includeBorderRadius includeStrokeWidth />
);

export const SwitchForm = (): ReactElement => <CommonToggleForm hideInnerOrientation />;
