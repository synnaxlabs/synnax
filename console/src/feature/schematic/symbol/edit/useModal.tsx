// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { type group, type schematic, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { useCombinedStateAndRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import { type bounds, id, type xy } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useRef, useState } from "react";
import type { z } from "zod";

import {
  ClickInspector,
  targetName,
} from "@/feature/schematic/symbol/edit/ClickInspector";
import { CommandInspector } from "@/feature/schematic/symbol/edit/CommandInspector";
import { HandleList } from "@/feature/schematic/symbol/edit/HandleList";
import { Inspector } from "@/feature/schematic/symbol/edit/Inspector";
import { ItemList } from "@/feature/schematic/symbol/edit/ItemList";
import { PartList } from "@/feature/schematic/symbol/edit/PartList";
import { Preview } from "@/feature/schematic/symbol/edit/Preview";
import { useConfirmSave } from "@/feature/schematic/symbol/edit/useConfirmSave";
import { useValues } from "@/feature/schematic/symbol/edit/useValues";
import { SelectVariantField } from "@/feature/schematic/symbol/SelectVariant";
import { Spec } from "@/feature/schematic/symbol/spec";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const path = Schematic.Symbol.formPath;

export interface ModalParams {
  /** When provided, the modal edits the existing symbol with this key. */
  symbolKey?: Schematic.Symbol.FormQuery["key"];
  /** The group a new symbol joins. */
  group?: group.Key;
  scale?: number;
  /// createKey, when set on a create-mode open, forces the new symbol to be
  /// persisted under this key. Used by the missing-symbol re-link flow so
  /// that creating a replacement symbol heals every node that already
  /// references the dangling key, without a manual re-link.
  createKey?: string;
}

const SCALE_BOUNDS: bounds.Bounds = { lower: 5, upper: 1000 };

const STATE_TYPES: Record<schematic.symbol.StateType, string> = {
  enumeration: "Enumeration",
  number: "Number",
};

interface CommandDetailProps {
  command: schematic.symbol.Command;
}

// CommandDetail names the value that a command sends, by the enumeration of its state
// when it has one.
const CommandDetail = ({ command }: CommandDetailProps): ReactNode => {
  const values = useValues(command.state);
  if (command.value.type === "ask") return "Asks for a value";
  const { value } = command.value;
  const named =
    values.state === command.state
      ? values.list.find((v) => v.value === value)?.name
      : undefined;
  return `Sends ${named ?? value}`;
};

// Item is a command or a click that the inspector edits.
interface Item {
  type: "command" | "click";
  key: string;
}

interface ValueBarProps {
  value: Spec.At;
  onChange: (value: Spec.At) => void;
}

// ValueBar picks the state that the inspector edits, and the value that the canvas
// draws.
const ValueBar = ({ value, onChange }: ValueBarProps): ReactElement | null => {
  const { state, states, list } = useValues(value.state);
  if (list.length < 2 && states.length < 2) return null;
  const handleStateChange = (key: string) =>
    onChange({ state: key, value: states.find((s) => s.key === key)?.default ?? 0 });
  return (
    <Flex.Box x align="center" gap="small" className={CSS.BE("schematic", "value-bar")}>
      <Text.Text level="small" color={9}>
        Preview
      </Text.Text>
      {states.length > 1 && (
        <Select.Simple<string>
          resourceName="state"
          value={state}
          onChange={handleStateChange}
          allowNone={false}
          triggerProps={{ "aria-label": "Preview state" }}
        >
          {states.map((s) => (
            <Select.Item<string> key={s.key} itemKey={s.key}>
              {s.name}
            </Select.Item>
          ))}
        </Select.Simple>
      )}
      <Select.Simple<number>
        resourceName="value"
        value={value.value}
        onChange={(v: number) => onChange({ state, value: v })}
        allowNone={false}
        triggerProps={{ "aria-label": "Preview value" }}
      >
        {list.map((v) => (
          <Select.Item<number> key={v.value} itemKey={v.value}>
            {v.name}
          </Select.Item>
        ))}
      </Select.Simple>
    </Flex.Box>
  );
};

// climb returns what an Alt-click on shape picks for part, as Select parent group does:
// the group above the outermost shape of part there, or above the shape. With no group
// above, it returns shape when part holds none of it.
const climb = (
  spec: schematic.symbol.Spec,
  part: string | undefined,
  shape: Schematic.Node.Region.Named,
): Schematic.Node.Region.Named | null => {
  const drawn = { ...spec, drawing: { ...spec.drawing, markup: shape.markup } };
  const from = part == null ? undefined : Spec.outermost(drawn, part, shape.selector);
  const group = Spec.parentGroup(drawn, [from ?? shape.selector]);
  if (group == null) return from == null ? shape : null;
  return Schematic.Node.Region.name(
    shape.markup,
    group.ownerDocument.documentElement,
    group,
  );
};

const Editor = ({
  symbolKey,
  group,
  createKey,
  close,
}: Modals.ContentProps<ModalParams, void>): ReactElement => {
  const isCreate = symbolKey == null;
  const initialValues: z.infer<typeof Schematic.Symbol.formSchema> = {
    key: createKey,
    name: "",
    group,
    data: {
      drawing: { type: "svg", markup: "" },
      parts: [],
      previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
      handles: [],
      variant: "static",
      strokeScaled: false,
      states: [Spec.createState()],
      commands: [],
      looks: [],
      interactions: [],
      scale: 1,
    },
  };
  const { form, save, variant } = Schematic.Symbol.useForm({
    query: symbolKey == null ? null : { key: symbolKey },
    initialValues,
    afterSave: () => close(),
  });
  Modals.useBusy(variant === "loading");
  const stored = Schematic.Symbol.useResult(
    symbolKey == null ? null : { key: symbolKey, includeUses: true },
  );
  Modals.useUnsaved(() => {
    const base = isCreate ? initialValues : stored.data;
    // Until the stored symbol loads, there is nothing to compare with.
    if (base == null) return false;
    const { name, data } = form.value();
    return name !== base.name || Spec.edited(base.data, data);
  });
  const confirmSave = useConfirmSave();
  const [preview, setPreview] = useState<Spec.At>({ state: Spec.STATE, value: 0 });
  const [selectedPart, setSelectedPart, selectedPartRef] =
    useCombinedStateAndRef<Spec.Target>(undefined);
  const [selectedHandle, setSelectedHandle] = useState<string | undefined>(undefined);
  const [selectedItem, setSelectedItem] = useState<Item | undefined>(undefined);
  // A part and an item share the inspector, so picking one drops the other.
  const selectPart = (key: Spec.Target) => {
    setSelectedPart(key);
    setSelectedItem(undefined);
  };
  const selectItem = (type: Item["type"]) => (key: string | undefined) => {
    setSelectedItem(key == null ? undefined : { type, key });
    setSelectedPart(undefined);
  };

  // addPart adds a part, with the shape of selector when one is given, and selects it.
  const addPart = (selector?: string) => {
    const parts = form.get(path.data.parts).value;
    const part: schematic.symbol.Part = {
      key: `part-${id.create()}`,
      name: Spec.nextPartName(parts),
      selectors: [],
    };
    const next = [...parts, part];
    const drawing = form.get(path.data.drawing).value;
    form.set(
      path.data.parts,
      selector == null
        ? next
        : Spec.toggleSelector({ drawing, parts: next }, part.key, selector),
    );
    selectPart(part.key);
  };

  const addNewHandle = () => {
    const currentHandles = form.get(path.data.handles).value;
    const newHandle: schematic.symbol.Handle = {
      key: `handle-${id.create()}`,
      position: { x: 0.5, y: 0.5 },
      orientation: "left",
    };
    form.set(path.data.handles, [...currentHandles, newHandle]);
    setSelectedHandle(newHandle.key);
  };

  const handleHandlePlace = (handleKey: string, position: xy.XY) => {
    const currentHandles = form.get(path.data.handles).value;
    const handleIndex = currentHandles.findIndex((h) => h.key === handleKey);

    if (handleIndex !== -1) {
      const updatedHandles = [...currentHandles];
      updatedHandles[handleIndex] = { ...updatedHandles[handleIndex], position };
      form.set(path.data.handles, updatedHandles);
    }
  };

  const lastClickSelectsRef = useRef(false);
  const handleElementClick = (
    shape: Schematic.Node.Region.Named,
    { detail, altKey }: Pick<MouseEvent, "detail" | "altKey">,
  ) => {
    // A later click of the same double click would undo a click that selects a part.
    if (detail > 1 && lastClickSelectsRef.current) return;
    const part = selectedPartRef.current;
    const named = altKey ? climb(form.get(path.data).value, part, shape) : shape;
    if (named == null) return;
    if (named.markup !== form.get(path.data.drawing.markup).value)
      form.set(path.data.drawing.markup, named.markup);
    const { selector } = named;
    const spec = form.get(path.data).value;
    // A shape in a group of another part cannot join part, so the click selects it.
    const holder = Spec.groupOwner(spec, selector);
    const toggles = part != null && (holder == null || holder.key === part);
    lastClickSelectsRef.current = !toggles;
    if (toggles)
      return form.set(path.data.parts, Spec.toggleSelector(spec, part, selector));
    const owner = part == null ? Spec.owner(spec, selector) : holder;
    if (owner == null) return addPart(selector);
    selectPart(owner.key);
  };
  const states = Form.useFieldValue(path.data.states, { ctx: form });
  const commands = Form.useFieldValue(path.data.commands, { ctx: form });
  const interactions = Form.useFieldValue(path.data.interactions, { ctx: form });
  const parts = Form.useFieldValue(path.data.parts, { ctx: form });
  const shown =
    selectedItem != null &&
    (selectedItem.type === "command" ? commands : interactions).some(
      (i) => i.key === selectedItem.key,
    )
      ? selectedItem
      : undefined;
  const describeClick = (click: schematic.symbol.Interaction): string => {
    const name = (key: string) => commands.find((c) => c.key === key)?.name ?? key;
    return click.type === "fire"
      ? `Sends ${name(click.command)}`
      : `Toggles ${click.commands.map(name).join(" and ")}`;
  };
  const clickDisabledReason =
    commands.length === 0
      ? "Add a command first"
      : Spec.freeTargets({ parts, interactions }).length === 0
        ? "Every part and the whole drawing have a click"
        : undefined;
  const hasSVG =
    Form.useFieldValue(path.data.drawing.markup, {
      ctx: form,
    }).length > 0;
  const createSaveText = isCreate ? "Create" : "Save";
  const handleSave = async () => {
    const spec = form.get(path.data).value;
    // A fill motion that cannot draw stops the save and selects its part, whose
    // inspector shows why. The inspector shows the issues only with the flag on.
    if (FLAGS.symbolMotion) {
      const [part] = Spec.levelIssues(spec).keys();
      if (part != null) return selectPart(part);
    }
    // Placement reads the variant, so a symbol with a click saves as an actuator.
    if (FLAGS.symbolNodes)
      form.set(path.data.variant, spec.interactions.length > 0 ? "actuator" : "static");
    const uses = stored.data?.uses ?? [];
    if (stored.data != null && uses.length > 0) {
      const next = form.get(path.data).value;
      if (
        Spec.edited(stored.data.data, next) &&
        (await confirmSave({ stored: stored.data, spec: next, uses })) !== true
      )
        return;
    }
    save();
  };

  return (
    <Form.Form<typeof Schematic.Symbol.formSchema> {...form}>
      <Modal.Frame
        className={CSS.BE("schematic", "symbol-create-layout")}
        background={1}
      >
        <Modal.Header icon={<Icon.Schematic />}>
          {isCreate ? "Schematic.Symbol.Create" : "Schematic.Symbol.Edit"}
        </Modal.Header>
        <Modal.Body full>
          <Flex.Box x grow>
            {hasSVG && (
              <Flex.Box
                className={CSS.BE("schematic", "sidebar")}
                y
                rounded="small"
                background={0}
                bordered
                borderColor={6}
              >
                <Flex.Box className={CSS.BE("schematic", "name")}>
                  <Form.TextField
                    path={path.name}
                    inputProps={{
                      placeholder: "Symbol name",
                      variant: "text",
                      level: "h4",
                    }}
                  />
                </Flex.Box>
                <Divider.Divider x />
                {FLAGS.symbolNodes ? (
                  <>
                    <ItemList
                      title="States"
                      noun="state"
                      items={states}
                      label={(s) => s.name}
                      name={(key) => path.data.states[key].name}
                      detail={(s) => STATE_TYPES[s.type]}
                      add={Spec.addState}
                      remove={Spec.removeState}
                      // The inspector shows no state, so no state looks selected.
                      value={undefined}
                      onChange={(key) => {
                        // The preview shows enumeration states alone.
                        const state = states.find((s) => s.key === key);
                        if (state?.type !== "enumeration") return;
                        setPreview({
                          state: state.key,
                          value: state.default,
                        });
                      }}
                      onRemove={() => {
                        // The preview moves to the first state left if its state goes.
                        const left = form
                          .get(path.data.states)
                          .value.filter(
                            (s): s is schematic.symbol.EnumerationState =>
                              s.type === "enumeration",
                          );
                        if (left.some((s) => s.key === preview.state)) return;
                        const first = left.at(0);
                        setPreview({
                          state: first?.key ?? Spec.STATE,
                          value: first?.default ?? 0,
                        });
                      }}
                    />
                    <Divider.Divider x />
                    <ItemList
                      title="Commands"
                      noun="command"
                      items={commands}
                      label={(c) => c.name}
                      name={(key) => path.data.commands[key].name}
                      detail={(c) => <CommandDetail command={c} />}
                      add={Spec.addCommand}
                      remove={Spec.removeCommand}
                      value={shown?.type === "command" ? shown.key : undefined}
                      onChange={selectItem("command")}
                    />
                    <Divider.Divider x />
                    <ItemList
                      title="Clicks"
                      noun="click"
                      items={interactions}
                      label={(i) => targetName(parts, i.part)}
                      detail={describeClick}
                      add={Spec.addInteraction}
                      addDisabledReason={clickDisabledReason}
                      remove={Spec.removeInteraction}
                      value={shown?.type === "click" ? shown.key : undefined}
                      onChange={selectItem("click")}
                    />
                  </>
                ) : (
                  <Flex.Box y>
                    <Header.Header level="p" bordered={false}>
                      <Header.Title level="p" weight={500}>
                        States
                      </Header.Title>
                    </Header.Header>
                    <Flex.Box className={CSS.BE("schematic", "states")}>
                      <SelectVariantField
                        onSelectValue={(value) =>
                          setPreview({ state: Spec.STATE, value })
                        }
                      />
                    </Flex.Box>
                  </Flex.Box>
                )}
                <Divider.Divider x />
                <PartList
                  value={shown == null ? selectedPart : null}
                  onChange={(value) => {
                    selectPart(value);
                    setSelectedHandle(undefined);
                  }}
                  onAdd={() => addPart()}
                />
                <Divider.Divider x />
                <HandleList
                  value={selectedHandle}
                  onChange={setSelectedHandle}
                  onAddHandle={addNewHandle}
                />
                <Divider.Divider x />
                <Flex.Box y>
                  <Header.Header level="p" bordered={false}>
                    <Header.Title level="p" weight={500}>
                      Properties
                    </Header.Title>
                  </Header.Header>
                  <Flex.Box className={CSS.BE("schematic", "properties")}>
                    <Form.Field
                      path={path.data.scale}
                      helpText="Sets the default scale when added to a schematic"
                    >
                      {({ onChange, value }) => (
                        <Input.Numeric
                          value={Math.round(value * 100)}
                          onChange={(v) => onChange(v / 100)}
                          bounds={SCALE_BOUNDS}
                          dragScale={0.5}
                          endContent="%"
                        />
                      )}
                    </Form.Field>
                    <Form.CheckboxField
                      path={path.data.strokeScaled}
                      label="Scale stroke"
                      align="start"
                    />
                  </Flex.Box>
                </Flex.Box>
              </Flex.Box>
            )}
            <Flex.Box y grow className={CSS.BE("schematic", "canvas")}>
              <Form.Field
                path={path.data.drawing.markup}
                showLabel={false}
                showHelpText={false}
              >
                {({ onChange }) => (
                  <Preview
                    value={new Map([[preview.state, preview.value]])}
                    selectedHandle={selectedHandle}
                    selectedPart={selectedPart}
                    onElementClick={handleElementClick}
                    onContentsChange={onChange}
                    onHandlePlace={handleHandlePlace}
                    onHandleSelect={setSelectedHandle}
                  />
                )}
              </Form.Field>
              {hasSVG && <ValueBar value={preview} onChange={setPreview} />}
            </Flex.Box>
            {hasSVG && (
              <Flex.Box
                className={CSS.BE("schematic", "sidebar")}
                y
                rounded="small"
                background={0}
                bordered
                borderColor={6}
              >
                {shown?.type === "command" ? (
                  <CommandInspector key={shown.key} command={shown.key} />
                ) : shown?.type === "click" ? (
                  <ClickInspector key={shown.key} click={shown.key} />
                ) : (
                  <Inspector
                    key={`${selectedPart ?? ""}/${preview.state}`}
                    part={selectedPart}
                    preview={preview}
                  />
                )}
              </Flex.Box>
            )}
          </Flex.Box>
        </Modal.Body>
        {hasSVG && (
          <Modal.Footer background={0}>
            <Nav.Bar.End>
              <Button.Button
                variant="filled"
                status={
                  !isCreate && stored.variant === "loading"
                    ? "loading"
                    : status.keepVariants(variant, "loading")
                }
                disabled={!isCreate && stored.variant === "error"}
                onClick={() => void handleSave()}
                trigger={Triggers.SAVE}
                triggerIndicator
              >
                {createSaveText}
              </Button.Button>
            </Nav.Bar.End>
          </Modal.Footer>
        )}
      </Modal.Frame>
    </Form.Form>
  );
};

export const useModal = Modals.create<ModalParams>(Editor);
