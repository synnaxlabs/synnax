// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Space, Input, Header, Nav, Button } from "@synnaxlabs/pluto";
import { useForm } from "react-hook-form";
import { AiFillBoxPlot } from "react-icons/ai";
import { useDispatch } from "react-redux";

import { addRange } from "../store/slice";

import { LayoutRendererProps } from "@/features/layout";

const timeStringToNanoseconds = (time: string): number => {
  const p = time.split(":");
  let s = 0;
  let m = 1;

  while (p.length > 0) {
    s += m * parseInt(p.pop() as string, 10);
    m *= 60;
  }

  return s * 1000000000;
};

const dateStringToNanoseconds = (date: string): number => {
  const dateObj = new Date(date);
  return dateObj.getTime() * 1000000;
};

export const DefineRange = ({
  layoutKey,
  onClose,
}: LayoutRendererProps): JSX.Element => {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm();
  const dispatch = useDispatch();

  const onSubmit = (data: any): void => {
    let start = dateStringToNanoseconds(data.dateStart);
    start += timeStringToNanoseconds(data.timeStart);
    let end = dateStringToNanoseconds(data.dateEnd);
    end += timeStringToNanoseconds(data.timeEnd);
    dispatch(
      addRange({
        name: data.name,
        key: data.name.replace(/\s/g, "").toLowerCase(),
        start,
        end,
      })
    );
    onClose();
  };

  return (
    <Space grow>
      <Header level="h4" icon={<AiFillBoxPlot />} divided>
        Define a Range
      </Header>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void handleSubmit(onSubmit)(e);
        }}
        id="define-range"
      >
        <Space grow className="delta-form">
          <Input.Item
            label="Name"
            helpText={errors.name?.message?.toString()}
            {...register("name")}
          />
          <Space direction="horizontal">
            <Input.Item
              label="Start Date"
              size="medium"
              helpText={errors.dateStart?.message?.toString()}
              {...register("dateStart", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Date}
            </Input.Item>
            <Input.Item
              label="Start Time"
              size="medium"
              helpText={errors.timeStart?.message?.toString()}
              {...register("timeStart", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Time}
            </Input.Item>
          </Space>
          <Space direction="horizontal">
            <Input.Item
              label="End Date"
              size="medium"
              helpText={errors.dateEnd?.message?.toString()}
              {...register("dateEnd", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Date}
            </Input.Item>
            <Input.Item
              label="End Time"
              size="medium"
              helpText={errors.timeEnd?.message?.toString()}
              {...register("timeEnd", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Time}
            </Input.Item>
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
