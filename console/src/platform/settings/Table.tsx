// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/settings/Table.css";

import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type record } from "@synnaxlabs/x";
import {
  createContext,
  Fragment,
  type KeyboardEvent,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  use,
  useEffect,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { Card } from "@/platform/settings/Page";

/** The items of a {@link Table}, from a Flux list or a static list. */
export interface TableData<
  K extends record.Key,
  E extends record.Keyed<K>,
> extends Pick<Select.FrameProps<K, E>, "getItem" | "subscribe"> {
  data: K[];
  search: (term: string) => void;
  more?: () => void;
  /** False until the first answer, so the empty text does not flash while loading. */
  answered?: boolean;
}

/** Props for the item that creates items in place in a {@link Table}. */
export interface CreateItemProps<K extends record.Key> {
  /** Closes the item. */
  onClose: () => void;
  /** Lists a new item under the create item, which then starts empty for the next. */
  onCreated: (key: K) => void;
}

export interface TableProps<K extends record.Key, E extends record.Keyed<K>> {
  list: TableData<K, E>;
  columns: List.Column[];
  /** The plural name of the items, such as "labels". */
  noun: string;
  menu: Menu.ContextMenuProps["menu"];
  /** The create button's text, such as "Create label". Omit when the user cannot. */
  createLabel?: string;
  /**
   * Renders an item that creates in place, used when every field it takes is a column.
   * Without it, the create button calls `onCreate`.
   */
  renderCreate?: (props: CreateItemProps<K>) => ReactNode;
  /** Opens a modal that creates an item, and resolves with its key or null. */
  onCreate?: () => Promise<K | null> | void;
  /** An item to select and scroll to once the list holds it. */
  selected?: K;
  /** Called once the list shows the item in `selected`. */
  onShowSelected?: () => void;
  children: List.ItemsProps<K>["children"];
}

interface ScrollToProps<K extends record.Key> {
  data: K[];
  itemKey?: K;
  onShow?: () => void;
}

const ScrollTo = <K extends record.Key>({
  data,
  itemKey,
  onShow,
}: ScrollToProps<K>): null => {
  const { scrollToIndex } = List.useScroller<K>();
  const index = itemKey == null ? -1 : data.indexOf(itemKey);
  useEffect(() => {
    if (index === -1) return;
    scrollToIndex(index);
    onShow?.();
  }, [itemKey, index !== -1]);
  return null;
};

const FreshContext = createContext<ReadonlySet<record.Key>>(new Set());
FreshContext.displayName = "Settings.Fresh";

/** Returns the class of an item created in this table, which fades it in. */
export const useFreshClass = (key: record.Key): string | undefined =>
  use(FreshContext).has(key) ? CSS.BE("settings-table", "fresh") : undefined;

/** Lays out the fields of a new item on the columns of its {@link Table}. */
export const CreateFrame = ({ children }: PropsWithChildren): ReactElement => (
  <div className={CSS.BE("settings-table", "create")}>
    <span className={CSS.BE("settings-table", "create-check")} />
    <List.Cells>{children}</List.Cells>
  </div>
);

/** Returns a key handler that saves on Enter and closes on Escape. */
export const createKeyDown =
  (save: () => void, close: () => void) =>
  (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  };

export interface CreateActionsProps {
  onSave: () => void;
  onCancel: () => void;
  /** The status of the create request, which shows on the confirm button. */
  status?: status.Variant;
}

/** The confirm and cancel buttons of a create item, at the end of its last column. */
export const CreateActions = ({
  onSave,
  onCancel,
  status,
}: CreateActionsProps): ReactElement => (
  <Flex.Box x gap="small" justify="end" className={CSS.BE("settings-table", "actions")}>
    <Button.Button
      variant="filled"
      size="small"
      tooltip="Create"
      aria-label="Create"
      status={status === "loading" ? "loading" : undefined}
      onClick={onSave}
    >
      <Icon.Check />
    </Button.Button>
    <Button.Button
      variant="text"
      size="small"
      tooltip="Cancel"
      aria-label="Cancel"
      onClick={onCancel}
    >
      <Icon.Close />
    </Button.Button>
  </Flex.Box>
);

/**
 * A searchable list of items on the columns of a Settings page, with a create action
 * and a context menu for the selected items. An item created in place lists at the top
 * until the page closes, so it does not jump to its sorted place.
 */
export const Table = <K extends record.Key, E extends record.Keyed<K>>({
  list,
  columns,
  noun,
  menu,
  createLabel,
  renderCreate,
  onCreate,
  selected: initial,
  onShowSelected,
  children,
}: TableProps<K, E>): ReactElement => {
  const { data, getItem, subscribe, more, search, answered = true } = list;
  const [term, setTerm] = useState("");
  const [selected, setSelected] = useState<K[]>(initial == null ? [] : [initial]);
  const [prevInitial, setPrevInitial] = useState(initial);
  if (initial !== prevInitial) {
    setPrevInitial(initial);
    if (initial != null) setSelected([initial]);
  }
  const [creating, setCreating] = useState<"open" | "closing" | null>(null);
  // A new round remounts the create item, so it starts empty after each create.
  const [round, setRound] = useState(0);
  const [fresh, setFresh] = useState<K[]>([]);
  const menuProps = Menu.useContextMenu();
  const handleSearch = (next: string): void => {
    setTerm(next);
    search(next);
  };
  const markFresh = (key: K): void =>
    setFresh((prev) => [key, ...prev.filter((k) => k !== key)]);
  const handleCreate = (): void => {
    if (renderCreate != null) {
      setCreating("open");
      return;
    }
    void onCreate?.()?.then((key) => {
      if (key != null) markFresh(key);
    });
  };
  const handleCreated = (key: K): void => {
    markFresh(key);
    setRound((r) => r + 1);
  };
  const handleClose = (): void => setCreating((c) => (c == null ? null : "closing"));
  const present = new Set(data);
  const freshShown = fresh.filter((k) => present.has(k));
  const freshSet = new Set<record.Key>(freshShown);
  const ordered =
    freshShown.length === 0
      ? data
      : [...freshShown, ...data.filter((k) => !freshSet.has(k))];
  return (
    <Select.Frame<K, E>
      multiple
      data={ordered}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      value={selected}
      onChange={setSelected}
    >
      <Menu.ContextMenu menu={menu} {...menuProps} />
      <ScrollTo data={ordered} itemKey={initial} onShow={onShowSelected} />
      <Flex.Box y gap="medium" className={CSS.B("settings-table")}>
        <Flex.Box x align="center" justify="between" gap="medium">
          <Input.Text
            value={term}
            onChange={handleSearch}
            placeholder={`Search ${noun}`}
            startContent={<Icon.Search />}
            className={CSS.BE("settings-table", "search")}
          />
          {createLabel != null && (
            <Button.Button
              variant="filled"
              onClick={handleCreate}
              disabled={creating === "open"}
            >
              <Icon.Add />
              {createLabel}
            </Button.Button>
          )}
        </Flex.Box>
        <Card flush>
          <List.Columns columns={columns}>
            <List.Headings checkable />
            {creating != null && renderCreate != null && (
              <div
                className={CSS.cls(
                  CSS.BE("settings-table", "reveal"),
                  creating === "closing" &&
                    CSS.BEM("settings-table", "reveal", "closing"),
                )}
                onAnimationEnd={(e) => {
                  if (creating === "closing" && e.target === e.currentTarget)
                    setCreating(null);
                }}
              >
                <div>
                  <Fragment key={round}>
                    {renderCreate({ onClose: handleClose, onCreated: handleCreated })}
                  </Fragment>
                </div>
              </div>
            )}
            <FreshContext value={freshSet}>
              <List.Scroll onContextMenu={menuProps.open}>
                <List.Items<K>
                  emptyContent={
                    answered &&
                    creating == null && (
                      <Flex.Box center className={CSS.BE("settings-table", "empty")}>
                        <Text.Text color={9}>No {noun}</Text.Text>
                      </Flex.Box>
                    )
                  }
                >
                  {children}
                </List.Items>
              </List.Scroll>
            </FreshContext>
          </List.Columns>
        </Card>
      </Flex.Box>
    </Select.Frame>
  );
};
