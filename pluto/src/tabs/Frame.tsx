// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tabs/Tabs.css";

import { type ReactElement, useId } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Select } from "@/select";
import { state } from "@/state";

const [Context, useFrameID] = context.create<string>({
  displayName: "Tabs.Context",
  providerName: "Tabs.Frame",
});

export { useFrameID };

/** tabID returns the DOM id of the tab handle for the given key within a Frame. */
export const tabID = (frameID: string, key: string): string => `${frameID}-tab-${key}`;

/** panelID returns the DOM id of the content panel for the given key within a Frame. */
export const panelID = (frameID: string, key: string): string =>
  `${frameID}-panel-${key}`;

export interface FrameProps
  extends
    Omit<Flex.BoxProps, "onChange" | "onSelect">,
    Partial<state.UsePurePassthroughProps<string>> {}

/**
 * Frame is the root of a composed tabbed interface. When given a value, initialValue,
 * or onChange it owns the selected tab key (controlled or uncontrolled) and publishes
 * it to descendants through the Selection context, so only the tabs whose selected
 * state changes re-render.
 *
 * When given no selection at all, the Frame owns nothing: its tabs bind to the nearest
 * enclosing selection context, letting a composite like Panel.Mosaic distribute a
 * single selection across many frames.
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
  return <Context value={id}>{content}</Context>;
};
