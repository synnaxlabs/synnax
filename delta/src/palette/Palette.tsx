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

import { OntologyID, OntologyResource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Dropdown,
  Input,
  List,
  ListItemProps,
  Trigger,
  Triggers,
  componentRenderProp,
  UseTriggerEvent,
  Button,
  UseDropdownReturn,
  useDebouncedCallback,
  createFilterTransform,
  Status,
  DropdownProps,
  Haul,
  Hauled,
  CSS as PCSS,
  Tooltip,
  Text,
  Typography,
  Space,
  TypographyLevel,
} from "@synnaxlabs/pluto";
import { AsyncTermSearcher } from "@synnaxlabs/x";
import { useStore } from "react-redux";

import { CSS } from "@/css";
import { LayoutPlacer, useLayoutPlacer } from "@/layout";
import { createLinePlot } from "@/line/store/slice";
import { ResourceType } from "@/resources/resources";

import "@/palette/Palette.css";

export interface PaletteTriggerConfig {
  command: Trigger[];
  resource: Trigger[];
}

export interface PaletteProps {
  searcher: AsyncTermSearcher<string, string, OntologyResource>;
  commands: Command[];
  resourceTypes: Record<string, ResourceType>;
  triggers: PaletteTriggerConfig;
  commandSymbol: string;
}

const normalizeTriggers = (triggers: PaletteTriggerConfig): Trigger[] =>
  Object.values(triggers).flat();

type PaletteMode = "command" | "resource";

const TOOLTIP_TEXT_LEVEL: TypographyLevel = "small";

const PaletteTooltipContent = (): ReactElement => (
  <Space>
    <Space direction="x" justify="spaceBetween" align="center">
      <Text level={TOOLTIP_TEXT_LEVEL}>Search</Text>
      <Space direction="x" size="small">
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>
          <Typography.Symbols.Meta />
        </Text.Keyboard>
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>P</Text.Keyboard>
      </Space>
    </Space>
    <Space direction="x" justify="spaceBetween" align="center">
      <Text level={TOOLTIP_TEXT_LEVEL}>Command Palette</Text>
      <Space direction="x" size="small">
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>
          <Typography.Symbols.Meta />
        </Text.Keyboard>
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>
          <Icon.Keyboard.Shift />
        </Text.Keyboard>
        <Text.Keyboard level={TOOLTIP_TEXT_LEVEL}>P</Text.Keyboard>
      </Space>
    </Space>
  </Space>
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

  const store = useStore() as RootStore;
  const placeLayout = useLayoutPlacer();
  const handleSelect = useCallback(
    ([key]: readonly string[], [entry]: Array<OntologyResource | Command>) => {
      dropdown.close();
      if (mode === "command") {
        (entry as Command).onSelect({
          store,
          placeLayout,
        });
      } else {
        const id = new OntologyID(key);
        const t = resourceTypes[id.type];
        t?.onSelect({
          store,
          placeLayout,
          resource: entry as OntologyResource,
        });
      }
    },
    [mode, commands, dropdown.close]
  );

  return (
    <List>
      <Tooltip location="bottom" hide={dropdown.visible}>
        <PaletteTooltipContent />
        <Dropdown
          ref={dropdown.ref}
          visible={dropdown.visible}
          className={CSS.B("palette")}
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
          <PaletteList
            mode={mode}
            resourceTypes={resourceTypes}
            onSelect={handleSelect}
            visible={dropdown.visible}
          />
        </Dropdown>
      </Tooltip>
    </List>
  );
};

export interface PaletteInputProps extends Pick<UseDropdownReturn, "visible" | "open"> {
  mode: PaletteMode;
  visible: boolean;
  setMode: (mode: PaletteMode) => void;
  searcher: AsyncTermSearcher<string, string, OntologyResource>;
  commandSymbol: string;
  triggerConfig: PaletteTriggerConfig;
  commands: Command[];
}

const canDrop = (entities: Hauled[]): boolean =>
  entities.length === 1 && entities[0].type === "pluto-mosaic-tab";

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
    ({ triggers, stage }: UseTriggerEvent) => {
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

  const placer = useLayoutPlacer();

  const { onDragLeave, onDragOver, onDrop } = Haul.useDropRegion({
    canDrop,
    onDrop: useCallback(
      ([item]) => {
        placer(
          createLinePlot({
            key: item.key as string,
            location: "window",
            window: {
              navTop: true,
              size: { height: 600, width: 900 },
            },
          })
        );
      },
      [placer]
    ),
  });

  const { dragging } = Haul.useState();

  return (
    <Input
      className={CSS(CSS.BE("palette", "input"), PCSS.dropRegion(canDrop(dragging)))}
      ref={inputRef}
      placeholder="Search Synnax"
      onDragLeave={onDragLeave}
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

export interface PalleteListProps extends Pick<DropdownProps, "visible"> {
  mode: PaletteMode;
  onSelect: (
    keys: readonly string[],
    entries: Array<OntologyResource | Command>
  ) => void;
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
    ) as FC<ListItemProps<string, OntologyResource | Command>>;
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
}: ListItemProps<string, Command>): ReactElement => {
  const handleSelect = (e): void => {
    e.stopPropagation();
    onSelect?.(key);
  };
  return (
    <Button
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
    </Button>
  );
};

export const createResourceListItem = (
  resourceTypes: Record<string, ResourceType>
): FC<ListItemProps<string, OntologyResource>> => {
  const ResourceListItem = ({
    entry: { name, key, id },
    hovered,
    onSelect,
    ...props
  }: ListItemProps<string, OntologyResource>): ReactElement | null => {
    if (id == null) return null;
    const handleSelect = (): void => onSelect?.(key);
    const resourceType = resourceTypes[id.type];
    return (
      <Button
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
      </Button>
    );
  };
  ResourceListItem.displayName = "ResourceListItem";
  return ResourceListItem;
};

export interface ResourceListItemProps extends ListItemProps<string, OntologyResource> {
  store: RootStore;
}

export interface CommandSelectionContext {
  store: RootStore;
  placeLayout: LayoutPlacer;
}

export interface Command {
  key: string;
  name: string;
  icon?: ReactElement;
  onSelect: (ctx: CommandSelectionContext) => void;
}
