// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Icon, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement, type ReactNode, useState } from "react";

import { WithTooltip } from "@/perf/components/WithTooltip";
import { STATUS_COLORS, TEXT_HEADER_COLOR } from "@/perf/constants";
import { type Status } from "@/perf/types";

interface SectionProps {
  title: string;
  secondaryText?: ReactNode;
  secondaryStatus?: Status;
  secondaryTooltip?: string;
  defaultOpen?: boolean;
  children?: ReactNode;
}

const SectionImpl = ({
  title,
  secondaryText,
  secondaryStatus,
  secondaryTooltip,
  defaultOpen = true,
  children,
}: SectionProps): ReactElement => {
  const [open, setOpen] = useState(defaultOpen);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      setOpen(!open);
    }
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
      <Text.Text level="small" color={TEXT_HEADER_COLOR} weight={500}>
        {title}
      </Text.Text>
      {secondaryText != null && (
        <Text.Text
          level="small"
          className="console-perf-section-header-value"
          color={secondaryStatus != null ? STATUS_COLORS[secondaryStatus] : TEXT_HEADER_COLOR}
        >
          {secondaryText}
        </Text.Text>
      )}
    </Flex.Box>
  );

  return (
    <Flex.Box y className="console-perf-section">
      <WithTooltip tooltip={secondaryTooltip}>
        {header}
      </WithTooltip>
      {open && children}
    </Flex.Box>
  );
};

export const Section = memo(SectionImpl);
