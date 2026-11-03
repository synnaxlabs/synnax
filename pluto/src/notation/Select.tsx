// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/notation/Select.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as BaseSelect } from "@synnaxlabs/lyra/select";
import { type notation } from "@synnaxlabs/x";
import { type ReactElement, useSyncExternalStore } from "react";

import { CSS } from "@/css";

export interface SelectNotationProps extends BaseSelect.ButtonsProps<notation.Notation> {}

const LABEL_CLASS = CSS.BE("notation-select", "label");
const ICON_CLASS = CSS.BE("notation-select", "icon");
// Matches the query in Select.css that swaps the labels for icons.
const ICONS_QUERY = "(width <= 1100px)";

const subscribe = (onChange: () => void): (() => void) => {
  const query = window.matchMedia(ICONS_QUERY);
  query.addEventListener("change", onChange);
  return () => query.removeEventListener("change", onChange);
};

const showsIcons = (): boolean => window.matchMedia(ICONS_QUERY).matches;

export const Select = ({ className, ...rest }: SelectNotationProps): ReactElement => {
  // A label already names its button, so only an icon gets a tooltip.
  const icons = useSyncExternalStore(subscribe, showsIcons);
  const tooltip = (label: string): string | undefined => (icons ? label : undefined);
  return (
    <BaseSelect.Buttons
      {...rest}
      className={CSS.cls(className, CSS.B("notation-select"))}
    >
      <BaseSelect.Item itemKey="standard" tooltip={tooltip("Standard")}>
        <Icon.Decimal className={ICON_CLASS} />
        <span className={LABEL_CLASS}>Standard</span>
      </BaseSelect.Item>
      <BaseSelect.Item itemKey="scientific" tooltip={tooltip("Scientific")}>
        <Icon.Scientific className={ICON_CLASS} />
        <span className={LABEL_CLASS}>Scientific</span>
      </BaseSelect.Item>
      <BaseSelect.Item itemKey="engineering" tooltip={tooltip("Engineering")}>
        <Icon.Engineering className={ICON_CLASS} />
        <span className={LABEL_CLASS}>Engineering</span>
      </BaseSelect.Item>
    </BaseSelect.Buttons>
  );
};
