// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Dialog } from "@/dialog";
import { type List } from "@/list";
import { Select } from "@/select";
import { type DialogProps } from "@/select/Dialog";
import { Frame, type MultipleFrameProps } from "@/select/Frame";
import { type MultipleTriggerProps } from "@/select/MultipleTrigger";
import {
  transformDialogVariant,
  transformTriggerVariant,
  type Variant,
} from "@/select/variant";

export interface MultipleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends Omit<MultipleFrameProps<K, E>, "multiple" | "children">,
    Pick<DialogProps<K>, "emptyContent" | "status" | "onSearch" | "actions">,
    Omit<Dialog.FrameProps, "onChange" | "children" | "variant">,
    Pick<MultipleTriggerProps<K>, "disabled" | "icon" | "haulType">,
    Pick<List.ItemsProps<K>, "children"> {
  resourceName: string;
  renderTag?: Select.MultipleTriggerProps<K>["children"];
  triggerProps?: Select.MultipleTriggerProps<K>;
  dialogProps?: Dialog.FrameProps;
  variant?: Variant;
}

export const Multiple = <K extends record.Key, E extends record.Keyed<K> | undefined>({
  resourceName,
  value,
  onChange,
  data,
  getItem,
  subscribe,
  haulType,
  icon,
  disabled,
  onSearch,
  emptyContent,
  status,
  onFetchMore,
  children,
  renderTag,
  actions,
  allowNone,
  replaceOnSingle,
  triggerProps,
  virtual = true,
  dialogProps,
  variant = "connected",
  ...rest
}: MultipleProps<K, E>): ReactElement => (
  <Dialog.Frame variant={transformDialogVariant(variant)} {...rest}>
    <Frame<K, E>
      multiple
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      allowNone={allowNone}
      replaceOnSingle={replaceOnSingle}
      virtual={virtual}
    >
      <Select.MultipleTrigger
        haulType={haulType}
        icon={icon}
        placeholder={`Select ${resourceName}s`}
        disabled={disabled}
        variant={transformTriggerVariant(variant)}
        {...triggerProps}
      >
        {renderTag}
      </Select.MultipleTrigger>
      <Select.Dialog<K>
        onSearch={onSearch}
        emptyContent={emptyContent}
        status={status}
        actions={actions}
        resourceName={resourceName}
        {...dialogProps}
      >
        {children}
      </Select.Dialog>
    </Frame>
  </Dialog.Frame>
);
