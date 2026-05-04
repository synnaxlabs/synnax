// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/forms/forms.css";

import { type channel } from "@synnaxlabs/client";
import {
  type bounds,
  color,
  type direction,
  id,
  type location,
  type xy,
} from "@synnaxlabs/x";
import { type CSSProperties, type FC, type ReactElement } from "react";

import { Button } from "@/button";
import { Channel } from "@/channel";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Direction } from "@/direction";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { type Label } from "@/schematic/node/common/label";
import { SelectOrientation } from "@/schematic/node/common/orientation/SelectOrientation";
import {
  type ControlStateProps,
  type StateMapping,
} from "@/schematic/node/common/symbol/factories";
import { Select } from "@/select";
import { type Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { Text } from "@/text";
import { type Toggle } from "@/vis/toggle";

export interface NodeFormProps extends Pick<Tabs.TabsProps, "actions"> {
  schematicKey?: string;
}

interface FormWrapperProps extends Flex.BoxProps {}

export const FormWrapper: FC<FormWrapperProps> = ({
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
  label?: Label.Config;
  orientation?: location.Outer;
}

interface ShowOrientationProps {
  hideOuter?: boolean;
  hideInner?: boolean;
  showOuterCenter?: boolean;
}

export const OrientationControl = ({
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

const MAX_INLINE_SIZE_STYLE: CSSProperties = { maxWidth: 125 };

export const LabelControls = ({
  path,
  omit = [],
}: LabelControlsProps): ReactElement => (
  <Flex.Box x align="stretch">
    <Form.Field<string> path={`${path}.label`} label="Label" padHelpText={false} grow>
      {(p) => <Input.Text selectOnFocus {...p} />}
    </Form.Field>
    <Form.NumericField
      visible={!omit.includes("maxInlineSize")}
      style={MAX_INLINE_SIZE_STYLE}
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

export const ColorControl: Form.FieldT<color.Crude> = (props): ReactElement => (
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

export const ScaleControl: Form.FieldT<number> = (props): ReactElement => (
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

interface ToggleControlFormProps {
  path: string;
  omit?: string[];
}

export const ACTIVATION_DELAY_INPUT_PROPS: Partial<Input.NumericProps> = {
  endContent: "ms",
  min: 0,
};

export const COMMON_TOGGLE_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "control", name: "Control" },
];

export const ToggleControlForm = ({
  path,
  omit = [],
}: ToggleControlFormProps): ReactElement => {
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
    <FormWrapper y empty>
      <Flex.Box x grow>
        <Input.Item label="State Channel" grow padHelpText={false}>
          <Channel.SelectSingle
            value={source.channel as number}
            onChange={handleSourceChange}
          />
        </Input.Item>
        <Input.Item label="Command Channel" grow padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x grow>
        {!omit.includes("onClickDelay") && (
          <Form.NumericField
            label="Activation Delay"
            path="onClickDelay"
            grow
            inputProps={ACTIVATION_DELAY_INPUT_PROPS}
            hideIfNull
            padHelpText={false}
          />
        )}
        <Form.SwitchField
          path="control.show"
          label="Show Control Chip"
          hideIfNull
          optional
          padHelpText={false}
        />
      </Flex.Box>
    </FormWrapper>
  );
};

export const SelectTextLevel = ({
  value,
  onChange,
}: Input.Control<Text.Level>): ReactElement => (
  <Select.Text.Level value={value} onChange={onChange} />
);

export const DIMENSIONS_DRAG_SCALE: xy.Crude = { y: 2, x: 0.25 };
export const PERCENT_DRAG_SCALE: xy.Crude = { y: 0.25, x: 0.05 };
export const STROKE_WIDTH_DRAG_SCALE: xy.Crude = { y: 0.1, x: 0.02 };
export const DIMENSIONS_BOUNDS: bounds.Bounds = { lower: 0, upper: 2000 };
export const BORDER_RADIUS_BOUNDS: bounds.Bounds = { lower: 0, upper: 51 };
export const STROKE_WIDTH_BOUNDS: bounds.Bounds = { lower: 0, upper: 21 };

export const DIMENSIONS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: DIMENSIONS_DRAG_SCALE,
  bounds: DIMENSIONS_BOUNDS,
  endContent: "px",
};

export const PERCENT_BORDER_RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: PERCENT_DRAG_SCALE,
  bounds: BORDER_RADIUS_BOUNDS,
  endContent: "%",
};

export const STROKE_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: STROKE_WIDTH_DRAG_SCALE,
  bounds: STROKE_WIDTH_BOUNDS,
  endContent: "px",
};

export const valueWidthInputProps = {
  dragScale: { x: 1, y: 0.25 },
  bounds: { lower: 40, upper: 500 },
  endContent: "px",
} as const;

interface StateMappingFormProps {
  path: string;
  showColor?: boolean;
}

interface StateMappingListItemProps {
  itemKey: string;
  index: number;
  path: string;
  showColor: boolean;
  duplicateValue: boolean;
  onRemove: (key: string) => void;
}

const StateMappingListItem = ({
  itemKey,
  index,
  path,
  showColor,
  duplicateValue,
  onRemove,
}: StateMappingListItemProps): ReactElement => {
  const basePath = `${path}.${itemKey}`;
  return (
    <List.Item
      key={itemKey}
      itemKey={itemKey}
      index={index}
      x
      align="center"
      gap="small"
    >
      <Form.TextField
        showLabel={false}
        showHelpText={false}
        path={`${basePath}.name`}
        grow
      />
      <Form.NumericField
        path={`${basePath}.value`}
        showHelpText={false}
        showLabel={false}
        inputProps={{
          status: duplicateValue ? "error" : undefined,
          className: CSS.BE("state-mapping-list", "value"),
          showDragHandle: false,
          tooltip: duplicateValue ? "Duplicate value" : undefined,
        }}
      />
      {showColor && (
        <Form.Field<color.Crude>
          path={`${basePath}.color`}
          showLabel={false}
          showHelpText={false}
        >
          {({ value, onChange }) => (
            <Color.Swatch value={value ?? color.ZERO} onChange={onChange} bordered />
          )}
        </Form.Field>
      )}
      <Button.Button
        onClick={() => onRemove(itemKey)}
        size="small"
        variant="text"
        ghost
      >
        <Icon.Close />
      </Button.Button>
    </List.Item>
  );
};

export const StateMappingForm = ({
  path,
  showColor = false,
}: StateMappingFormProps): ReactElement => {
  const { data, push, remove } = Form.useFieldList<string, StateMapping>(path);
  const options = Form.useFieldValue<StateMapping[]>(path);

  const handleAddOption = (): void => {
    const nextValue =
      options.length === 0 ? 0 : Math.max(...options.map((o) => o.value)) + 1;
    push({ key: id.create(), name: "", value: nextValue });
  };

  const duplicateValues = new Set(
    options.map((o) => o.value).filter((v, i, arr) => arr.indexOf(v) !== i),
  );

  return (
    <Flex.Box
      y
      gap="small"
      align="stretch"
      grow={options.length === 0}
      className={CSS.B("state-mapping-list")}
    >
      <List.Frame data={data}>
        <List.Items<string>
          grow
          emptyContent={
            <Flex.Box center grow>
              <Text.Text center status="disabled" gap="tiny">
                No options added.
                <Text.Text variant="link" onClick={handleAddOption}>
                  Add an option
                </Text.Text>
              </Text.Text>
            </Flex.Box>
          }
        >
          {({ itemKey, index }) => {
            if (index >= options.length) return null;
            return (
              <StateMappingListItem
                itemKey={itemKey}
                index={index}
                path={path}
                showColor={showColor}
                duplicateValue={duplicateValues.has(options[index].value)}
                onRemove={remove}
              />
            );
          }}
        </List.Items>
      </List.Frame>
      {options.length > 0 && (
        <Button.Button
          onClick={handleAddOption}
          variant="text"
          size="small"
          textColor={10}
        >
          <Icon.Add />
          Add option
        </Button.Button>
      )}
    </Flex.Box>
  );
};
