// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/Forms.css";

import { type channel } from "@synnaxlabs/client";
import { type bounds, type location, type xy } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback, useEffect } from "react";

import { Align } from "@/align";
import { Channel } from "@/channel";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Form } from "@/form";
import { Input } from "@/input";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { Text } from "@/text";
import { type Button as CoreButton } from "@/vis/button";
import { type LabelExtensionProps } from "@/vis/schematic/Labeled";
import { SelectOrientation } from "@/vis/schematic/SelectOrientation";
import { type ControlStateProps } from "@/vis/schematic/Symbols";
import { Setpoint } from "@/vis/setpoint";
import { type Toggle } from "@/vis/toggle";

export interface SymbolFormProps {}

interface FormWrapperProps extends Align.SpaceProps {}

const FormWrapper: FC<FormWrapperProps> = ({
  className,
  direction,
  ...props
}): ReactElement => (
  <Align.Space
    direction={direction}
    align="stretch"
    className={CSS(CSS.B("symbol-form"), className)}
    size={direction === "x" ? "large" : "medium"}
    {...props}
  />
);

interface SymbolOrientation {
  label?: LabelExtensionProps;
  orientation?: location.Outer;
}

const OrientationControl: Form.FieldT<SymbolOrientation> = (props): ReactElement => (
  <Form.Field<SymbolOrientation> label="Orientation" padHelpText={false} {...props}>
    {({ value, onChange }) => (
      <SelectOrientation
        value={{
          inner: value.orientation ?? "top",
          outer: value.label?.orientation ?? "top",
        }}
        onChange={(v) =>
          onChange({
            ...value,
            orientation: v.inner,
            label: {
              ...value.label,
              orientation: v.outer,
            },
          })
        }
      />
    )}
  </Form.Field>
);

const LabelControls = ({ path }: { path: string }): ReactElement => (
  <Align.Space direction="x" align="stretch">
    <Form.Field<string> path={path + ".label"} label="Label" padHelpText={false} grow>
      {(p) => <Input.Text selectOnFocus {...p} />}
    </Form.Field>
    <Form.Field<Text.Level>
      path={path + ".level"}
      label="Label Size"
      padHelpText={false}
    >
      {(p) => <Text.SelectLevel {...p} />}
    </Form.Field>
  </Align.Space>
);

const ColorControl: Form.FieldT<Color.Crude> = (props): ReactElement => (
  <Form.Field hideIfNull label="Color" align="start" padHelpText={false} {...props}>
    {({ value, onChange, variant: _, ...props }) => (
      <Color.Swatch
        value={value ?? Color.ZERO.setAlpha(1).rgba255}
        onChange={(v) => onChange(v.rgba255)}
        {...props}
        bordered
      />
    )}
  </Form.Field>
);

const ScaleControl: Form.FieldT<number> = (props): ReactElement => (
  <Form.Field hideIfNull label="Scale" align="start" padHelpText={false} {...props}>
    {(p) => <Input.Numeric dragScale={1} bounds={{ lower: 0.5, upper: 10 }} {...p} />}
  </Form.Field>
);

export const CommonStyleForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Align.Space direction="y" grow>
      <LabelControls path="label" />
      <Align.Space direction="x" grow>
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
    <OrientationControl path="" />
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
    v = v ?? 0;
    const t = telem.sourcePipeline("boolean", {
      connections: [
        {
          from: "valueStream",
          to: "threshold",
        },
      ],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
        threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };

  const handleSinkChange = (v: channel.Key | null): void => {
    v = v ?? 0;
    const t = telem.sinkPipeline("boolean", {
      connections: [
        {
          from: "setpoint",
          to: "setter",
        },
      ],
      segments: {
        setter: control.setChannelValue({ channel: v }),
        setpoint: telem.setpoint({
          truthy: 1,
          falsy: 0,
        }),
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
        chip: {
          sink: controlChipSink,
          source: authSource,
        },
        showIndicator: true,
        indicator: {
          statusSource: authSource,
        },
      },
    });
  };

  return (
    <FormWrapper direction="x" grow align="stretch">
      <Input.Item label="State Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
        />
      </Input.Item>
      <Input.Item label="Command Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
    </FormWrapper>
  );
};

const COMMON_TOGGLE_FORM_TABS: Tabs.Tab[] = [
  {
    tabKey: "style",
    name: "Style",
  },
  {
    tabKey: "control",
    name: "Control",
  },
];

export const CommonToggleForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <ToggleControlForm path="" />;
      default:
        return <CommonStyleForm />;
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};

const DIMENSIONS_DRAG_SCALE: xy.Crude = { y: 2, x: 0.25 };
const DIMENSIONS_BOUNDS: bounds.Bounds = { lower: 0, upper: 2000 };

export interface TankFormProps {
  includeBorderRadius?: boolean;
}

export const TankForm = ({
  includeBorderRadius = false,
}: TankFormProps): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Align.Space direction="y" grow>
      <LabelControls path="label" />
      <Align.Space direction="x">
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.Field<number>
          path="borderRadius.x"
          hideIfNull
          optional
          label="X Border Radius"
          grow
        >
          {({ value, ...props }) => (
            <Input.Numeric
              value={value}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
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
          {({ value, ...props }) => (
            <Input.Numeric
              value={value}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
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
            {({ value, ...props }) => (
              <Input.Numeric
                value={value}
                dragScale={DIMENSIONS_DRAG_SCALE}
                bounds={DIMENSIONS_BOUNDS}
                {...props}
              />
            )}
          </Form.Field>
        )}
        <Form.Field<number> path="dimensions.width" label="Width" grow>
          {({ value, ...props }) => (
            <Input.Numeric
              value={value ?? 200}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
              {...props}
            />
          )}
        </Form.Field>
        <Form.Field<number> path="dimensions.height" label="Height" grow>
          {({ value, ...props }) => (
            <Input.Numeric
              value={value ?? 200}
              dragScale={DIMENSIONS_DRAG_SCALE}
              bounds={DIMENSIONS_BOUNDS}
              {...props}
            />
          )}
        </Form.Field>
      </Align.Space>
    </Align.Space>
    <OrientationControl path="" />
  </FormWrapper>
);

const VALUE_FORM_TABS: Tabs.Tab[] = [
  {
    tabKey: "style",
    name: "Style",
  },
  {
    tabKey: "telemetry",
    name: "Telemetry",
  },
];

interface ValueTelemFormT {
  telem: telem.StringSourceSpec;
  tooltip: string[];
}

const ValueTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<ValueTelemFormT>({ path });
  const sourceP = telem.sourcePipelinePropsZ.parse(value.telem?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const stringifier = telem.stringifyNumberProps.parse(
    sourceP.segments.stringifier.props,
  );
  const rollingAverage = telem.rollingAverageProps.parse(
    sourceP.segments.rollingAverage.props,
  );
  const handleSourceChange = (v: channel.Key | null): void => {
    const t = telem.sourcePipeline("string", {
      connections: [
        {
          from: "valueStream",
          to: "rollingAverage",
        },
        {
          from: "rollingAverage",
          to: "stringifier",
        },
      ],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v ?? 0 }),
        stringifier: telem.stringifyNumber({
          precision: stringifier.precision ?? 2,
        }),
        rollingAverage: telem.rollingAverage({
          windowSize: rollingAverage.windowSize ?? 1,
        }),
      },
      outlet: "stringifier",
    });
    onChange({ ...value, telem: t });
  };

  const handlePrecisionChange = (precision: number): void => {
    const t = telem.sourcePipeline("string", {
      connections: [
        {
          from: "valueStream",
          to: "rollingAverage",
        },
        {
          from: "rollingAverage",
          to: "stringifier",
        },
      ],
      segments: {
        valueStream: telem.streamChannelValue({ channel: source.channel }),
        stringifier: telem.stringifyNumber({ precision }),
        rollingAverage: telem.rollingAverage({ windowSize: rollingAverage.windowSize }),
      },
      outlet: "stringifier",
    });
    onChange({ ...value, telem: t });
  };

  const handleRollingAverageChange = (windowSize: number): void => {
    const t = telem.sourcePipeline("string", {
      connections: [
        {
          from: "valueStream",
          to: "rollingAverage",
        },
        {
          from: "rollingAverage",
          to: "stringifier",
        },
      ],
      segments: {
        stringifier: telem.stringifyNumber({ precision: stringifier.precision ?? 2 }),
        valueStream: telem.streamChannelValue({ channel: source.channel }),
        rollingAverage: telem.rollingAverage({ windowSize }),
      },
      outlet: "stringifier",
    });
    onChange({ ...value, telem: t });
  };

  const c = Channel.useName(source.channel as number);
  useEffect(() => {
    onChange({ ...value, tooltip: [c] });
  }, [c]);

  return (
    <FormWrapper direction="x" align="stretch">
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
        />
      </Input.Item>
      <Input.Item label="Precision" align="start">
        <Input.Numeric
          value={stringifier.precision ?? 2}
          bounds={{ lower: 0, upper: 10 }}
          onChange={handlePrecisionChange}
        />
      </Input.Item>
      <Input.Item label="Averaging Window" align="start">
        <Input.Numeric
          value={rollingAverage.windowSize ?? 1}
          bounds={{ lower: 1, upper: 100 }}
          onChange={handleRollingAverageChange}
        />
      </Input.Item>
    </FormWrapper>
  );
};

export const ValueForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return <ValueTelemForm path="" />;
      default:
        return (
          <FormWrapper direction="x">
            <Align.Space direction="y" grow>
              <LabelControls path="label" />
              <Align.Space direction="x">
                <ColorControl path="color" />
                <Form.Field<string>
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                >
                  {(p) => <Input.Text {...p} />}
                </Form.Field>
              </Align.Space>
            </Align.Space>
            <OrientationControl path="" />
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

  const handleSourceChange = (v: channel.Key | null): void => {
    v = v ?? 0;
    const t = telem.sourcePipeline("boolean", {
      connections: [
        {
          from: "valueStream",
          to: "threshold",
        },
      ],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
        threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };

  const c = Channel.useName(source.channel as number);

  useEffect(() => onChange({ ...value }), [c]);

  return (
    <FormWrapper direction="x" align="stretch">
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
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
    v = v ?? 0;
    const t = telem.sinkPipeline("boolean", {
      connections: [
        {
          from: "setpoint",
          to: "setter",
        },
      ],
      segments: {
        setter: control.setChannelValue({ channel: v }),
        setpoint: telem.setpoint({
          truthy: 1,
          falsy: 0,
        }),
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
        chip: {
          sink: controlChipSink,
          source: authSource,
        },
        showIndicator: true,
        indicator: {
          statusSource: authSource,
        },
      },
    });
  };

  return (
    <FormWrapper direction="y">
      <Input.Item label="Output Channel">
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
    </FormWrapper>
  );
};

export const ButtonForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <ButtonTelemForm path="" />;
      default:
        return <CommonStyleForm />;
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};

export const SetpointTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<
    Omit<Setpoint.UseProps, "aetherKey"> & { control: ControlStateProps }
  >({ path });
  const sourceP = telem.sourcePipelinePropsZ.parse(value.source?.props);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSourceChange = (v: channel.Key | null): void => {
    v = v ?? 0;
    const t = telem.sourcePipeline("number", {
      connections: [],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
      },
      outlet: "valueStream",
    });
    onChange({ ...value, source: t });
  };

  const handleSinkChange = (v: channel.Key | null): void => {
    v = v ?? 0;
    const t = telem.sinkPipeline("number", {
      connections: [],
      segments: {
        setter: control.setChannelValue({ channel: v }),
      },
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
        chip: {
          sink: controlChipSink,
          source: authSource,
        },
        showIndicator: true,
        indicator: {
          statusSource: authSource,
        },
      },
    });
  };

  return (
    <FormWrapper direction="x" grow align="stretch">
      <Input.Item label="State Channel" grow>
        <Channel.SelectSingle
          value={source.channel as number}
          onChange={handleSourceChange}
        />
      </Input.Item>
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
          <FormWrapper direction="x" align="stretch">
            <Align.Space direction="y" grow>
              <LabelControls path="label" />
              <Form.Field<string>
                path="units"
                label="Units"
                align="start"
                padHelpText={false}
              >
                {(p) => <Input.Text {...p} />}
              </Form.Field>
            </Align.Space>
            <OrientationControl path="" />
          </FormWrapper>
        );
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};

export const TextBoxForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Align.Space direction="y" grow>
      <Align.Space direction="x">
        <ColorControl path="color" />
        <Form.Field<string> path="text" label="Text" align="start" padHelpText={false}>
          {(p) => <Input.Text {...p} />}
        </Form.Field>
      </Align.Space>
    </Align.Space>
  </FormWrapper>
);
