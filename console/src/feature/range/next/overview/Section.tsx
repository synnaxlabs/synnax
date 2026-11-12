// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, type ReactNode, useState } from "react";

import { CSS } from "@/platform/css";
import { Section as PlatformSection } from "@/platform/section";

interface SectionProps {
  title: string;
  icon: Icon.ReactElement;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

/**
 * An outlined box with its title and actions in a bar at the top. Clicking the title
 * collapses the box to its bar, and any action opens it again.
 */
export const Section = ({
  title,
  icon,
  actions,
  children,
  className,
}: SectionProps): ReactElement => {
  const [open, setOpen] = useState(true);
  return (
    <PlatformSection.Section
      y
      empty
      className={CSS.cls(
        !open && CSS.BEM("range-overview", "section", "collapsed"),
        className,
      )}
    >
      <PlatformSection.Header level="small" bordered={open}>
        <Button.Button
          variant="text"
          size="small"
          level="p"
          gap="small"
          textColor={10}
          aria-expanded={open}
          onClick={() => setOpen((prev) => !prev)}
        >
          <Text.Text el="span" color={9}>
            {icon}
          </Text.Text>
          <Text.Text el="span" weight={500} color={10}>
            {title}
          </Text.Text>
          <Text.Text el="span" color={8} className={CSS.BE("range-overview", "caret")}>
            <Icon.Caret.Down />
          </Text.Text>
        </Button.Button>
        {actions != null && (
          <Flex.Box x align="center" gap="small" onClickCapture={() => setOpen(true)}>
            {actions}
          </Flex.Box>
        )}
      </PlatformSection.Header>
      {open && children}
    </PlatformSection.Section>
  );
};

interface AddButtonProps {
  tooltip: string;
  onClick: () => void;
}

export const AddButton = ({ tooltip, onClick }: AddButtonProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    textColor={9}
    tooltip={tooltip}
    tooltipLocation="left"
    onClick={onClick}
  >
    <Icon.Add />
  </Button.Button>
);

interface AddLineProps {
  onClick: () => void;
  children: string;
}

/** Stands in for an empty section: one muted line that adds its first item. */
export const AddLine = ({ onClick, children }: AddLineProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    textColor={9}
    gap="small"
    className={CSS.BE("range-overview", "add")}
    onClick={onClick}
  >
    <Icon.Add />
    {children}
  </Button.Button>
);

interface FieldProps {
  name: ReactNode;
  children: ReactNode;
  /** Shows at the end of the field at all times. */
  note?: ReactNode;
  /** Shows at the end of the field on hover and focus. */
  actions?: ReactNode;
}

export const Field = ({ name, children, note, actions }: FieldProps): ReactElement => (
  <div className={CSS.BE("range-overview", "field")}>
    {name}
    {children}
    <Flex.Box x align="center" gap="small">
      {note}
      {actions != null && (
        <Flex.Box
          x
          align="center"
          className={CSS.BE("range-overview", "field-actions")}
        >
          {actions}
        </Flex.Box>
      )}
    </Flex.Box>
  </div>
);

export const FieldName = ({ children }: { children: ReactNode }): ReactElement => (
  <Text.Text
    color={9}
    overflow="fade"
    className={CSS.BE("range-overview", "field-name")}
  >
    {children}
  </Text.Text>
);

interface RemoveButtonProps {
  onClick: () => void;
}

export const RemoveButton = ({ onClick }: RemoveButtonProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    textColor={9}
    tooltip="Remove"
    onClick={onClick}
  >
    <Icon.Close />
  </Button.Button>
);
