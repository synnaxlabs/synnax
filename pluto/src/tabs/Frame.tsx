// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tabs/Tabs.css";

import { type ReactElement, useId, useMemo } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Select } from "@/select/base";
import { state } from "@/state";

export interface ContextValue {
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

export interface FrameProps
  extends
    Omit<Flex.BoxProps, "onChange" | "onSelect">,
    Partial<state.UsePurePassthroughProps<string>> {}

/**
 * Frame is the root of a composed tabbed interface. It owns the selected tab key
 * (controlled via value/onChange or uncontrolled via initialValue) and distributes
 * the close and rename handlers to Close and Name parts rendered within it.
 * Selection state is available to descendants through the Selection package, so
 * only the tabs whose selected state changes re-render.
 *
 * When given no selection at all (no value, initialValue, or onChange), the Frame
 * does not own a selection: its tabs bind to the nearest enclosing selection
 * context, letting a composite like Panel.Mosaic distribute a single selection
 * across many frames.
 */
export const Frame = ({
  value,
  initialValue,
  onChange,
  className,
  children,
  empty = true,
  ...rest
}: FrameProps): ReactElement => {
  const id = useId();
  const ownsSelection =
    value !== undefined || initialValue !== undefined || onChange !== undefined;
  const [selected, setSelected] = state.usePurePassthrough<string>({
    initialValue: initialValue ?? "",
    value,
    onChange,
  });
  const ctxValue = useMemo<ContextValue>(
    () => ({
      getTabID: (key: string) => `${id}-tab-${key}`,
      getPanelID: (key: string) => `${id}-panel-${key}`,
    }),
    [id],
  );
  let content = (
    <Flex.Box empty={empty} className={CSS(CSS.B("tabs"), className)} {...rest}>
      {children}
    </Flex.Box>
  );
  if (ownsSelection)
    content = (
      <Select.Context value={selected} onSelect={setSelected}>
        {content}
      </Select.Context>
    );
  return <Context value={ctxValue}>{content}</Context>;
};
