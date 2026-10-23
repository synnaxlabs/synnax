// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/cells/ChangeVariant.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { REGISTRY, type Variant } from "@/table/cells/registry";

export interface ChangeVariantProps {
  /** The current variant, or undefined when the cells disagree. */
  value?: Variant;
  onChange: (variant: Variant) => void;
}

/** A "Change" button that opens a searchable picker of cell variants. */
export const ChangeVariant = ({
  value,
  onChange,
}: ChangeVariantProps): ReactElement => (
  <Dialog.Frame variant="floating">
    <Select.Frame<Variant, undefined>
      value={value}
      onChange={onChange}
      closeDialogOnSelect
    >
      <Dialog.Trigger
        variant="text"
        size="small"
        hideCaret
        textColor="var(--pluto-primary-p1)"
        aria-label="Change cell type"
      >
        Change
      </Dialog.Trigger>
      <Select.Dialog className={CSS.BE("table-cell-change-variant", "dialog")}>
        <Select.Search placeholder="Search cell types..." />
        <Select.List bordered borderColor={6} grow rounded full="x">
          {Object.values(REGISTRY).map(({ key, name, Icon }) => (
            <Select.Item<Variant> key={key} itemKey={key}>
              <Icon />
              {name}
            </Select.Item>
          ))}
        </Select.List>
      </Select.Dialog>
    </Select.Frame>
  </Dialog.Frame>
);
