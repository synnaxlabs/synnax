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
import { Status } from "@synnaxlabs/lyra/status";
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
  /** The status of the list query. An error shows in place of the items. */
  status?: status.Status;
}

/** What the create item of a {@link Table} uses to finish. */
export interface CreateContextValue {
  /** Closes the create item. */
  close: () => void;
  /** Lists a new item under the create item, which then starts empty for the next. */
  created: (key: record.Key) => void;
}

const CreateContext = createContext<CreateContextValue | null>(null);
CreateContext.displayName = "Settings.Create";

/** Returns the callbacks of the create item. Throws outside a create item. */
export const useCreate = (): CreateContextValue => {
  const ctx = use(CreateContext);
  if (ctx == null) throw new Error("Settings.useCreate must be inside a create item");
  return ctx;
};

export interface CreateProviderProps extends CreateContextValue, PropsWithChildren {}

/**
 * Gives its children the callbacks of a create item, for a create item that a page
 * draws in its list instead of at the top of the {@link Table}.
 */
export const CreateProvider = ({
  close,
  created,
  children,
}: CreateProviderProps): ReactElement => (
  <CreateContext value={{ close, created }}>{children}</CreateContext>
);

export interface TableProps<K extends record.Key, E extends record.Keyed<K>> {
  list: TableData<K, E>;
  columns: List.Column[];
  /** The plural name of the items, such as "labels". */
  noun: string;
  menu: Menu.ContextMenuProps["menu"];
  /** The create button's text, such as "Create label". Omit when the user cannot. */
  createLabel?: string;
  /**
   * An item that creates in place, used when every field it takes is a column. It
   * finishes through {@link useCreate}. Without it, the create button calls `onCreate`.
   */
  create?: ReactNode;
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

/** Returns the class of an item in a {@link Table}. An item created in it fades in. */
export const useItemClass = (key: record.Key): string =>
  CSS.cls(
    CSS.BE("settings-table", "item"),
    use(FreshContext).has(key) && CSS.BE("settings-table", "fresh"),
  );

/** Lays out the fields of a new item on the columns of its {@link Table}. */
export const CreateFrame = ({ children }: PropsWithChildren): ReactElement => (
  <div className={CSS.BE("settings-table", "create")}>
    <span className={CSS.BE("settings-table", "create-check")} />
    <List.Cells>{children}</List.Cells>
  </div>
);

/** Returns a key handler for a field of the create item: Enter saves, Escape closes. */
export const useCreateKeyDown = (save: () => void): ((e: KeyboardEvent) => void) => {
  const { close } = useCreate();
  return (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      save();
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  };
};

export interface CreateActionsProps {
  onSave: () => void;
  /** The status of the create request, which shows on the confirm button. */
  status?: status.Variant;
}

/** The confirm and cancel buttons of a create item, at the end of its last column. */
export const CreateActions = ({ onSave, status }: CreateActionsProps): ReactElement => {
  const { close } = useCreate();
  return (
    <Flex.Box
      x
      gap="small"
      justify="end"
      className={CSS.BE("settings-table", "actions")}
    >
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
        onClick={close}
      >
        <Icon.Close />
      </Button.Button>
    </Flex.Box>
  );
};

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
  create,
  onCreate,
  selected: initial,
  onShowSelected,
  children,
}: TableProps<K, E>): ReactElement => {
  const {
    data,
    getItem,
    subscribe,
    more,
    search,
    answered = true,
    status: query,
  } = list;
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
  const [fresh, setFresh] = useState<record.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const handleSearch = (next: string): void => {
    setTerm(next);
    search(next);
  };
  const markFresh = (key: record.Key): void =>
    setFresh((prev) => [key, ...prev.filter((k) => k !== key)]);
  const handleCreate = (): void => {
    if (create != null) {
      setCreating("open");
      return;
    }
    void onCreate?.()?.then((key) => {
      if (key != null) markFresh(key);
    });
  };
  const handleCreated = (key: record.Key): void => {
    markFresh(key);
    setRound((r) => r + 1);
  };
  const handleClose = (): void => setCreating((c) => (c == null ? null : "closing"));
  const present = new Map<record.Key, K>(data.map((k) => [k, k]));
  const freshShown = fresh.flatMap((k) => {
    const shown = present.get(k);
    return shown == null ? [] : [shown];
  });
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
        <Flex.Box x wrap align="center" justify="between" gap="medium">
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
          <div className={CSS.BE("settings-table", "scroll")}>
            <div className={CSS.BE("settings-table", "columns")}>
              <List.Columns columns={columns}>
                <List.Headings checkable />
                {creating != null && create != null && (
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
                      <CreateContext
                        value={{ close: handleClose, created: handleCreated }}
                      >
                        <Fragment key={round}>{create}</Fragment>
                      </CreateContext>
                    </div>
                  </div>
                )}
                <FreshContext value={freshSet}>
                  <List.Scroll onContextMenu={menuProps.open}>
                    <List.Items<K>
                      emptyContent={
                        query?.variant === "error" ? (
                          <Status.Summary
                            center
                            status={query}
                            className={CSS.BE("settings-table", "empty")}
                          />
                        ) : (
                          answered &&
                          creating == null && (
                            <Status.Summary
                              center
                              variant="disabled"
                              hideIcon
                              className={CSS.BE("settings-table", "empty")}
                            >
                              No {noun}
                            </Status.Summary>
                          )
                        )
                      }
                    >
                      {children}
                    </List.Items>
                  </List.Scroll>
                </FreshContext>
              </List.Columns>
            </div>
          </div>
        </Card>
      </Flex.Box>
    </Select.Frame>
  );
};
