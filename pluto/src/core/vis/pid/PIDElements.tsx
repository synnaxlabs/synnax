// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FC } from "react";

import { XY } from "@synnaxlabs/x";

import { InputControl } from "@/core/std";

export type PIDElementProps<P> = P & {
  position: XY;
  selected: boolean;
  editable: boolean;
};

export interface PIDElementFormProps<P> extends InputControl<P> {}

export interface PIDElementSpec<P> {
  title: string;
  Form: FC<PIDElementFormProps<P>>;
  Element: FC<PIDElementProps<P>>;
  Preview: FC<null>;
}
