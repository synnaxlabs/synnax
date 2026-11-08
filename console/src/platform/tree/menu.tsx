// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource, type Synnax as Client } from "@synnaxlabs/client";
import { type Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { unique } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { useStore } from "react-redux";

import { ContextMenu } from "@/platform/context-menu";
import { Errors } from "@/platform/errors";
import { Panel } from "@/platform/panel";
import { findAncestors, findContainedKeys } from "@/platform/tree/ancestry";
import { type DefaultContextMenuProps } from "@/platform/tree/DefaultContextMenu";
import { type Item } from "@/platform/tree/item";
import { MultipleSelectionContextMenu } from "@/platform/tree/MultipleSelectionContextMenu";
import { type ResultNodes } from "@/platform/tree/search";
import { type Store } from "@/platform/tree/store";
import { type ContextMenuProps, type TreeState } from "@/platform/tree/types";
import { useResources } from "@/platform/tree/useResource";
import { type Action, type State } from "@/session/store";

const FallbackContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

interface SelectionContextMenuProps extends ContextMenuProps {
  menu: FC<ContextMenuProps>;
}

// A resource deleted while its menu is open stays in the selection. A menu reads every
// selected resource as present.
const SelectionContextMenu = ({ menu: M, ...props }: SelectionContextMenuProps) => {
  const resources = useResources(props.state, props.selection.ids);
  if (resources.length < props.selection.ids.length) return <FallbackContextMenu />;
  return <M {...props} />;
};

export interface UseMenuProps {
  client: Client;
  store: Store;
  root: resource.ID;
  results?: ResultNodes;
  selected: string[];
  state: TreeState;
  /** The menu for a right-click on empty space. */
  rootContextMenu: FC<DefaultContextMenuProps>;
  resolveItem: (type: resource.Type) => Item;
  handleError: Status.ErrorHandler;
}

/** Returns the context menu of a tree: for the selection, or for empty space. */
export const useMenu = ({
  client,
  store,
  root,
  results,
  selected,
  state,
  rootContextMenu: RootContextMenu,
  resolveItem,
  handleError,
}: UseMenuProps): ((props: Menu.ContextMenuMenuProps) => ReactElement) => {
  const openTab = Panel.useOpenTab();
  const addStatus = Status.useAdder();
  const session = useStore<State, Action>();

  const render = ({ keys: clicked }: Menu.ContextMenuMenuProps): ReactElement => {
    if (clicked.length === 0) return <RootContextMenu root={root} state={state} />;
    const rightClickedButNotSelected = clicked.find((v) => !selected.includes(v));
    // A right-click outside the selection shows a menu for that item alone. Otherwise
    // the selection state gives the keys, since the virtual tree can lack the DOM
    // elements.
    const keys =
      rightClickedButNotSelected != null ? [rightClickedButNotSelected] : selected;
    const ids = keys.map((key) => resource.parseID(key));
    // A selected node's own children do not count, since an action on the node carries
    // them along. Flat search results show both at the same depth.
    const contained = findContainedKeys(store, keys, results?.ancestors);
    const top = Base.getAllNodesOfMinDepth(
      Base.filterShape(state.shape, (key) => keys.includes(key) && !contained.has(key)),
    );
    const parents = unique.by(
      top.map((key) => findAncestors(store, key, root, results?.paths)[0]),
      (id) => resource.idToString(id),
    );
    const props: ContextMenuProps = {
      selection: {
        rootID: root,
        parentID: parents.length === 1 ? parents[0] : null,
        ids,
        top: top.map((key) => resource.parseID(key)),
      },
      state,
      client,
      store: session,
      openTab,
      addStatus,
      handleError,
    };
    const allSameType = ids.every((id) => id.type === ids[0].type);
    const M = allSameType
      ? resolveItem(ids[0].type).ContextMenu
      : MultipleSelectionContextMenu;
    if (M == null) return <FallbackContextMenu />;
    return <SelectionContextMenu menu={M} {...props} />;
  };

  return (props) => (
    <Errors.SuspenseBoundary loading={null}>{render(props)}</Errors.SuspenseBoundary>
  );
};
