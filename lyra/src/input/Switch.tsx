// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Switch.css";

import { type ReactElement } from "react";

import { Boolean, type BooleanProps } from "@/input/Boolean";

/** Props for {@link Switch}. */
export interface SwitchProps extends Omit<BooleanProps, "inputType"> {}

/** A boolean input drawn as a sliding switch. Use it for a setting that takes effect
 * at once. */
export const Switch = (props: SwitchProps): ReactElement => (
  <Boolean {...props} inputType="switch" />
);
