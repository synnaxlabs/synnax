// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Select } from "@/select";
import { panelID, tabID, useFrameID } from "@/tabs/Frame";

export interface ContentProps extends Flex.BoxProps {
  /**
   * itemKey renders this content as the panel for the tab with the same key,
   * mounted only while that tab is selected. When omitted, the content is a
   * plain always-rendered slot for consumers that drive their own visibility,
   * e.g. through portals.
   */
  itemKey?: string;
  /**
   * keepMounted keeps the panel mounted and hidden while its tab is unselected
   * instead of unmounting it. Use for content that is expensive to remount,
   * such as WebGL canvases.
   */
  keepMounted?: boolean;
}

/**
 * Content renders the content area of a composed tabbed interface. See
 * {@link ContentProps.itemKey} for the keyed and keyless modes.
 */
export const Content = ({
  itemKey,
  keepMounted = false,
  className,
  children,
  ...rest
}: ContentProps): ReactElement | null => {
  const frameID = useFrameID("Tabs.Content");
  const { selected } = Select.useItemState(itemKey ?? "");
  const cls = CSS(CSS.BE("tabs", "content"), className);
  if (itemKey == null)
    return (
      <Flex.Box className={cls} {...rest}>
        {children}
      </Flex.Box>
    );
  if (!selected && !keepMounted) return null;
  return (
    <Flex.Box
      id={panelID(frameID, itemKey)}
      role="tabpanel"
      aria-labelledby={tabID(frameID, itemKey)}
      hidden={!selected}
      className={cls}
      {...rest}
    >
      {children}
    </Flex.Box>
  );
};
