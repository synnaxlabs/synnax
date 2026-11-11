// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/common.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { Vcs } from "@/platform/vcs";

export interface SectionHeaderProps extends PropsWithChildren {
  title: string;
  /** Shown dimmed after the title, such as a count. */
  detail?: ReactNode;
  /** Shown before the title, such as a checkbox that checks every item. */
  start?: ReactNode;
}

/** A quiet header that splits a toolbar list into sections. */
export const SectionHeader = ({
  title,
  detail,
  start,
  children,
}: SectionHeaderProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    justify="between"
    shrink={false}
    className={CSS.BE("vcs", "section-header")}
  >
    <Flex.Box x align="center" gap="small" className={CSS.BE("vcs", "section-title")}>
      {start}
      <Text.Text level="small" weight={450} color={9}>
        {title}
      </Text.Text>
      {detail != null && (
        <Text.Text level="small" color={8} className={CSS.BE("vcs", "count")}>
          {detail}
        </Text.Text>
      )}
    </Flex.Box>
    {children != null && (
      <Flex.Box x align="center" gap="tiny" shrink={false}>
        {children}
      </Flex.Box>
    )}
  </Flex.Box>
);

export interface ResourceItemProps extends Omit<
  List.ItemProps<string>,
  "children" | "start"
> {
  type: string;
  name: string;
  /** Shown before the type icon, such as a checkbox. */
  start?: ReactNode;
  /** Shown at the end of the item. */
  end?: ReactNode;
  /** Shown under the name, lined up with it. */
  below?: ReactNode;
}

/** A list item for one resource: its type icon and its name. */
export const ResourceItem = ({
  type,
  name,
  start,
  end,
  below,
  className,
  ...rest
}: ResourceItemProps): ReactElement => {
  const TypeIcon = Vcs.typeIcon(type);
  return (
    <List.Item
      className={CSS.cls(
        CSS.BE("vcs", "item"),
        start != null && CSS.BEM("vcs", "item", "start"),
        className,
      )}
      y
      gap="tiny"
      {...rest}
    >
      <Flex.Box x align="center" gap="small" className={CSS.BE("vcs", "item-line")}>
        {start}
        <TypeIcon className={CSS.BE("vcs", "type-icon")} />
        <Text.Text overflow="fade" className={CSS.BE("vcs", "name")}>
          {name}
        </Text.Text>
        {end != null && (
          <Flex.Box x align="center" gap="tiny" className={CSS.BE("vcs", "item-end")}>
            {end}
          </Flex.Box>
        )}
      </Flex.Box>
      {below != null && <div className={CSS.BE("vcs", "item-below")}>{below}</div>}
    </List.Item>
  );
};

export interface ListProps extends PropsWithChildren {
  className?: string;
  onContextMenu?: Flex.BoxProps["onContextMenu"];
}

/** A scrolling list inset from the toolbar edges, like the Console's trees. */
export const ItemList = ({
  className,
  children,
  onContextMenu,
}: ListProps): ReactElement => (
  <Flex.Box
    y
    empty
    grow
    className={CSS.cls(CSS.BE("vcs", "list"), className)}
    onContextMenu={onContextMenu}
  >
    {children}
  </Flex.Box>
);

export interface ViewHeaderProps extends PropsWithChildren {
  title: string;
  onBack: () => void;
}

/** The header of a view that replaces the tabs, such as a plan, with a way back. */
export const ViewHeader = ({ title, onBack, children }: ViewHeaderProps) => (
  <Flex.Box
    x
    align="center"
    justify="between"
    shrink={false}
    className={CSS.BE("vcs", "view-header")}
  >
    <Flex.Box x align="center" gap="small" className={CSS.BE("vcs", "view-title")}>
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        tooltip="Back"
        aria-label="Back"
        onClick={onBack}
      >
        <Icon.Arrow.Left />
      </Button.Button>
      <Text.Text weight={450} overflow="fade">
        {title}
      </Text.Text>
    </Flex.Box>
    {children}
  </Flex.Box>
);

export interface EmptyProps {
  message: string;
  description?: string;
}

/** Fills an empty section with a short message. */
export const Empty = ({ message, description }: EmptyProps): ReactElement => (
  <Flex.Box y center gap="tiny" className={CSS.BE("vcs", "empty")}>
    <Text.Text status="disabled">{message}</Text.Text>
    {description != null && (
      <Text.Text level="small" color={8} className={CSS.BE("vcs", "empty-text")}>
        {description}
      </Text.Text>
    )}
  </Flex.Box>
);

export interface LoadingProps {
  className?: string;
}

/** Shows that a section is loading. */
export const Loading = ({ className }: LoadingProps): ReactElement => (
  <Status.Summary
    variant="loading"
    level="small"
    message="Loading"
    className={CSS.cls(CSS.BE("vcs", "status"), className)}
  />
);

export interface FooterProps extends PropsWithChildren {
  className?: string;
}

/** The bar at the bottom of a toolbar view that holds its primary action. */
export const Footer = ({ className, children }: FooterProps): ReactElement => (
  <Flex.Box
    y
    gap="small"
    shrink={false}
    className={CSS.cls(CSS.BE("vcs", "footer"), className)}
  >
    {children}
  </Flex.Box>
);
