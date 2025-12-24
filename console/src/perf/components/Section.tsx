// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/Section.css";

import { Flex, Icon, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement, type ReactNode, useState } from "react";

import { WithTooltip } from "@/perf/components/WithTooltip";
import { type DisplayStatus } from "@/perf/ui-types";

interface SectionProps {
  title: string;
  secondaryText?: ReactNode;
  secondaryStatus?: DisplayStatus;
  secondaryTooltip?: string;
  actions?: ReactNode;
  subheader?: ReactNode;
  defaultOpen?: boolean;
  children?: ReactNode;
}

const SectionImpl = ({
  title,
  secondaryText,
  secondaryStatus,
  secondaryTooltip,
  actions,
  subheader,
  defaultOpen = false,
  children,
}: SectionProps): ReactElement => {
  const [open, setOpen] = useState(defaultOpen);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      setOpen(!open);
    }
  };

  const handleActionsClick = (e: React.MouseEvent) => {
    e.stopPropagation();
  };

  const header = (
    <Flex.Box
      x
      className="console-perf-section-header"
      onClick={() => setOpen(!open)}
      onKeyDown={handleKeyDown}
      role="button"
      tabIndex={0}
      aria-expanded={open}
      data-open={open}
    >
      <Icon.Caret.Right />
      <Text.Text level="small" className="console-perf-section-title" weight={500}>
        {title}
      </Text.Text>
      {secondaryText != null && (
        <Text.Text
          level="small"
          className="console-perf-section-header-value"
          data-status={secondaryStatus}
        >
          {secondaryText}
        </Text.Text>
      )}
      {actions != null && (
        <Flex.Box
          x
          onClick={handleActionsClick}
          className="console-perf-section-actions"
        >
          {actions}
        </Flex.Box>
      )}
    </Flex.Box>
  );

  return (
    <Flex.Box y className="console-perf-section">
      <WithTooltip tooltip={secondaryTooltip}>{header}</WithTooltip>
      {subheader}
      {open && children}
    </Flex.Box>
  );
};

export const Section = memo(SectionImpl);
