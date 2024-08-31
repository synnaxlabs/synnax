// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/palette/Palette.css";

import { ontology } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Icon } from "@synnaxlabs/media";
import {
  Button,
  componentRenderProp,
  CSS as PCSS,
  Haul,
  Input,
  Mosaic,
  Status,
  Synnax,
  Text,
  Tooltip,
  Triggers,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { List } from "@synnaxlabs/pluto/list";
import { box, dimensions, runtime, xy } from "@synnaxlabs/x";
import { listen } from "@tauri-apps/api/event";
import { Window } from "@tauri-apps/api/window";
import {
  type FC,
  type ReactElement,
  useCallback,
  useId,
  useLayoutEffect,
  useMemo,
  useState,
} from "react";
import { useDispatch, useStore } from "react-redux";

import { Confirm } from "@/confirm";
import { CreateConfirmModal } from "@/confirm/Confirm";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";
import { type Service } from "@/ontology/service";
import { TooltipContent } from "@/palette/Tooltip";
import { type Mode, type TriggerConfig } from "@/palette/types";
import { Permissions } from "@/permissions";
import { SchematicServices } from "@/schematic/services";
import { RootState, type RootStore } from "@/store";
import { UserServices } from "@/user/services";

export interface PaletteProps {
  commands: Command[];
  services: Ontology.Services;
  triggers: TriggerConfig;
  commandSymbol: string;
}

type Entry = Command | ontology.Resource;
type Key = string;

const useDropOutsideMacOS = ({
  onDrop,
  canDrop,
  key,
  type,
}: Haul.UseDropOutsideProps) => {
  const ctx = Haul.useContext();
  if (ctx == null) return;
  const { drop } = ctx;
  const dragging = Haul.useDraggingRef();
  const key_ = key ?? useId();
  const target: Haul.Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const store = useStore<RootState>();
  useAsyncEffect(
    async () =>
      listen("mouse_up", async ({ payload: [x, y] }: { payload: [number, number] }) => {
        if (dragging.current.items.length === 0 || !canDrop(dragging.current)) return;
        const state = store.getState();
        const layout = Layout.select(state, dragging.current.items[0].key as string);
        if (layout?.windowKey == null) return;
        const winLabel = Drift.selectWindowLabel(state, layout.windowKey);
        if (winLabel == null || winLabel !== Drift.MAIN_WINDOW) return;
        const win = Window.getByLabel(winLabel);
        if (win == null) return;
        const sf = await win.scaleFactor();
        const rawCursor = xy.construct(x, y);
        const cursor = xy.scale(rawCursor, sf);
        const pos = xy.construct(await win.outerPosition());
        const size = dimensions.construct(await win.innerSize());
        const b = box.construct(pos, size);
        if (box.contains(b, cursor)) return;
        const dropped = onDrop(dragging.current, rawCursor);
        drop({ target, dropped });
      }),
    [target],
  );
};

const useDropOutside =
  runtime.getOS() === "MacOS" ? useDropOutsideMacOS : Haul.useDropOutside;

export const Palette = ({
  commands,
  services,
  triggers: triggerConfig,
  commandSymbol,
}: PaletteProps): ReactElement => {
  const permissions = Permissions.useSelectAll();
  const newCommands = commands.filter((command) => {
    if (command.key === SchematicServices.createCommand.key)
      return permissions.schematic;
    if (command.key === UserServices.registerUserCommand.key) return permissions.admin;
    return true;
  });
  const dropdown = Dropdown.use();

  const [value, setValue] = useState("");

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
  const store = useStore<RootState>();

  const handleDrop = useCallback(
    ({ items: [item] }: Haul.OnDropProps, cursor?: xy.XY) => {
      const windows = Drift.selectWindows(store.getState());
      const boxes = windows
        .filter((w) => w.stage === "created")
        .map((w) => box.construct(w.position, w.size));
      if (boxes.some((b) => box.contains(b, cursor))) return [];
      const { key } = placer(
        Layout.createMosaicWindow({
          position: cursor ? xy.translate(cursor, { x: -80, y: -45 }) : undefined,
        }),
      );
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
  );

  const dropProps = {
    type: "Palette",
    canDrop,
    onDrop: handleDrop,
  };

  useDropOutside(dropProps);
  const { onDragOver, onDrop } = Haul.useDrop(dropProps);

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
            value={value}
            onChange={setValue}
            commands={newCommands}
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

export interface PaletteListProps {
  mode: Mode;
  resourceTypes: Ontology.Services;
  commandSelectionContext: CommandSelectionContext;
}

const PaletteList = ({
  mode,
  resourceTypes,
  commandSelectionContext,
}: PaletteListProps): ReactElement => {
  const item = useMemo(() => {
    const Item = (
      mode === "command"
        ? createCommandListItem(commandSelectionContext)
        : createResourceListItem(resourceTypes)
    ) as FC<List.ItemProps<string, ontology.Resource | Command>>;
    return componentRenderProp(Item);
  }, [commandSelectionContext, mode, resourceTypes]);
  return (
    <List.Core
      className={CSS.BE("palette", "list")}
      itemHeight={27}
      style={{ flexGrow: 1 }}
    >
      {item}
    </List.Core>
  );
};

export interface PaletteDialogProps extends Input.Control<string> {
  services: Ontology.Services;
  commandSymbol: string;
  resourceTypes: Record<string, Service>;
  commands: Command[];
  close: () => void;
}

const PalletteDialogContent = ({
  value,
  onChange,
  commands,
  services,
  commandSymbol,
  close,
}: PaletteDialogProps): ReactElement => {
  const { setSourceData } = List.useDataUtilContext<Key, Entry>();
  const addStatus = Status.useAggregator();
  const client = Synnax.use();
  const store = useStore() as RootStore;
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();

  const mode = value.startsWith(commandSymbol) ? "command" : "resource";

  useLayoutEffect(() => setSourceData(mode === "command" ? commands : []), [mode]);

  const confirm = Confirm.useModal();

  const cmdSelectCtx = useMemo<CommandSelectionContext>(
    () => ({ store, placeLayout, confirm }),
    [store, placeLayout],
  );

  const handleSelect = useCallback(
    (key: Key, { entries }: List.UseSelectOnChangeExtra<Key, Entry>) => {
      close();
      if (mode === "command") {
        const entry = entries[0];
        (entry as Command).onSelect(cmdSelectCtx);
      } else {
        if (client == null) return;
        const id = new ontology.ID(key);
        const t = services[id.type];
        t?.onSelect({
          services,
          store,
          addStatus,
          placeLayout,
          removeLayout,
          client,
          selection: entries as ontology.Resource[],
        });
      }
    },
    [mode, commands, close, client, services, addStatus],
  );

  const { value: searchValue, onChange: onSearchChange } = List.useSearch<
    string,
    ontology.Resource
  >({
    value,
    onChange,
    searcher: client?.ontology,
  });

  const { value: filterValue, onChange: onFilterChange } = List.useFilter({
    value,
    onChange,
    transformBefore: (term) => term.slice(1),
  });

  const handleChange = useCallback(
    (value: string) => {
      const mode = value.startsWith(commandSymbol) ? "command" : "resource";
      if (mode === "command") onFilterChange(value);
      else onSearchChange(value);
    },
    [commandSymbol, searchValue, onFilterChange, onChange],
  );

  const actualValue = mode === "command" ? filterValue : searchValue;

  return (
    <List.Selector value={null} onChange={handleSelect} allowMultiple={false}>
      <List.Hover initialHover={0}>
        <Align.Pack className={CSS.BE("palette", "content")} direction="y">
          <Input.Text
            className={CSS(CSS.BE("palette", "input"))}
            placeholder={
              <Text.WithIcon level="h3" startIcon={<Icon.Search key="hello" />}>
                Type to search or {">"} to view commands
              </Text.WithIcon>
            }
            size="huge"
            autoFocus
            onChange={handleChange}
            value={actualValue}
            autoComplete="off"
          />
          <PaletteList
            mode={mode}
            resourceTypes={services}
            commandSelectionContext={cmdSelectCtx}
          />
        </Align.Pack>
      </List.Hover>
    </List.Selector>
  );
};

const CommandAction = ({
  name,
  trigger: keyboardShortcut,
}: CommandActionProps & { ctx: CommandSelectionContext }): ReactElement => (
  <Align.Pack direction="x" className={CSS.BE("palette", "action")}>
    <Text.Keyboard
      level="small"
      style={{
        display: "flex",
        alignItems: "center",
        padding: "0 1.5rem",
      }}
      shade={7}
    >
      {Triggers.toSymbols(keyboardShortcut)}
    </Text.Keyboard>
    <Button.Button variant="outlined" size="small" shade={7}>
      {name}
    </Button.Button>
  </Align.Pack>
);

const createCommandListItem = (
  ctx: CommandSelectionContext,
): FC<List.ItemProps<string, Command>> => {
  const CommandListItem = (props: List.ItemProps<string, Command>): ReactElement => {
    const {
      entry: { icon, name, actions },
    } = props;
    return (
      <List.ItemFrame
        highlightHovered
        style={{ height: "6.5rem" }}
        justify="spaceBetween"
        align="center"
        {...props}
      >
        <Text.WithIcon startIcon={icon} level="p" weight={400} shade={9} size="medium">
          {name}
        </Text.WithIcon>
        <Align.Space direction="x" className={CSS.BE("palette", "actions")}>
          {actions != null &&
            actions.map((action, i) => <CommandAction key={i} {...action} ctx={ctx} />)}
        </Align.Space>
      </List.ItemFrame>
    );
  };
  return CommandListItem;
};

type OntologyListItemProps = List.ItemProps<string, ontology.Resource>;

export const createResourceListItem = (
  resourceTypes: Ontology.Services,
): FC<OntologyListItemProps> => {
  const ResourceListItem = (props: OntologyListItemProps): ReactElement | null => {
    const {
      entry: { name, id },
    } = props;
    if (id == null) return null;
    const resourceType = resourceTypes[id.type];
    const PI = resourceType?.PaletteListItem;
    if (PI != null) return <PI {...props} />;
    return (
      <List.ItemFrame style={{ padding: "1.5rem" }} highlightHovered {...props}>
        <Text.WithIcon
          startIcon={resourceType?.icon}
          level="p"
          weight={450}
          shade={9}
          size="medium"
        >
          {name}
        </Text.WithIcon>
      </List.ItemFrame>
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
  confirm: CreateConfirmModal;
}

interface CommandActionProps {
  name: string;
  trigger: Triggers.Trigger;
  onClick: (ctx: CommandSelectionContext) => void;
}

export interface Command {
  key: string;
  name: ReactElement | string;
  icon?: ReactElement;
  onSelect: (ctx: CommandSelectionContext) => void;
  actions?: CommandActionProps[];
}
