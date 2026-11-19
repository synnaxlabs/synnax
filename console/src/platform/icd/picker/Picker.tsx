// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/icd/picker/Picker.css";

import { Button } from "@synnaxlabs/lyra/button";
import { context } from "@synnaxlabs/lyra/context";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Portal } from "@synnaxlabs/lyra/portal";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Text } from "@synnaxlabs/lyra/text";
import { type CrudeTimeSpan } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { HintContext } from "@/platform/icd/picker/hint";

const [ActiveContext, useActiveContext] = context.create<string>({
  displayName: "Picker.ActiveContext",
  providerName: "Picker.Root",
});

const [FrameContext, useFrameContext] = context.create<string>({
  displayName: "Picker.FrameContext",
  providerName: "Picker.Frame",
});

export interface RootProps extends PropsWithChildren {
  /** The key of the frame that shows. */
  value: string;
  /** Says what a pick becomes, under the title of each pane. */
  hint: string;
  className?: string;
}

/**
 * The frame of the dialog. Shows one {@link Frame} at a time and puts its
 * {@link Status} in the {@link Bar}.
 */
export const Root = ({ value, hint, className, children }: RootProps): ReactElement => (
  <Portal.Context>
    <ActiveContext value={value}>
      <HintContext value={hint}>
        <Modal.Frame className={CSS.cls(CSS.B("icd-picker"), className)}>
          {children}
        </Modal.Frame>
      </HintContext>
    </ActiveContext>
  </Portal.Context>
);

/** The bar above the frames: its children, then the status of the frame that shows. */
export const Bar = ({ children }: PropsWithChildren): ReactElement => (
  <Nav.Bar location="top" size="8rem" bordered className={CSS.BE("icd-picker", "bar")}>
    <Nav.Bar.Start>{children}</Nav.Bar.Start>
    <Nav.Bar.End align="center">
      <Portal.Out itemKey={useActiveContext("Picker.Bar")} />
    </Nav.Bar.End>
  </Nav.Bar>
);

export interface FrameProps extends PropsWithChildren {
  itemKey: string;
}

/**
 * The body of one tab. It mounts its children the first time it shows and keeps them
 * mounted while hidden, so a picker listens only once opened and keeps what it shows.
 */
export const Frame = ({ itemKey, children }: FrameProps): ReactElement | null => {
  const active = useActiveContext("Picker.Frame") === itemKey;
  const [opened, setOpened] = useState(active);
  if (active && !opened) setOpened(true);
  if (!opened) return null;
  return (
    <FrameContext value={itemKey}>
      <Flex.Box
        x
        grow
        empty
        className={CSS.cls(CSS.BE("icd-picker", "frame"), !active && CSS.M("hidden"))}
      >
        {children}
      </Flex.Box>
    </FrameContext>
  );
};

export interface StatusProps {
  status: string;
  loading: boolean;
  /** How long a listen takes. The progress bar fills over it. */
  duration?: CrudeTimeSpan;
  onListen: () => void;
}

/** The state of the listen of its {@link Frame}, shown in the {@link Bar}. */
export const Status = ({
  status,
  loading,
  duration,
  onListen,
}: StatusProps): ReactElement => (
  <Portal.In itemKey={useFrameContext("Picker.Status")}>
    <Flex.Box x align="center" gap="small">
      <Text.Text level="small" color={9}>
        {status}
      </Text.Text>
      <Button.Button
        variant="text"
        size="small"
        tooltip="Listen again"
        aria-label="Listen again"
        disabled={loading}
        onClick={onListen}
      >
        <Icon.Refresh />
      </Button.Button>
      {loading && (
        <Progress.Progress
          thin
          duration={duration}
          className={CSS.BE("icd-picker", "progress")}
        />
      )}
    </Flex.Box>
  </Portal.In>
);

/** The column that lists what the broker publishes. */
export const Side = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y empty className={CSS.BE("icd-picker", "side")}>
    {children}
  </Flex.Box>
);

export interface SearchProps extends Pick<
  Input.TextProps,
  "value" | "onChange" | "placeholder"
> {}

/** A field at the head of a {@link Side} that filters its list. */
export const Search = (props: SearchProps): ReactElement => (
  <Input.Text
    size="small"
    startContent={<Icon.Search />}
    className={CSS.BE("icd-picker", "search")}
    {...props}
  />
);

/** The part of a {@link Side} that lists what to pick. It fills the room left over. */
export const SideList = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y empty className={CSS.BE("icd-picker", "side-list")}>
    {children}
  </Flex.Box>
);

export interface SideActionProps extends Pick<
  Button.ButtonProps,
  "onClick" | "children"
> {}

/** A button at the foot of a {@link Side} that adds what its list does not hold. */
export const SideAction = ({ children, ...rest }: SideActionProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    className={CSS.BE("icd-picker", "side-action")}
    {...rest}
  >
    <Icon.Add />
    {children}
  </Button.Button>
);

export interface NoticeProps extends Pick<Empty.ActionProps, "message"> {}

/** Says why a list or the pane shows nothing. */
export const Notice = ({ message }: NoticeProps): ReactElement => (
  <Empty.Action message={message} className={CSS.BE("icd-picker", "notice")} />
);

/** The area that shows the values of what the side selects. */
export const Pane = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box y grow empty className={CSS.BE("icd-picker", "pane")}>
    {children}
  </Flex.Box>
);

/**
 * @returns the label of the button that adds the picks, from the count of each noun.
 * With no picks, it names idle.
 */
const addLabel = (counts: [noun: string, count: number][], idle: string): string => {
  const totals = new Map<string, number>();
  for (const [noun, count] of counts)
    if (count > 0) totals.set(noun, (totals.get(noun) ?? 0) + count);
  if (totals.size === 0) return `Add ${pluralize(idle)}`;
  const parts = [...totals].map(([noun, count]) => pluralize(noun, count, true));
  return `Add ${parts.join(" and ")}`;
};

export interface FooterProps extends PropsWithChildren {
  /** The count of each noun picked, such as ["field", 2]. Counts of a noun add up. */
  counts: [noun: string, count: number][];
  /** Names what the add button adds while nothing is picked. */
  idle: string;
  onAdd: () => void;
  onCancel: () => void;
}

/** The footer of the dialog: a summary of the picks, then Cancel and the add button. */
export const Footer = ({
  counts,
  idle,
  onAdd,
  onCancel,
  children,
}: FooterProps): ReactElement => (
  <Modal.Footer bordered className={CSS.BE("icd-picker", "footer")}>
    <Nav.Bar.Start>{children}</Nav.Bar.Start>
    <Nav.Bar.End x align="center" gap="small">
      <Button.Button variant="outlined" onClick={onCancel}>
        Cancel
      </Button.Button>
      <Button.Button
        variant="filled"
        disabled={counts.every(([, count]) => count === 0)}
        onClick={onAdd}
      >
        {addLabel(counts, idle)}
      </Button.Button>
    </Nav.Bar.End>
  </Modal.Footer>
);

/** A checkbox that holds the place of one still loading. */
export const CheckSkeleton = (): ReactElement => (
  <Input.Checkbox size="small" value={false} onChange={() => {}} disabled />
);
