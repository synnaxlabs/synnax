// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { type ReactElement, type ReactNode } from "react";

import { Dialog } from "@/dialog";
import { DefaultEmptyContent } from "@/select/Body";
import { Dialog as SelectDialog } from "@/select/Dialog";
import { Frame, type SingleFrameProps } from "@/select/Frame";
import { List } from "@/select/List";
import { usePlaceholder } from "@/select/placeholder";
import { useVisibleCount } from "@/select/registry";
import { useClosed } from "@/select/scope";
import { Search } from "@/select/Search";
import { SingleTrigger, type SingleTriggerProps } from "@/select/SingleTrigger";

export interface SimpleProps<K extends record.Key>
  extends
    Omit<
      SingleFrameProps<K, undefined>,
      | "multiple"
      | "children"
      | "data"
      | "getItem"
      | "subscribe"
      | "virtual"
      | "itemHeight"
      | "overscan"
      | "onFetchMore"
    >,
    Omit<Dialog.FrameProps, "onChange" | "children" | "variant">,
    Pick<SingleTriggerProps, "disabled" | "icon" | "haulType"> {
  /** Singular name of the thing being selected. It builds the placeholder and the
   * empty content. */
  resourceName: string;
  /** The options, as {@link Item}s. */
  children: ReactNode;
  /** Shown when the search hides every option. */
  emptyContent?: ReactNode;
  variant?: Dialog.FrameProps["variant"];
  /** Whether to render the trigger flat and inert, for use inside a preview. */
  preview?: boolean;
  triggerProps?: SingleTriggerProps;
  dialogProps?: Dialog.FrameProps;
}

const Empty = ({
  resourceName,
  emptyContent,
}: Pick<SimpleProps<record.Key>, "resourceName" | "emptyContent">): ReactNode => {
  const closed = useClosed();
  const count = useVisibleCount();
  if (closed || count > 0) return null;
  return emptyContent ?? <DefaultEmptyContent resourceName={resourceName} />;
};

/**
 * A dropdown that selects one of a few fixed options, given as {@link Item} children.
 * Its search filters the options by their text.
 *
 * @example
 * <Select.Simple resourceName="mode" value={mode} onChange={setMode}>
 *   <Select.Item itemKey="fast">Fast</Select.Item>
 *   <Select.Item itemKey="slow">Slow</Select.Item>
 * </Select.Simple>
 */
export const Simple = <K extends record.Key>({
  resourceName,
  value,
  onChange,
  allowNone,
  autoSelectOnNone,
  initialHover,
  enableTriggers,
  closeDialogOnSelect = true,
  children,
  emptyContent,
  haulType,
  disabled,
  icon,
  variant = "connected",
  preview,
  triggerProps,
  dialogProps,
  ...rest
}: SimpleProps<K>): ReactElement => {
  const placeholder = usePlaceholder(resourceName);
  return (
    <Dialog.Frame {...rest} variant={variant}>
      <Frame<K, undefined>
        value={value}
        onChange={onChange}
        allowNone={allowNone}
        autoSelectOnNone={autoSelectOnNone}
        initialHover={initialHover}
        enableTriggers={enableTriggers}
        closeDialogOnSelect={closeDialogOnSelect}
      >
        <SingleTrigger
          haulType={haulType}
          icon={icon}
          placeholder={placeholder}
          aria-label={caseconv.capitalize(resourceName)}
          disabled={disabled}
          preview={preview}
          {...triggerProps}
        />
        <SelectDialog {...dialogProps}>
          <Search placeholder={`Search ${plural(resourceName)}...`} />
          <List bordered borderColor={6} grow rounded full="x">
            {children}
            <Empty resourceName={resourceName} emptyContent={emptyContent} />
          </List>
        </SelectDialog>
      </Frame>
    </Dialog.Frame>
  );
};
