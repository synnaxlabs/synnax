// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/device/Option.css";

import { type Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** Lets a menu grow to fit the descriptions beside the option names. */
export const MENU_PROPS = { className: CSS.BE("modbus", "menu") };

/** A trigger that shows a picked option without its description. */
export const TRIGGER_PROPS: Select.SingleTriggerProps = {
  className: CSS.BE("modbus", "trigger"),
};

/** A quiet {@link TRIGGER_PROPS} trigger. */
export const SHADOW_TRIGGER_PROPS: Select.SingleTriggerProps = {
  ...TRIGGER_PROPS,
  variant: "shadow",
};

export interface OptionProps {
  glyph: ReactElement;
  name: string;
  description: string;
}

/** A menu option: a glyph, a name, and a description that the trigger hides. */
export const Option = ({ glyph, name, description }: OptionProps): ReactElement => (
  <>
    {glyph}
    <span>{name}</span>
    <span className={CSS.BE("modbus", "option-description")}>{description}</span>
  </>
);
