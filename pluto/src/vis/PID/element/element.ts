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

import { Input } from "@/input";

export type Props<P extends object = {}> = P & {
  position: XY;
  selected: boolean;
  editable: boolean;
  onChange: (props: P) => void;
};

export interface FormProps<P extends object = {}> extends Input.Control<P> {}

export interface Spec<P extends object = {}> {
  type: string;
  title: string;
  initialProps: P;
  Element: FC<Props<P>>;
  Form: FC<FormProps<P>>;
  Preview: FC<{}>;
}
