// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/notation/Select.css";

import { notation } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Select as BaseSelect } from "@/select";

export interface SelectNotationProps extends Omit<
  BaseSelect.ButtonsProps<notation.Notation>,
  "keys"
> {}

const LABEL_CLASS = CSS.BE("notation-select", "label");
const ICON_CLASS = CSS.BE("notation-select", "icon");

export const Select = ({ className, ...rest }: SelectNotationProps): ReactElement => (
  <BaseSelect.Buttons
    {...rest}
    keys={notation.NOTATIONS}
    className={CSS(className, CSS.B("notation-select"))}
  >
    <BaseSelect.Button itemKey="standard" tooltip="Standard">
      <Icon.Decimal className={ICON_CLASS} />
      <span className={LABEL_CLASS}>Standard</span>
    </BaseSelect.Button>
    <BaseSelect.Button itemKey="scientific" tooltip="Scientific">
      <Icon.Scientific className={ICON_CLASS} />
      <span className={LABEL_CLASS}>Scientific</span>
    </BaseSelect.Button>
    <BaseSelect.Button itemKey="engineering" tooltip="Engineering">
      <Icon.Engineering className={ICON_CLASS} />
      <span className={LABEL_CLASS}>Engineering</span>
    </BaseSelect.Button>
  </BaseSelect.Buttons>
);
