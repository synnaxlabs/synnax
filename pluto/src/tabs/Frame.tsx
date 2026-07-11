// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tabs/Tabs.css";

import { type ReactElement, useCallback, useId, useMemo } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useSyncedRef } from "@/hooks";
import { Select } from "@/select/base";
import { state } from "@/state";

export interface ContextValue {
  /** onClose is the Frame-level close handler invoked by {@link Close} parts. */
  onClose?: (key: string) => void;
  /** onRename is the Frame-level rename handler invoked by {@link Editable} parts. */
  onRename?: (key: string, name: string) => void;
  /** getTabID returns the DOM id of the tab element for the given key. */
  getTabID: (key: string) => string;
  /** getPanelID returns the DOM id of the content panel for the given key. */
  getPanelID: (key: string) => string;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tabs.Context",
  providerName: "Tabs.Frame",
});

export { useContext };

export interface FrameProps extends Omit<Flex.BoxProps, "onChange" | "onSelect"> {
  /** The key of the selected tab. Pair with onChange for controlled selection. */
  value?: string;
  /** The key of the tab selected on first render when uncontrolled. */
  initialValue?: string;
  /** onChange is called with a tab's key when it is selected. */
  onChange?: (key: string) => void;
  /** onClose is called with a tab's key when its {@link Close} part is clicked. */
  onClose?: (key: string) => void;
  /** onRename is called when a tab's {@link Editable} part commits a new name. */
  onRename?: (key: string, name: string) => void;
}

/**
 * Frame is the root of a composed tabbed interface. It owns the selected tab key
 * (controlled via value/onChange or uncontrolled via initialValue) and distributes
 * the close and rename handlers to Close and Editable parts rendered within it.
 * Selection state is available to descendants through the Selection package, so
 * only the tabs whose selected state changes re-render.
 */
export const Frame = ({
  value,
  initialValue,
  onChange,
  onClose,
  onRename,
  className,
  children,
  empty = true,
  ...rest
}: FrameProps): ReactElement => {
  const id = useId();
  const [selected, setSelected] = state.usePurePassthrough<string>({
    initial: initialValue ?? "",
    value,
    onChange,
  });
  const valueRef = useSyncedRef(value);
  const handleChange = useCallback(
    (key: string): void => {
      setSelected(key);
      if (valueRef.current == null) onChange?.(key);
    },
    [setSelected, onChange],
  );
  const ctx = useMemo<ContextValue>(
    () => ({
      onClose,
      onRename,
      getTabID: (key: string) => `${id}-tab-${key}`,
      getPanelID: (key: string) => `${id}-panel-${key}`,
    }),
    [id, onClose, onRename],
  );
  return (
    <Context value={ctx}>
      <Select.Context value={selected} onSelect={handleChange}>
        <Flex.Box empty={empty} className={CSS(CSS.B("tabs"), className)} {...rest}>
          {children}
        </Flex.Box>
      </Select.Context>
    </Context>
  );
};
