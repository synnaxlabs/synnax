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
import { Frame, type SingleFrameProps } from "@/select/Frame";
import { type SingleTriggerProps } from "@/select/SingleTrigger";

export interface SingleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends Omit<SingleFrameProps<K, E>, "multiple" | "children">,
    Pick<DialogProps<K>, "emptyContent" | "status" | "onSearch" | "actions">,
    Omit<Dialog.FrameProps, "onChange" | "children">,
    Pick<SingleTriggerProps, "disabled" | "icon" | "haulType">,
    Pick<List.ItemsProps<K>, "children"> {
  resourceName: string;
  triggerVariant?: Select.SingleTriggerProps["variant"];
  dialogProps?: Dialog.FrameProps;
}

export const Single = <K extends record.Key, E extends record.Keyed<K> | undefined>({
  resourceName,
  onChange,
  value,
  allowNone,
  emptyContent,
  haulType,
  data,
  getItem,
  subscribe,
  itemHeight,
  onFetchMore,
  disabled,
  onSearch,
  status,
  icon,
  children,
  variant = "connected",
  actions,
  triggerVariant,
  dialogProps,
  ...rest
}: SingleProps<K, E>): ReactElement => (
  <Dialog.Frame {...rest} variant={variant}>
    <Frame<K, E>
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      allowNone={allowNone}
      onFetchMore={onFetchMore}
      itemHeight={itemHeight}
      virtual
    >
      <Select.SingleTrigger
        haulType={haulType}
        icon={icon}
        placeholder={`Select a ${resourceName}`}
        disabled={disabled}
        variant={triggerVariant}
      />
      <Select.Dialog<K>
        onSearch={onSearch}
        resourceName={resourceName}
        searchPlaceholder={`Search ${resourceName}s...`}
        emptyContent={emptyContent}
        status={status}
        actions={actions}
        {...dialogProps}
      >
        {children}
      </Select.Dialog>
    </Frame>
  </Dialog.Frame>
);
