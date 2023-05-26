// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Pack, Typography } from "@/core";
import { CSS } from "@/core/css";
import { Handle, Position } from "reactflow"

import "./SensorNumeric.css"

export interface SensorNumericProps {
  label?: string;
  units?: string;
}



export const SensorNumeric = ({
  data: { label, units = "" },
}: { data: SensorNumericProps }) => {
  return (
    <>
      <Handle position={Position.Top} type="source" />
      <Handle position={Position.Left} type="source" />
      <Handle position={Position.Bottom} type="source" />
      <Handle position={Position.Right} type="source" />
      <Pack justify="center" align="center" direction="x" className={CSS.B("sensor-numeric")}>
        <Typography.Text level="p">{label}</Typography.Text>
        <Typography.Text level="p">1066.2 {units}</Typography.Text>
      </Pack >
    </>
  )
}
