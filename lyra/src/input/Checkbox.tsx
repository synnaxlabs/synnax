// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Checkbox.css";

import { type ReactElement } from "react";

import { Boolean, type BooleanProps } from "@/input/Boolean";

/** Props for {@link Checkbox}. */
export interface CheckboxProps extends Omit<BooleanProps, "inputType"> {}

/** A boolean input drawn as a checkbox. Use it for a choice the user then submits. */
export const Checkbox = (props: CheckboxProps): ReactElement => (
  <Boolean {...props} inputType="checkbox" />
);
