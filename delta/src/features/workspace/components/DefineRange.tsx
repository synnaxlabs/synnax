// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { zodResolver } from "@hookform/resolvers/zod";
import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import type {
  InputDateProps,
  InputTimeProps,
  InputControl,
  ButtonProps,
} from "@synnaxlabs/pluto";
import { Button, Header, Input, Nav, Space, Pack } from "@synnaxlabs/pluto";
import { useForm } from "react-hook-form";
import { AiFillBoxPlot } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Icon } from "@/components/Icon";
import { LayoutRendererProps } from "@/features/layout";

import { useSelectRange } from "../store";
import { addRange } from "../store/slice";

const formSchema = z.object({
  name: z.string(),
  startDate: z.number().int(),
  startTime: z.number().int(),
  endDate: z.number().int(),
  endTime: z.number().int(),
});

type DefineRangeFormProps = z.infer<typeof formSchema>;

export const DefineRange = ({
  layoutKey,
  onClose,
}: LayoutRendererProps): JSX.Element => {
  const now = TimeStamp.now().valueOf();
  const range = useSelectRange(layoutKey);
  let defaultValues;
  if (range != null) {
    defaultValues = {
      name: range.name,
      startDate: range.start,
      startTime: range.start,
      endDate: range.end,
      endTime: range.end,
    };
  } else {
    defaultValues = {
      name: "",
      startDate: now,
      startTime: now,
      endDate: now,
      endTime: now,
    };
  }

  const { control, handleSubmit } = useForm({
    defaultValues,

    // @ts-expect-error
    resolver: zodResolver(formSchema),
  });
  const dispatch = useDispatch();

  const onSubmit = ({
    name,
    startDate,
    startTime,
    endDate,
    endTime,
  }: DefineRangeFormProps): void => {
    const start = startDate + startTime;
    const end = endDate + endTime;
    name = name.trim();
    if (name.length === 0) name = range?.name as string;
    const key = range?.key ?? name.replace(/\s/g, "").toLowerCase();
    // remove leading and trailing whitespace

    dispatch(addRange({ name, start, end, key }));
    onClose();
  };

  return (
    <Space grow>
      <Header level="h4" divided>
        <Header.Title startIcon={<AiFillBoxPlot />}>Define a Range</Header.Title>
      </Header>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void handleSubmit(onSubmit)(e);
        }}
        style={{ flexGrow: 1 }}
        id="define-range"
      >
        <Space grow className="delta-form" size="small">
          <Input.ItemC control={control} name="name" />
          <Space direction="x">
            <Input.ItemC<number, number, InputDateProps, DefineRangeFormProps>
              name="startDate"
              control={control}
              grow
            >
              {Input.Date}
            </Input.ItemC>
            <Input.ItemC<number, number, InputTimeProps, DefineRangeFormProps>
              name="startTime"
              control={control}
              grow
            >
              {Input.Time}
              {TimeModifierRow}
            </Input.ItemC>
          </Space>

          <Space direction="x">
            <Input.ItemC<number, number, InputDateProps, DefineRangeFormProps>
              name="endDate"
              control={control}
              grow
            >
              {Input.Date}
            </Input.ItemC>
            <Input.ItemC<number, number, InputTimeProps, DefineRangeFormProps>
              name="endTime"
              control={control}
              grow
            >
              {Input.Time}
              {(props) => <TimeModifierRow op="add" {...props} />}
            </Input.ItemC>
          </Space>
        </Space>
      </form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button type="submit" form="define-range">
            Save
          </Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Space>
  );
};

interface TimeModifierRowProps extends InputControl<number> {
  op?: "add" | "subtract";
}

const TimeModifierRow = ({
  op,
  value,
  onChange,
}: TimeModifierRowProps): JSX.Element => {
  const onClickFactory =
    (span?: TimeSpan): ButtonProps["onChange"] =>
    (e) => {
      e.preventDefault();
      if (span == null) return onChange(TimeStamp.now().valueOf());
      value = op === "add" ? value + span.valueOf() : value - span.valueOf();
      onChange(value);
    };
  const icon = op === "add" ? <Icon.Add /> : <Icon.Subtract />;
  return (
    <Pack direction="x" size="medium" grow>
      <Button
        variant="outlined"
        onClick={onClickFactory(TimeSpan.hours(1))}
        startIcon={icon}
        justify="center"
        grow
      >
        Hour
      </Button>
      <Button
        variant="outlined"
        onClick={onClickFactory(TimeSpan.minutes(15))}
        startIcon={icon}
        justify="center"
        grow
      >
        15 Minutes
      </Button>
      <Button
        variant="outlined"
        onClick={onClickFactory(TimeSpan.minutes(1))}
        startIcon={icon}
        justify="center"
        grow
      >
        Minute
      </Button>
      <Button variant="outlined" onClick={onClickFactory()} justify="center" grow>
        Now
      </Button>
    </Pack>
  );
};
