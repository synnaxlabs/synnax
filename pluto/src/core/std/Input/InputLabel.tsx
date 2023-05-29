// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, HTMLAttributes, ReactElement } from "react";

import { CSS } from "@/core/css";

import "@/core/std/Input/InputLabel.css";

export interface InputLabelProps
  extends DetailedHTMLProps<HTMLAttributes<HTMLLabelElement>, HTMLLabelElement> {
  label?: string;
}

export const InputLabel = ({ className, ...props }: InputLabelProps): ReactElement => {
  return <label className={CSS(CSS.B("input-label"), className)} {...props} />;
};
