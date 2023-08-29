// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  FC,
  ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
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
  Text,
  Align,
  Mosaic,
  Divider,
} from "@synnaxlabs/pluto";
import { AsyncTermSearcher, Key } from "@synnaxlabs/x";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { ResourceType } from "@/resources/resources";
import { RootStore } from "@/store";

import "@/palette/Palette.css";

export interface PaletteTriggerConfig {
  command: Triggers.Trigger[];
  resource: Triggers.Trigger[];
}

export interface PaletteProps {
  searcher: AsyncTermSearcher<string, string, ontology.Resource>;
  commands: Command[];
  resourceTypes: Record<string, ResourceType>;
  triggers: PaletteTriggerConfig;
  commandSymbol: string;
}

const normalizeTriggers = (triggers: PaletteTriggerConfig): Triggers.Trigger[] =>
  Object.values(triggers).flat();

type PaletteMode = "command" | "resource";

const TOOLTIP_TEXT_LEVEL: Text.Level = "small";

const PaletteTooltipContent = (): ReactElement => (
  <Align.Space>
    <Align.Space direction="x" justify="spaceBetween" align="center">
      <Text.Text level={TOOLTIP_TEXT_LEVEL}>Search</Text.Text>
      <Align.Space direction="x" size="small">
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>
          <Text.Symbols.Meta />
        </Text.Keyboard>
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>P</Text.Keyboard>
      </Align.Space>
    </Align.Space>
    <Align.Space direction="x" justify="spaceBetween" align="center">
      <Text.Text level={TOOLTIP_TEXT_LEVEL}>Command Palette</Text.Text>
      <Align.Space direction="x" size="small">
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>
          <Text.Symbols.Meta />
        </Text.Keyboard>
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>
          <Icon.Keyboard.Shift />
        </Text.Keyboard>
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>P</Text.Keyboard>
      </Align.Space>
    </Align.Space>
  </Align.Space>
);

export const Palette = ({
  commands,
  searcher,
  resourceTypes,
  triggers,
  commandSymbol,
}: PaletteProps): ReactElement => {
  const dropdown = Dropdown.use();

  const [mode, setMode] = useState<PaletteMode>("resource");

  const notifications = Status.useNotifications({});

  const store = useStore() as RootStore;
  const placeLayout = Layout.usePlacer();
  const handleSelect: List.SelectorProps["onChange"] = useCallback(
    ([key]: Key[], { entries: [entry] }) => {
      dropdown.close();
      if (mode === "command") {
        (entry as Command).onSelect({
          store,
          placeLayout,
        });
      } else {
        const id = new ontology.ID(key as string);
        const t = resourceTypes[id.type];
        t?.onSelect({
          store,
          placeLayout,
          selected: entry as ontology.Resource,
        });
      }
    },
    [mode, commands, dropdown.close]
  );

  const showDropdown = dropdown.visible || notifications.length > 0;

  return (
    <List.List>
      <Tooltip.Dialog location="bottom" hide={showDropdown}>
        <PaletteTooltipContent />
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
            {notifications.length > 0 && dropdown.visible && (
              <Divider.Divider direction="x" />
            )}
            <ErrorsList statuses={notifications} />
          </>
        </Dropdown.Dialog>
      </Tooltip.Dialog>
    </List.List>
  );
};

interface ErrorsListProps {
  statuses: Status.Spec[];
}

const ErrorsList = ({ statuses }: ErrorsListProps): ReactElement => (
  <>
    {statuses.map(({ message, variant, time }) => (
      <Align.Space
        direction="x"
        key={time.toString()}
        style={{ height: "6rem", padding: "1rem" }}
        align="center"
      >
        <Text.DateTime level="p" format="time">
          {time}
        </Text.DateTime>
        <Status.Text variant={variant}>{message}</Status.Text>
      </Align.Space>
    ))}
  </>
);

export interface PaletteInputProps
  extends Pick<Dropdown.UseReturn, "visible" | "open"> {
  mode: PaletteMode;
  visible: boolean;
  setMode: (mode: PaletteMode) => void;
  searcher: AsyncTermSearcher<string, string, ontology.Resource>;
  commandSymbol: string;
  triggerConfig: PaletteTriggerConfig;
  commands: Command[];
}

const canDrop: Haul.CanDrop = ({ items }) =>
  items.length === 1 && items[0].type === Mosaic.HAUL_TYPE;

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
    List.useContext();

  useEffect(() => {
    if (!visible) inputRef.current?.blur();
  }, [visible, setMode]);

  const handleBlur = useCallback(() => setValue(""), []);

  const updateMode = useCallback(
    (nextMode: PaletteMode) => {
      setMode(nextMode);
      if (nextMode === "command") setSourceData(commands);
      else {
        setSourceData([]);
        setEmptyContent(
          <Status.Text.Centered
            level="h4"
            variant="disabled"
            hideIcon
            style={{ height: 150 }}
          >
            Type to search
          </Status.Text.Centered>
        );
      }
    },
    [setMode, setSourceData, commands]
  );

  const handleSearch = useDebouncedCallback(
    (term: string) => {
      if (term.length === 0)
        return setEmptyContent(
          <Status.Text.Centered
            level="h4"
            variant="disabled"
            hideIcon
            style={{ height: 150 }}
          >
            Type to search
          </Status.Text.Centered>
        );
      searcher
        .search(term)
        .then((d) => {
          if (d.length === 0)
            setEmptyContent(
              <Status.Text.Centered
                level="h4"
                variant="disabled"
                hideIcon
                style={{ height: 150 }}
              >
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
      const mode = Triggers.match(triggers, triggerConfig.command)
        ? "command"
        : "resource";
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

  const triggers = useMemo(() => normalizeTriggers(triggerConfig), [triggerConfig]);

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
  mode: PaletteMode;
  onSelect: List.SelectorProps["onChange"];
  resourceTypes: Record<string, ResourceType>;
}

export const PaletteList = ({
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
      <List.Core.Virtual itemHeight={27}>{item}</List.Core.Virtual>
    </>
  );
};

export const CommandListItem = ({
  entry: { icon, name, key },
  hovered,
  onSelect,
  ...props
}: List.ItemProps<string, Command>): ReactElement => {
  const handleSelect = (e): void => {
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
