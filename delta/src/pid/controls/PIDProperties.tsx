// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { ReactElement } from "react";

import { Status, ValuePIDElementSpec, Space } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useSelectSelectedPIDElementsProps } from "../store/selectors";
import { setPIDElementProps } from "../store/slice";

import { CSS } from "@/css";

import "@/pid/controls/PIDProperties.css";

export interface PIDPropertiesProps {
  layoutKey: string;
}

export const PIDProperties = ({ layoutKey }: PIDPropertiesProps): ReactElement => {
  const elements = useSelectSelectedPIDElementsProps(layoutKey);

  const dispatch = useDispatch();

  const handleChange = (props: any) => {
    dispatch(setPIDElementProps({ layoutKey, key: elements[0].key, props }));
  };

  if (elements.length === 0) {
    return (
      <Status.Text.Centered variant="disabled" hideIcon>
        No PID Element Selected. Select an Element To Edit Its Properties.
      </Status.Text.Centered>
    );
  }

  return (
    <Space className={CSS.B("pid-properties")} size="small">
      <ValuePIDElementSpec.Form value={elements[0].props} onChange={handleChange} />;
    </Space>
  );
};
