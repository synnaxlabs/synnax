// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  ContextMenu as PContextMenu,
  Flex,
  Header,
  Icon,
  type Input,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { CONNECT_LAYOUT } from "@/cluster/Connect";
import { Item } from "@/cluster/list/Item";
import { useSelectMany } from "@/cluster/selectors";
import { remove } from "@/cluster/slice";
import { ContextMenu as CMenu } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";

export interface ListProps
  extends Input.Control<string | undefined>,
    Omit<Flex.BoxProps, "onChange"> {}

export const List = ({ value, onChange, ...rest }: ListProps): ReactElement => {
  const contextMenuProps = PContextMenu.use();
  const dispatch = useDispatch();
  const allClusters = useSelectMany().sort((a, b) => a.name.localeCompare(b.name));
  const keys = useMemo(() => allClusters.map((c) => c.key), [allClusters]);
  const addStatus = Status.useAdder();

  const validateName = useCallback(
    (name: string): boolean => {
      const allNames = allClusters.map((c) => c.name);
      if (!allNames.includes(name)) return true;
      addStatus({
        variant: "error",
        message: `Cannot rename Core to ${name}`,
        description: `A Core with name "${name}" already exists.`,
      });
      return false;
    },
    [allClusters, addStatus],
  );

  const handleRemove = (key: string): void => {
    dispatch(remove(key));
  };

  const handleLink = Link.useCopyToClipboard();

  const contextMenu = useCallback(
    ({ keys: [key] }: PContextMenu.MenuProps): ReactElement => {
      if (key == null) return <Layout.DefaultContextMenu />;
      return (
        <>
          <CMenu.RenameItem
            onClick={() => Text.edit(`cluster-dropdown-${key}`)}
            showBottomDivider
          />
          <PContextMenu.Item onClick={() => handleRemove(key)} showBottomDivider>
            <Icon.Delete />
            Remove
          </PContextMenu.Item>
          <Link.CopyContextMenuItem
            onClick={() =>
              handleLink({
                clusterKey: key,
                name: allClusters.find((c) => c.key === key)?.name ?? "",
              })
            }
            showBottomDivider
          />
          <CMenu.ReloadConsoleItem />
        </>
      );
    },
    [handleRemove],
  );

  const placeLayout = Layout.usePlacer();

  return (
    <Select.Frame data={keys} value={value} onChange={onChange} itemHeight={54}>
      <Flex.Box y bordered grow empty {...rest}>
        <PContextMenu.ContextMenu menu={contextMenu} {...contextMenuProps} />
        <Header.Header gap="small" x style={{ padding: "0.666rem" }}>
          <Header.Title level="h4" color={11}>
            <Icon.Cluster />
            Cores
          </Header.Title>
          <Button.Button onClick={() => placeLayout(CONNECT_LAYOUT)} variant="filled">
            <Icon.Add />
          </Button.Button>
        </Header.Header>
        <Flex.Box empty onContextMenu={contextMenuProps.open} grow>
          {keys.map((key, i) => (
            <Item key={key} index={i} itemKey={key} validateName={validateName} />
          ))}
        </Flex.Box>
      </Flex.Box>
    </Select.Frame>
  );
};
