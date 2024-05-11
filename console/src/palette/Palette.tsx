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
  useMemo,
  useState,
  type MouseEventHandler,
  useLayoutEffect,
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
import { TooltipContent } from "@/palette/Tooltip";
import { type Mode, type TriggerConfig } from "@/palette/types";
import { type RootStore } from "@/store";

import "@/palette/Palette.css";
import { Align } from "@synnaxlabs/pluto";

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
  triggers: triggerConfig,
  commandSymbol,
}: PaletteProps): ReactElement => {
  const dropdown = Dropdown.use();

  const [value, setValue] = useState("");
  const mode = value.startsWith(commandSymbol) ? "command" : "resource";

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start" || dropdown.visible) return;
      const mode = Triggers.determineMode(triggerConfig, triggers);
      setValue(mode === "command" ? commandSymbol : "");
      dropdown.open();
    },
    [dropdown.visible, dropdown.open, triggerConfig.command, commandSymbol],
  );

  const triggers = useMemo(
    () => Triggers.flattenConfig(triggerConfig),
    [triggerConfig],
  );

  Triggers.use({ triggers, callback: handleTrigger });

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
    <List.List>
      <Tooltip.Dialog location="bottom" hide={dropdown.visible}>
        <TooltipContent triggers={triggerConfig} />
        <Dropdown.Dialog
          {...dropdown}
          keepMounted={false}
          visible={dropdown.visible}
          className={CSS.B("palette")}
          location="bottom"
          variant="modal"
        >
          <Button.Button
            onClick={dropdown.open}
            className={CSS(
              CSS.BE("palette", "btn"),
              PCSS.dropRegion(canDrop(dragging)),
            )}
            variant="outlined"
            align="center"
            size="medium"
            justify="center"
            startIcon={<Icon.Search />}
            shade={7}
            onDragOver={onDragOver}
            onDrop={onDrop}
          >
            Quick Search & Command
          </Button.Button>
          <PalletteDialogContent
            mode={mode}
            value={value}
            onChange={setValue}
            commands={commands}
            services={services}
            commandSymbol={commandSymbol}
            resourceTypes={services}
            close={dropdown.close}
          />
        </Dropdown.Dialog>
      </Tooltip.Dialog>
    </List.List>
  );
};
const canDrop: Haul.CanDrop = ({ items }) =>
  items.length === 1 && items[0].type === Mosaic.HAUL_DROP_TYPE;

const TYPE_TO_SEARCH = (
  <Align.Space align="center" className={CSS.BE("palette", "empty")} grow>
    <Status.Text level="h4" variant="disabled" hideIcon>
      Type to search
    </Status.Text>
  </Align.Space>
);

export interface PaletteListProps {
  mode: Mode;
  resourceTypes: Record<string, Service>;
}

const PaletteList = ({ mode, resourceTypes }: PaletteListProps): ReactElement => {
  const item = useMemo(() => {
    const Item = (
      mode === "command" ? CommandListItem : createResourceListItem(resourceTypes)
    ) as FC<List.ItemProps<string, ontology.Resource | Command>>;
    return componentRenderProp(Item);
  }, [mode, resourceTypes]);
  return (
    <List.Core.Virtual
      className={CSS.BE("palette", "list")}
      itemHeight={27}
      style={{ flexGrow: 1 }}
    >
      {item}
    </List.Core.Virtual>
  );
};

export interface PaletteDialogProps extends Input.Control<string> {
  mode: Mode;
  services: Ontology.Services;
  commandSymbol: string;
  resourceTypes: Record<string, Service>;
  commands: Command[];
  close: () => void;
}

const PalletteDialogContent = ({
  value,
  onChange,
  mode,
  commands,
  services,
  commandSymbol,
  close,
}: PaletteDialogProps): ReactElement => {
  const { setSourceData, setTransform, deleteTransform, setEmptyContent } =
    List.useDataUtilContext<Key, Entry>();

  const client = Synnax.use();
  const store = useStore() as RootStore;
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();

  useLayoutEffect(() => {
    if (mode === "command") {
      setSourceData(commands);
    } else {
      setSourceData([]);
      setEmptyContent(TYPE_TO_SEARCH);
    }
  }, [mode]);

  const handleSelect = useCallback(
    (key: Key, { entries }: List.UseSelectOnChangeExtra<Key, Entry>) => {
      close();
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
    [mode, commands, close, client, services],
  );

  const handleSearch = useDebouncedCallback(
    (term: string) => {
      if (term.length === 0 || client?.ontology == null)
        return setEmptyContent(TYPE_TO_SEARCH);
      client.ontology
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
    [client?.ontology, setSourceData],
  );

  const debouncedSetTransform = useDebouncedCallback(setTransform, 250, []);

  const handleFilter = useCallback(
    (term: string) => {
      if (term.length === 0) deleteTransform("filter");
      else debouncedSetTransform("filter", createFilterTransform({ term }));
    },
    [debouncedSetTransform, deleteTransform],
  );
  const handleChange = useCallback(
    (value: string) => {
      const mode = value.startsWith(commandSymbol) ? "command" : "resource";
      if (mode === "command") handleFilter(value.slice(1));
      else handleSearch(value);
      onChange(value);
    },
    [commandSymbol, handleFilter, handleSearch, onChange],
  );

  return (
    <List.Selector value={null} onChange={handleSelect} allowMultiple={false}>
      <List.Hover initialHover={0}>
        <Align.Pack className={CSS.BE("palette", "content")} direction="y">
          <Input.Text
            className={CSS(CSS.BE("palette", "input"))}
            placeholder={
              <Text.WithIcon level="h3" startIcon={<Icon.Search key="hello" />}>
                Search Synnax
              </Text.WithIcon>
            }
            size="huge"
            autoFocus
            onChange={handleChange}
            value={value}
            autoComplete="off"
          />
          <PaletteList mode={mode} resourceTypes={services} />
        </Align.Pack>
      </List.Hover>
    </List.Selector>
  );
};

export const CommandListItem = ({
  entry: { icon, name, key },
  hovered,
  onSelect,
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
    translate,
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
        style={{
          position: "absolute",
          transform: `translateY(${translate}px)`,
        }}
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
