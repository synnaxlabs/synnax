// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useRef, type ReactElement, useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Header,
  Input,
  Nav,
  Synnax,
  useAsyncEffect,
  componentRenderProp,
} from "@synnaxlabs/pluto";
import { useForm } from "react-hook-form";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { type Layout } from "@/layout";
import { useSelect } from "@/range/selectors";
import { add } from "@/range/slice";

const formSchema = z.object({
  name: z.string().min(2),
  startDate: z.number().int(),
  startTime: z.number().int(),
  endDate: z.number().int(),
  endTime: z.number().int(),
});

const RANGE_WINDOW_KEY = "defineRange";

export const defineWindowLayout: Layout.LayoutState = {
  key: RANGE_WINDOW_KEY,
  type: RANGE_WINDOW_KEY,
  windowKey: RANGE_WINDOW_KEY,
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

export const Define = ({ layoutKey, onClose }: Layout.RendererProps): ReactElement => {
  const now = TimeStamp.now().valueOf();
  const range = useSelect(layoutKey);
  const [loading, setLoading] = useState(false);
  const client = Synnax.use();
  let defaultValues;
  const isCreate = layoutKey === RANGE_WINDOW_KEY;
  const isRemoteEdit = client != null && !isCreate && range != null;

  if (isCreate)
    defaultValues = {
      name: "",
      startDate: now,
      startTime: now,
      endDate: now,
      endTime: now,
    };
  else if (range != null && range.variant === "static")
    defaultValues = {
      name: range.name,
      startDate: range.timeRange.start,
      startTime: range.timeRange.start,
      endDate: range.timeRange.end,
      endTime: range.timeRange.end,
      savePermanent: true,
    };

  const { control, handleSubmit, reset } = useForm({
    defaultValues,
    resolver: zodResolver(formSchema),
  });

  useAsyncEffect(async () => {
    if (!isRemoteEdit) return;
    const rng = await client.ranges.retrieve(layoutKey);
    reset({
      name: rng.name,
      startDate: rng.timeRange.start.valueOf(),
      startTime: rng.timeRange.start.valueOf(),
      endDate: rng.timeRange.end.valueOf(),
      endTime: rng.timeRange.end.valueOf(),
    });
  }, [isRemoteEdit]);

  const dispatch = useDispatch();
  const savePermanently = useRef(false);
  const isPersistedEdit = !isCreate && range?.variant === "static" && range.persisted;

  const onSubmit = async ({
    name,
    startDate,
    startTime,
    endDate,
    endTime,
  }: DefineRangeFormProps): Promise<void> => {
    const start = Input.combineDateAndTimeValue(startDate, startTime).valueOf();
    const end = Input.combineDateAndTimeValue(endDate, endTime).valueOf();
    name = name.trim();
    if (name.length === 0) name = range?.name!;
    // remove leading and trailing whitespace
    const key = isCreate ? uuidv4() : layoutKey;

    const persisted = savePermanently.current || isPersistedEdit;

    if (persisted && client != null) {
      try {
        setLoading(true);
        await client.ranges.create({
          name,
          timeRange: new TimeRange(start, end),
          key,
        });
      } finally {
        setLoading(false);
      }
    }
    dispatch(
      add({
        ranges: [
          {
            variant: "static",
            name,
            timeRange: { start, end },
            key,
            persisted,
          },
        ],
      }),
    );
    onClose();
  };

  const formRef = useRef<HTMLFormElement>(null);

  return (
    <Align.Space grow>
      <Header.Header level="h4">
        <Header.Title startIcon={<Icon.Range />}>Define a Range</Header.Title>
      </Header.Header>
      <form
        ref={formRef}
        onSubmit={(e) => {
          e.preventDefault();
          void handleSubmit(onSubmit)(e);
        }}
        style={{ flexGrow: 1 }}
        id="define-range"
      >
        <Align.Space grow className="console-form">
          <Input.ItemControlled control={control} name="name" autoFocus>
            {(props) => <Input.Text {...props} />}
          </Input.ItemControlled>
          <Align.Space direction="x" size="small">
            <Input.ItemControlled<number, number, Input.DateProps, DefineRangeFormProps>
              name="startDate"
              control={control}
              grow
            >
              {componentRenderProp(Input.Date)}
            </Input.ItemControlled>
            <Input.ItemControlled<number, number, Input.TimeProps, DefineRangeFormProps>
              name="startTime"
              control={control}
              grow
            >
              {componentRenderProp(Input.Time)}
            </Input.ItemControlled>
          </Align.Space>

          <Align.Space direction="x" size="small">
            <Input.ItemControlled<number, number, Input.DateProps, DefineRangeFormProps>
              name="endDate"
              control={control}
              grow
            >
              {componentRenderProp(Input.Date)}
            </Input.ItemControlled>
            <Input.ItemControlled<number, number, Input.TimeProps, DefineRangeFormProps>
              name="endTime"
              control={control}
              grow
            >
              {componentRenderProp(Input.Time)}
            </Input.ItemControlled>
          </Align.Space>
        </Align.Space>
      </form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          {isCreate && (
            <Button.Button
              onClick={() => {
                savePermanently.current = true;
                formRef.current?.requestSubmit();
              }}
              variant="outlined"
              disabled={client == null || loading}
              loading={loading}
            >
              Save Permanently
            </Button.Button>
          )}
          <Button.Button onClick={() => formRef.current?.requestSubmit()}>
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
