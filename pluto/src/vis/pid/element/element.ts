// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";

import { type xy, type UnknownRecord } from "@synnaxlabs/x";

import { type Theming } from "@/index";
import { type Input } from "@/input";

export type Props<P extends object = UnknownRecord> = P & {
  position: xy.XY;
  selected: boolean;
  editable: boolean;
  onChange: (props: P) => void;
};

export interface FormProps<P extends object = UnknownRecord> extends Input.Control<P> {}

export interface Spec<P extends object = UnknownRecord> {
  type: string;
  title: string;
  initialProps: (theme: Theming.Theme) => P;
  Element: FC<Props<P>>;
  Form: FC<FormProps<P>>;
  Preview: FC<P>;
}
