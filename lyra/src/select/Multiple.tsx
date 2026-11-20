// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, strings } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Dialog as BaseDialog } from "@/dialog";
import type { List } from "@/list";
import { Body, type BodyProps } from "@/select/Body";
import { Dialog, type DialogProps } from "@/select/Dialog";
import { Frame, type MultipleFrameProps } from "@/select/Frame";
import { MultipleTrigger, type MultipleTriggerProps } from "@/select/MultipleTrigger";
import { plural, usePlaceholder } from "@/select/placeholder";

export interface MultipleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
>
  extends
    Omit<MultipleFrameProps<K, E>, "multiple" | "children" | "selectedFirst">,
    Pick<
      BodyProps<K>,
      "emptyContent" | "status" | "onSearch" | "actions" | "footer" | "fixedItems"
    >,
    Omit<BaseDialog.FrameProps, "onChange" | "children" | "variant">,
    Pick<
      MultipleTriggerProps<K, E>,
      "disabled" | "icon" | "haulType" | "createHaulItem"
    >,
    Pick<DialogProps, "menu" | "hasMenu">,
    Pick<List.ItemsProps<K>, "children"> {
  /**
   * Singular name of the thing being selected. It is pluralized for the placeholder.
   */
  resourceName: string;
  /** Renders one tag in the trigger. Defaults to the entry name. */
  renderTag?: MultipleTriggerProps<K, E>["children"];
  triggerProps?: MultipleTriggerProps<K, E>;
  dialogProps?: BaseDialog.FrameProps;
  variant?: BaseDialog.FrameProps["variant"];
  preview?: boolean;
}

/**
 * A dropdown that selects any number of entries, showing each as a removable tag in the
 * trigger. Shift extends a range and control toggles one entry.
 */
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
  fixedItems,
  allowNone,
  replaceOnSingle,
  triggerProps,
  virtual = true,
  dialogProps,
  menu,
  hasMenu,
  variant = "connected",
  preview,
  itemHeight,
  closeDialogOnSelect,
  autoSelectOnNone,
  initialHover,
  enableTriggers,
  cursor,
  moveX,
  pinned,
  overscan,
  ...rest
}: MultipleProps<K, E>): ReactElement => {
  const placeholder = usePlaceholder(plural(resourceName));
  return (
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
        itemHeight={itemHeight}
        closeDialogOnSelect={closeDialogOnSelect}
        autoSelectOnNone={autoSelectOnNone}
        initialHover={initialHover}
        enableTriggers={enableTriggers}
        cursor={cursor}
        moveX={moveX}
        pinned={pinned}
        overscan={overscan}
        selectedFirst
      >
        <MultipleTrigger<K, E>
          haulType={haulType}
          createHaulItem={createHaulItem}
          icon={icon}
          placeholder={placeholder}
          aria-label={strings.capitalize(plural(resourceName))}
          resourceName={resourceName}
          disabled={disabled}
          preview={preview}
          {...triggerProps}
        >
          {renderTag}
        </MultipleTrigger>
        <Dialog {...dialogProps} menu={menu} hasMenu={hasMenu}>
          <Body<K>
            onSearch={onSearch}
            emptyContent={emptyContent}
            status={status}
            actions={actions}
            footer={footer}
            fixedItems={fixedItems}
            resourceName={resourceName}
          >
            {children}
          </Body>
        </Dialog>
      </Frame>
    </BaseDialog.Frame>
  );
};
