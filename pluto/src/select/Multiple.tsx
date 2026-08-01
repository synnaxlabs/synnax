// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement } from "react";

import { Dialog as BaseDialog } from "@/dialog";
import { type List } from "@/list";
import { Dialog, type DialogProps } from "@/select/Dialog";
import { Frame, type MultipleFrameProps } from "@/select/Frame";
import { MultipleTrigger, type MultipleTriggerProps } from "@/select/MultipleTrigger";

export interface MultipleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
>
  extends
    Omit<MultipleFrameProps<K, E>, "multiple" | "children">,
    Pick<DialogProps<K>, "emptyContent" | "status" | "onSearch" | "actions" | "footer">,
    Omit<BaseDialog.FrameProps, "onChange" | "children" | "variant">,
    Pick<
      MultipleTriggerProps<K, E>,
      "disabled" | "icon" | "haulType" | "createHaulItem"
    >,
    Pick<List.ItemsProps<K>, "children"> {
  resourceName: string;
  renderTag?: MultipleTriggerProps<K, E>["children"];
  triggerProps?: MultipleTriggerProps<K, E>;
  dialogProps?: BaseDialog.FrameProps;
  variant?: BaseDialog.FrameProps["variant"];
  preview?: boolean;
}

export const Multiple = <K extends record.Key, E extends record.Keyed<K> | undefined>({
  resourceName,
  value,
  onChange,
  data,
  getItem,
  subscribe,
  haulType,
  createHaulItem,
  icon,
  disabled,
  onSearch,
  emptyContent,
  status,
  onFetchMore,
  children,
  renderTag,
  actions,
  footer,
  allowNone,
  replaceOnSingle,
  triggerProps,
  virtual = true,
  dialogProps,
  variant = "connected",
  preview,
  ...rest
}: MultipleProps<K, E>): ReactElement => (
  <BaseDialog.Frame variant={variant} {...rest}>
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
      <MultipleTrigger<K, E>
        haulType={haulType}
        createHaulItem={createHaulItem}
        icon={icon}
        placeholder={`Select ${plural(resourceName)}`}
        disabled={disabled}
        preview={preview}
        {...triggerProps}
      >
        {renderTag}
      </MultipleTrigger>
      <Dialog<K>
        onSearch={onSearch}
        emptyContent={emptyContent}
        status={status}
        actions={actions}
        footer={footer}
        resourceName={resourceName}
        {...dialogProps}
      >
        {children}
      </Dialog>
    </Frame>
  </BaseDialog.Frame>
);
