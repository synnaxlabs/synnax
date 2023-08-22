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
} from "@synnaxlabs/pluto";
import { AsyncTermSearcher } from "@synnaxlabs/x";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import {
  LayoutPlacer,
  createLayoutMosaicWindow,
  moveLayoutMosaicTab,
  useLayoutPlacer,
} from "@/layout";
import { ResourceType } from "@/resources/resources";
import { RootStore } from "@/store";

import "@/palette/Palette.css";

export interface PaletteTriggerConfig {
  command: Triggers.Trigger[];
  resource: Triggers.Trigger[];
}

export interface PaletteProps {
  searcher: AsyncTermSearcher<string, string, OntologyResource>;
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
    <List.List>
      <Tooltip.Dialog location="bottom" hide={dropdown.visible}>
        <PaletteTooltipContent />
        <Dropdown.Dialog
          ref={dropdown.ref}
          visible={dropdown.visible}
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
          <PaletteList
            mode={mode}
            resourceTypes={resourceTypes}
            onSelect={handleSelect}
            visible={dropdown.visible}
          />
        </Dropdown.Dialog>
      </Tooltip.Dialog>
    </List.List>
  );
};

export interface PaletteInputProps
  extends Pick<Dropdown.UseReturn, "visible" | "open"> {
  mode: PaletteMode;
  visible: boolean;
  setMode: (mode: PaletteMode) => void;
  searcher: AsyncTermSearcher<string, string, OntologyResource>;
  commandSymbol: string;
  triggerConfig: PaletteTriggerConfig;
  commands: Command[];
}

const canDrop = (entities: Haul.Item[]): boolean =>
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

  const placer = useLayoutPlacer();
  const d = useDispatch();

  const { onDragOver, onDrop } = Haul.useDrop({
    canDrop,
    onDrop: useCallback(
      ([item]) => {
        const { key } = placer(createLayoutMosaicWindow());
        d(
          moveLayoutMosaicTab({
            windowKey: key,
            key: 1,
            tabKey: item.key as string,
            loc: "center",
          })
        );
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
    ) as FC<List.ItemProps<string, OntologyResource | Command>>;
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
): FC<List.ItemProps<string, OntologyResource>> => {
  const ResourceListItem = ({
    entry: { name, key, id },
    hovered,
    onSelect,
    ...props
  }: List.ItemProps<string, OntologyResource>): ReactElement | null => {
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
  extends List.ItemProps<string, OntologyResource> {
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
