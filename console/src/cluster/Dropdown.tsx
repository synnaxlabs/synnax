// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Dropdown.css";

import { Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Dialog,
  Flex,
  List as CoreList,
  Select,
  Synnax,
  Text,
  User,
} from "@synnaxlabs/pluto";
import {
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";
import { useDispatch } from "react-redux";

import { ConnectionBadge } from "@/cluster/Badges";
import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { useSelect } from "@/cluster/selectors";
import { rename, setActive } from "@/cluster/slice";
import { EmptyAction } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Workspace } from "@/workspace";

interface ListItemProps extends CoreList.ItemProps<string> {
  validateName: (name: string) => boolean;
}

const ListItem = ({ validateName, ...rest }: ListItemProps): ReactElement | null => {
  const dispatch = useDispatch();
  const item = useSelect(rest.itemKey);
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const handleChange = (value: string) => {
    if (!validateName(value) || item == null) return;
    dispatch(rename({ key: item.key, name: value }));
  };

  if (item == null) return null;
  return (
    <CoreList.Item
      className={CSS(CSS.B("cluster-list-item"))}
      y
      selected={selected}
      onSelect={onSelect}
      gap="small"
      {...rest}
    >
      <Text.MaybeEditable
        id={`cluster-dropdown-${item.key}`}
        weight={500}
        value={item.name}
        onChange={handleChange}
        allowDoubleClick={false}
      />
      <Text.Text color={9} weight={450}>
        {item.host}:{item.port}
      </Text.Text>
    </CoreList.Item>
  );
};

export interface NoneConnectedBoundaryProps extends PropsWithChildren {}

export const NoneConnectedBoundary = ({
  children,
}: NoneConnectedBoundaryProps): ReactNode => {
  const client = Synnax.use();
  if (client != null) return children;
  return <NoneConnected />;
};

export interface NoneConnectedProps extends Flex.BoxProps<"div"> {}

export const NoneConnected = (props: NoneConnectedProps): ReactElement => {
  const placeLayout = Layout.usePlacer();

  const handleCluster: Text.TextProps["onClick"] = (e: MouseEvent) => {
    e.stopPropagation();
    placeLayout(CONNECT_LAYOUT);
  };

  return (
    <EmptyAction
      message="No Core connected."
      action="Connect a Core"
      onClick={handleCluster}
      {...props}
    />
  );
};

export const Dropdown = (): ReactElement => {
  const dispatch = useDispatch();
  const { data: u } = User.useRetrieve({});
  let text = u?.username;
  if (u?.firstName != "") text = `${u?.firstName}`;
  return (
    <Dialog.Frame>
      <Flex.Box x>
        <Flex.Box x pack>
          <Flex.Box
            square
            style={{
              minWidth: "4.5rem",
              height: "4.5rem",
              padding: "0.5rem",
              background: "var(--pluto-gray-l0)",
            }}
            bordered
            borderColor={6}
          >
            <div
              style={{
                background: User.avatar(u?.username ?? ""),
                width: "100%",
                height: "100%",
                borderRadius: "0.5rem",
              }}
            />
          </Flex.Box>
          <Dialog.Trigger
            contrast={2}
            hideCaret
            textColor={10}
            gap="small"
            weight={400}
          >
            {text}
          </Dialog.Trigger>
        </Flex.Box>
        <ConnectionBadge />
      </Flex.Box>
      <Dialog.Dialog bordered borderColor={6} style={{ padding: "1rem", width: 200 }}>
        <Button.Button
          onClick={() => {
            dispatch(Workspace.setActive(null));
            dispatch(setActive(null));
            dispatch(Layout.clearWorkspace());
          }}
          variant="text"
        >
          Sign Out
        </Button.Button>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
