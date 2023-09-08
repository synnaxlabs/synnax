// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type FC,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MouseEventHandler,
} from "react";

import { ontology } from "@synnaxlabs/client";
import {
  Dropdown,
  Input,
  List,
  Triggers,
  componentRenderProp,
  Button,
  useDebouncedCallback,
  createFilterTransform,
  Status,
  Haul,
  CSS as PCSS,
  Tooltip,
  Mosaic,
  Divider,
} from "@synnaxlabs/pluto";
import { TimeSpan, type AsyncTermSearcher } from "@synnaxlabs/x";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Notifications } from "@/palette/Notifications";
import { TooltipContent } from "@/palette/Tooltip";
import { type Mode, type TriggerConfig } from "@/palette/types";
import { type Service } from "@/resources/service";
import { type RootStore } from "@/store";

import "@/palette/Palette.css";

export interface PaletteProps {
  searcher: AsyncTermSearcher<string, string, ontology.Resource>;
  commands: Command[];
  resourceTypes: Record<string, Service>;
  triggers: TriggerConfig;
  commandSymbol: string;
}

type Entry = Command | ontology.Resource;
type Key = string;

export const Palette = ({
  commands,
  searcher,
  resourceTypes,
  triggers,
  commandSymbol,
}: PaletteProps): ReactElement => {
  const dropdown = Dropdown.use();

  const [mode, setMode] = useState<Mode>("resource");

  const notifications = Status.useNotifications({ expiration: TimeSpan.seconds(5) });

  const store = useStore() as RootStore;
  const placeLayout = Layout.usePlacer();
  const handleSelect: List.SelectorProps<Key, Entry>["onChange"] = useCallback(
    ([key]: Key[], { entries: [entry] }) => {
      dropdown.close();
      if (mode === "command") {
        (entry as Command).onSelect({
          store,
          placeLayout,
        });
      } else {
        const id = new ontology.ID(key);
        const t = resourceTypes[id.type];
        t?.onSelect({
          store,
          placeLayout,
          selected: entry,
        });
      }
    },
    [mode, commands, dropdown.close]
  );

  const showDropdown = dropdown.visible || notifications.statuses.length > 0;
  const showDivider = notifications.statuses.length > 0 && dropdown.visible;

  return (
    <List.List>
      <Tooltip.Dialog location="bottom" hide={showDropdown}>
        <TooltipContent triggers={triggers} />
        <Dropdown.Dialog
          ref={dropdown.ref}
          visible={showDropdown}
          className={CSS.B("palette")}
          location="bottom"
          matchTriggerWidth
        >
          <PaletteInput
            mode={mode}
            setMode={setMode}
            searcher={searcher}
            commandSymbol={commandSymbol}
            triggerConfig={triggers}
            commands={commands}
            visible={dropdown.visible}
            open={dropdown.open}
          />
          <>
            {dropdown.visible && (
              <PaletteList
                mode={mode}
                resourceTypes={resourceTypes}
                onSelect={handleSelect}
                visible={dropdown.visible}
              />
            )}
            {showDivider && <Divider.Divider direction="x" />}
            <Notifications {...notifications} />
          </>
        </Dropdown.Dialog>
      </Tooltip.Dialog>
    </List.List>
  );
};
export interface PaletteInputProps
  extends Pick<Dropdown.UseReturn, "visible" | "open"> {
  mode: Mode;
  visible: boolean;
  setMode: (mode: Mode) => void;
  searcher: AsyncTermSearcher<string, string, ontology.Resource>;
  commandSymbol: string;
  triggerConfig: TriggerConfig;
  commands: Command[];
}

const canDrop: Haul.CanDrop = ({ items }) =>
  items.length === 1 && items[0].type === Mosaic.HAUL_TYPE;

const TYPE_TO_SEARCH = (
  <Status.Text.Centered level="h4" variant="disabled" hideIcon>
    Type to search
  </Status.Text.Centered>
);

export const PaletteInput = ({
  mode,
  setMode,
  visible,
  open,
  searcher,
  commandSymbol,
  triggerConfig,
  commands,
}: PaletteInputProps): ReactElement => {
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const { setSourceData, setTransform, deleteTransform, setEmptyContent } =
    List.useContext<Key, Entry>();

  useEffect(() => {
    if (!visible) inputRef.current?.blur();
  }, [visible, setMode]);

  const handleBlur = useCallback(() => setValue(""), []);

  const updateMode = useCallback(
    (nextMode: Mode) => {
      setMode(nextMode);
      if (nextMode === "command") setSourceData(commands);
      else {
        setSourceData([]);
        setEmptyContent(TYPE_TO_SEARCH);
      }
    },
    [setMode, setSourceData, commands]
  );

  const handleSearch = useDebouncedCallback(
    (term: string) => {
      if (term.length === 0) return setEmptyContent(TYPE_TO_SEARCH);
      searcher
        .search(term)
        .then((d) => {
          if (d.length === 0)
            setEmptyContent(
              <Status.Text.Centered level="h4" variant="disabled" hideIcon>
                No resources found
              </Status.Text.Centered>
            );
          setSourceData(d);
        })
        .catch((e) => {
          setEmptyContent(
            <Status.Text.Centered level="h4" variant="error">
              {e.message}
            </Status.Text.Centered>
          );
        });
    },
    250,
    [searcher, setSourceData]
  );

  const debouncedSetTransform = useDebouncedCallback(setTransform, 250, []);

  const handleFilter = useCallback(
    (term: string) => {
      if (term.length === 0) deleteTransform("filter");
      else debouncedSetTransform("filter", createFilterTransform({ term }));
    },
    [debouncedSetTransform, deleteTransform]
  );

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start" || visible) return;
      const mode = Triggers.determineMode(triggerConfig, triggers);
      if (mode === "command") {
        handleChange(commandSymbol);
        updateMode("command");
      } else {
        handleChange("");
        updateMode("resource");
      }
      inputRef.current?.focus();
    },
    [visible, triggerConfig.command, commandSymbol, updateMode]
  );

  const triggers = useMemo(
    () => Triggers.flattenConfig(triggerConfig),
    [triggerConfig]
  );

  Triggers.use({ triggers, callback: handleTrigger });

  const handleChange = useCallback(
    (value: string) => {
      const nextMode = value.startsWith(commandSymbol) ? "command" : "resource";
      if (mode !== nextMode) updateMode(nextMode);
      if (nextMode === "command") handleFilter(value.slice(1));
      else handleSearch(value);
      setValue(value);
    },
    [mode, setMode, commandSymbol, handleFilter, handleSearch]
  );

  const placer = Layout.usePlacer();
  const d = useDispatch();

  const { onDragOver, onDrop } = Haul.useDrop({
    type: "Palette",
    canDrop,
    onDrop: useCallback(
      ({ items: [item] }) => {
        const { key } = placer(Layout.createMosaicWindow());
        d(
          Layout.moveMosaicTab({
            windowKey: key,
            key: 1,
            tabKey: item.key as string,
            loc: "center",
          })
        );
        return [item];
      },
      [placer]
    ),
  });

  const dragging = Haul.useDraggingState();

  return (
    <Input.Text
      className={CSS(CSS.BE("palette", "input"), PCSS.dropRegion(canDrop(dragging)))}
      ref={inputRef}
      placeholder="Search Synnax"
      onDragOver={onDragOver}
      onDrop={onDrop}
      centerPlaceholder
      onBlur={handleBlur}
      onFocus={open}
      onChange={handleChange}
      value={value}
      autoComplete="off"
    />
  );
};

export interface PalleteListProps extends Pick<Dropdown.DialogProps, "visible"> {
  mode: Mode;
  onSelect: List.SelectorProps<Key, Entry>["onChange"];
  resourceTypes: Record<string, ResourceType>;
}

const PaletteList = ({
  mode,
  onSelect,
  visible,
  resourceTypes,
}: PalleteListProps): ReactElement => {
  const item = useMemo(() => {
    const Item = (
      mode === "command" ? CommandListItem : createResourceListItem(resourceTypes)
    ) as FC<List.ItemProps<string, ontology.Resource | Command>>;
    return componentRenderProp(Item);
  }, [mode, resourceTypes]);
  return (
    <>
      <List.Selector value={[]} onChange={onSelect} allowMultiple={false} />
      {visible && <List.Hover />}
      <List.Core.Virtual className={CSS.BE("palette", "list")} itemHeight={27}>
        {item}
      </List.Core.Virtual>
    </>
  );
};

export const CommandListItem = ({
  entry: { icon, name, key },
  hovered,
  onSelect,
  ...props
}: List.ItemProps<string, Command>): ReactElement => {
  const handleSelect: MouseEventHandler = (e): void => {
    e.stopPropagation();
    onSelect?.(key);
  };
  return (
    <Button.Button
      startIcon={icon}
      onClick={handleSelect}
      variant="text"
      className={CSS(
        CSS.BE("palette", "item"),
        hovered && CSS.BEM("palette", "item", "hovered"),
        CSS.BEM("palette", "item", "command")
      )}
      sharp
      {...props}
    >
      {name}
    </Button.Button>
  );
};

export const createResourceListItem = (
  resourceTypes: Record<string, ResourceType>
): FC<List.ItemProps<string, ontology.Resource>> => {
  const ResourceListItem = ({
    entry: { name, key, id },
    hovered,
    onSelect,
    ...props
  }: List.ItemProps<string, ontology.Resource>): ReactElement | null => {
    if (id == null) return null;
    const handleSelect = (): void => onSelect?.(key);
    const resourceType = resourceTypes[id.type];
    return (
      <Button.Button
        startIcon={resourceType?.icon}
        onClick={handleSelect}
        variant="text"
        className={CSS(
          CSS.BE("palette", "item"),
          hovered && CSS.BEM("palette", "item", "hovered"),
          CSS.BEM("palette", "item", "resource")
        )}
        {...props}
      >
        {name}
      </Button.Button>
    );
  };
  ResourceListItem.displayName = "ResourceListItem";
  return ResourceListItem;
};

export interface ResourceListItemProps
  extends List.ItemProps<string, ontology.Resource> {
  store: RootStore;
}

export interface CommandSelectionContext {
  store: RootStore;
  placeLayout: Layout.Placer;
}

export interface Command {
  key: string;
  name: string;
  icon?: ReactElement;
  onSelect: (ctx: CommandSelectionContext) => void;
}
