// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Header, Input, Nav } from "@synnaxlabs/pluto";
import { useForm } from "react-hook-form";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { type Layout } from "@/layout";

import { useSelectRange } from "./selectors";
import { addRange } from "./slice";

const formSchema = z.object({
  name: z.string(),
  startDate: z.number().int(),
  startTime: z.number().int(),
  endDate: z.number().int(),
  endTime: z.number().int(),
});

export const rangeWindowLayout: Layout.LayoutState = {
  key: "defineRange",
  type: "defineRange",
  name: "Define Range",
  location: "window",
  window: {
    resizable: false,
    size: { height: 410, width: 625 },
    navTop: true,
    transparent: true,
  },
};

type DefineRangeFormProps = z.infer<typeof formSchema>;

export const DefineRange = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const now = TimeStamp.now().valueOf();
  const range = useSelectRange(layoutKey);
  let defaultValues;
  if (range != null && range.variant === "static") {
    defaultValues = {
      name: range.name,
      startDate: range.timeRange.start,
      startTime: range.timeRange.start,
      endDate: range.timeRange.end,
      endTime: range.timeRange.end,
    };
  } else {
    defaultValues = {
      startDate: now,
      startTime: now,
      endDate: now,
      endTime: now,
    };
  }

  const { control, handleSubmit } = useForm({
    defaultValues,
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
    const start = Input.combineDateAndTimeValue(startDate, startTime).valueOf();
    const end = Input.combineDateAndTimeValue(endDate, endTime).valueOf();
    name = name.trim();
    if (name.length === 0) name = range?.name as string;
    // remove leading and trailing whitespace
    const key = range?.key ?? (name ?? "").replace(/\s/g, "").toLowerCase();
    dispatch(addRange({ variant: "static", name, timeRange: { start, end }, key }));
    onClose();
  };

  return (
    <Align.Space grow>
      <Header.Header level="h4" divided>
        <Header.Title startIcon={<Icon.Range />}>Define a Range</Header.Title>
      </Header.Header>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void handleSubmit(onSubmit)(e);
        }}
        style={{ flexGrow: 1 }}
        id="define-range"
      >
        <Align.Space grow className="delta-form">
          <Input.ItemControlled control={control} name="name" autoFocus />
          <Align.Space direction="x" size="small">
            <Input.ItemControlled<number, number, Input.DateProps, DefineRangeFormProps>
              name="startDate"
              control={control}
              grow
            >
              {Input.Date}
            </Input.ItemControlled>
            <Input.ItemControlled<number, number, Input.TimeProps, DefineRangeFormProps>
              name="startTime"
              control={control}
              grow
            >
              {Input.Time}
              {(props) => <TimeModifierRow {...props} />}
            </Input.ItemControlled>
          </Align.Space>

          <Align.Space direction="x" size="small">
            <Input.ItemControlled<number, number, Input.DateProps, DefineRangeFormProps>
              name="endDate"
              control={control}
              grow
            >
              {Input.Date}
            </Input.ItemControlled>
            <Input.ItemControlled<number, number, Input.TimeProps, DefineRangeFormProps>
              name="endTime"
              control={control}
              grow
            >
              {Input.Time}
              {(props) => <TimeModifierRow op="add" {...props} />}
            </Input.ItemControlled>
          </Align.Space>
        </Align.Space>
      </form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button type="submit" form="define-range">
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

interface TimeModifierRowProps extends Input.Control<number> {
  op?: "add" | "subtract";
}

const TimeModifierRow = ({
  op,
  value,
  onChange,
}: TimeModifierRowProps): ReactElement => {
  const onClickFactory =
    (span?: TimeSpan): Button.ButtonProps["onChange"] =>
    (e) => {
      e.preventDefault();
      if (span == null) return onChange(TimeStamp.now().valueOf());
      value = op === "add" ? value + span.valueOf() : value - span.valueOf();
      onChange(value);
    };
  const icon = op === "add" ? <Icon.Add /> : <Icon.Subtract />;
  return (
    <Align.Pack direction="x" size="medium" grow>
      <Button.Button
        variant="outlined"
        onClick={onClickFactory(TimeSpan.hours(1))}
        startIcon={icon}
        justify="center"
        grow
      >
        Hour
      </Button.Button>
      <Button.Button
        variant="outlined"
        onClick={onClickFactory(TimeSpan.minutes(15))}
        startIcon={icon}
        justify="center"
        grow
      >
        15 Minutes
      </Button.Button>
      <Button.Button
        variant="outlined"
        onClick={onClickFactory(TimeSpan.minutes(1))}
        startIcon={icon}
        justify="center"
        grow
      >
        Minute
      </Button.Button>
      <Button.Button
        variant="outlined"
        onClick={onClickFactory()}
        justify="center"
        grow
      >
        Now
      </Button.Button>
    </Align.Pack>
  );
};
