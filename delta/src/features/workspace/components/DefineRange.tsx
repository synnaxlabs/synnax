// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";
import type { InputDateProps, InputTimeProps } from "@synnaxlabs/pluto";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import { useForm } from "react-hook-form";
import { AiFillBoxPlot } from "react-icons/ai";
import { useDispatch } from "react-redux";

import { addRange } from "../store/slice";

import { LayoutRendererProps } from "@/features/layout";

export const DefineRange = ({
  layoutKey,
  onClose,
}: LayoutRendererProps): JSX.Element => {
  const now = TimeStamp.now();
  const { control, handleSubmit } = useForm({
    defaultValues: {
      startDate: now,
      startTime: now,
      endDate: now,
      endTime: now,
    },
  });
  const dispatch = useDispatch();

  interface DefineRangeFormProps {
    name: string;
    startDate: number;
    startTime: number;
    endDate: number;
    endTime: number;
  }

  const onSubmit = ({
    name,
    startDate,
    startTime,
    endDate,
    endTime,
  }: DefineRangeFormProps): void => {
    const start = startDate + startTime;
    const end = endDate + endTime;

    dispatch(
      addRange({
        name,
        start,
        end,
        key: name.replace(/\s/g, "").toLowerCase(),
      })
    );
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
        id="define-range"
      >
        <Space grow className="delta-form">
          <Input.ItemC control={control} name="name" />
          <Space direction="horizontal">
            <Input.ItemC<number, InputDateProps>
              name="startDate"
              control={control}
              grow
            >
              {Input.Date}
            </Input.ItemC>
            <Input.ItemC<number, InputTimeProps>
              name="startTime"
              control={control}
              grow
            >
              {Input.Time}
            </Input.ItemC>
          </Space>
          <Space direction="horizontal">
            <Input.ItemC<number, InputDateProps> name="endDate" control={control} grow>
              {Input.Date}
            </Input.ItemC>
            <Input.ItemC<number, InputTimeProps> name="endTime" control={control} grow>
              {Input.Time}
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
