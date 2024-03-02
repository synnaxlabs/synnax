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
import { Icon } from "@synnaxlabs/media";
import {
  Input,
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
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { List } from "@synnaxlabs/pluto/list";
import { TimeSpan, type AsyncTermSearcher } from "@synnaxlabs/x";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";
import { type Service } from "@/ontology/service";
import { Notifications } from "@/palette/Notifications";
import { TooltipContent } from "@/palette/Tooltip";
import { type Mode, type TriggerConfig } from "@/palette/types";
import { type RootStore } from "@/store";

import "@/palette/Palette.css";

type OntologySearcher = AsyncTermSearcher<string, string, ontology.Resource>;

export interface PaletteProps {
  commands: Command[];
  services: Ontology.Services;
  triggers: TriggerConfig;
  commandSymbol: string;
}

type Entry = Command | ontology.Resource;
type Key = string;

export const Palette = ({
  commands,
  services,
  triggers,
  commandSymbol,
}: PaletteProps): ReactElement => {
  const dropdown = Dropdown.use();
  const client = Synnax.use();
  const store = useStore() as RootStore;
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();

  const [mode, setMode] = useState<Mode>("resource");

  const notifications = Status.useNotifications({ expiration: TimeSpan.seconds(5) });

  const handleSelect = useCallback(
    (key: Key, { entries }: List.UseSelectOnChangeExtra<Key, Entry>) => {
      dropdown.close();
      if (mode === "command") {
        const entry = entries[0];
        (entry as Command).onSelect({
          store,
          placeLayout,
        });
      } else {
        if (client == null) return;
        const id = new ontology.ID(key);
        const t = services[id.type];
        t?.onSelect({
          services,
          store,
          placeLayout,
          removeLayout,
          client,
          selection: entries as ontology.Resource[],
        });
      }
    },
    [mode, commands, dropdown.close, client, services],
  );

  const showDropdown = dropdown.visible || notifications.statuses.length > 0;
  const showDivider = notifications.statuses.length > 0 && dropdown.visible;

  return (
    <List.List>
      <List.Selector value={null} onChange={handleSelect} allowMultiple={false}>
        <List.Hover disabled={!dropdown.visible} initialHover={0}>
          <Tooltip.Dialog location="bottom" hide={showDropdown}>
            <TooltipContent triggers={triggers} />
            <Dropdown.Dialog
              {...dropdown}
              visible={showDropdown}
              className={CSS.B("palette")}
              location="bottom"
              matchTriggerWidth
            >
              <PaletteInput
                mode={mode}
                setMode={setMode}
                searcher={client?.ontology}
                commandSymbol={commandSymbol}
                triggerConfig={triggers}
                commands={commands}
                visible={dropdown.visible}
                open={dropdown.open}
              />
              <>
                {dropdown.visible && (
                  <PaletteList mode={mode} resourceTypes={services} />
                )}
                {showDivider && <Divider.Divider direction="x" />}
                <Notifications {...notifications} />
              </>
            </Dropdown.Dialog>
          </Tooltip.Dialog>
        </List.Hover>
      </List.Selector>
    </List.List>
  );
};
export interface PaletteInputProps
  extends Pick<Dropdown.UseReturn, "visible" | "open"> {
  mode: Mode;
  visible: boolean;
  setMode: (mode: Mode) => void;
  searcher?: OntologySearcher;
  commandSymbol: string;
  triggerConfig: TriggerConfig;
  commands: Command[];
}

const canDrop: Haul.CanDrop = ({ items }) =>
  items.length === 1 && items[0].type === Mosaic.HAUL_DROP_TYPE;

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
    List.useDataUtilContext<Key, Entry>();

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
    [setMode, setSourceData, commands, setEmptyContent],
  );

  useEffect(() => {
    if (visible) return;
    inputRef.current?.blur();
    updateMode("resource");
  }, [visible, setMode]);

  const handleSearch = useDebouncedCallback(
    (term: string) => {
      if (term.length === 0 || searcher == null) return setEmptyContent(TYPE_TO_SEARCH);
      searcher
        .search(term)
        .then((d) => {
          if (d.length === 0)
            setEmptyContent(
              <Status.Text.Centered level="h4" variant="disabled" hideIcon>
                No resources found
              </Status.Text.Centered>,
            );
          setSourceData(d);
        })
        .catch((e) => {
          setEmptyContent(
            <Status.Text.Centered level="h4" variant="error">
              {e.message}
            </Status.Text.Centered>,
          );
        });
    },
    250,
    [searcher, setSourceData],
  );

  const debouncedSetTransform = useDebouncedCallback(setTransform, 250, []);

  const handleFilter = useCallback(
    (term: string) => {
      if (term.length === 0) deleteTransform("filter");
      else debouncedSetTransform("filter", createFilterTransform({ term }));
    },
    [debouncedSetTransform, deleteTransform],
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
    [visible, triggerConfig.command, commandSymbol, updateMode],
  );

  const triggers = useMemo(
    () => Triggers.flattenConfig(triggerConfig),
    [triggerConfig],
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
    [mode, setMode, commandSymbol, handleFilter, handleSearch],
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
          }),
        );
        return [item];
      },
      [placer],
    ),
  });

  const dragging = Haul.useDraggingState();

  return (
    <Input.Text
      className={CSS(CSS.BE("palette", "input"), PCSS.dropRegion(canDrop(dragging)))}
      ref={inputRef}
      placeholder={
        <Text.WithIcon level="p" startIcon={<Icon.Search key="hello" />}>
          Search Synnax
        </Text.WithIcon>
      }
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

export interface PalleteListProps {
  mode: Mode;
  resourceTypes: Record<string, Service>;
}

const PaletteList = ({ mode, resourceTypes }: PalleteListProps): ReactElement => {
  const item = useMemo(() => {
    const Item = (
      mode === "command" ? CommandListItem : createResourceListItem(resourceTypes)
    ) as FC<List.ItemProps<string, ontology.Resource | Command>>;
    return componentRenderProp(Item);
  }, [mode, resourceTypes]);
  return (
    <List.Core.Virtual className={CSS.BE("palette", "list")} itemHeight={27}>
      {item}
    </List.Core.Virtual>
  );
};

export const CommandListItem = ({
  entry: { icon, name, key },
  hovered,
  onSelect,
  style,
  translate,
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
        CSS.BEM("palette", "item", "command"),
      )}
      sharp
      style={{
        position: "absolute",
        transform: `translateY(${translate}px)`,
        ...style,
      }}
      {...props}
    >
      {name}
    </Button.Button>
  );
};

export const createResourceListItem = (
  resourceTypes: Record<string, Ontology.Service>,
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
          CSS.BEM("palette", "item", "resource"),
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
