// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, useState } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface ControlGroupProps {
  trigger: ReactElement;
  expanded: boolean;
  children: ReactNode;
}

/**
 * ControlGroup organizes diagram controls into expandable/collapsible groups.
 *
 * @param trigger - The always-visible button that represents the group
 * @param expanded - Whether to show the children buttons (e.g., parent button active)
 * @param children - The buttons to show when expandedg
 *
 * @example
 * ```tsx
 * <ControlGroup
 *   trigger={<Diagram.ToggleEditControl />}
 *   expanded={editable}
 * >
 *   <Diagram.SelectViewportModeControl />
 * </ControlGroup>
 * ```
 */
export const ControlGroup = ({
  trigger,
  expanded,
  children,
}: ControlGroupProps): ReactElement => {
  const [isHovering, setIsHovering] = useState(false);

  return (
    <Flex.Box
      direction="y"
      className={CSS.BE("diagram", "control-group")}
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      {trigger}
      {expanded && isHovering && (
        <Flex.Box direction="y" className={CSS.BE("diagram", "control-group-items")}>
          {children}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
