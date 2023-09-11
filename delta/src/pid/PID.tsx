// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo, useRef } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  PID as Core,
  PIDElement,
  Control,
  Button,
  Haul,
  Theming,
  Text,
  Viewport,
  Triggers,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { box, scale, xy } from "@synnaxlabs/x";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import {
  useSelect,
  useSelectElementProps,
  useSelectViewporMode,
} from "@/pid/selectors";
import {
  toggleControl,
  setControlStatus,
  setEdges,
  setEditable,
  setElementProps,
  setNodes,
  setViewport,
  addElement,
  copySelection,
  calculatePos,
  pasteSelection,
} from "@/pid/slice";

const ElementRenderer = ({
  elementKey,
  position,
  selected,
  layoutKey,
  editable,
  zoom,
}: Core.ElementProps & { layoutKey: string }): ReactElement | null => {
  const el = useSelectElementProps(layoutKey, elementKey);
  if (el == null) return null;
  const {
    props: { type, ...props },
  } = el;
  const dispatch = useDispatch();

  const handleChange = useCallback(
    (props: object) => {
      dispatch(
        setElementProps({ layoutKey, key: elementKey, props: { type, ...props } })
      );
    },
    [dispatch, elementKey, layoutKey, type]
  );

  const C = PIDElement.REGISTRY[type];

  const refZoom = useRef(zoom);

  return (
    <C.Element
      position={position}
      selected={selected}
      onChange={handleChange}
      editable={editable}
      zoom={refZoom.current}
      {...props}
    />
  );
};

export const PID: Layout.Renderer = ({ layoutKey }) => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const pid = useSelect(layoutKey);
  const dispatch = useDispatch();
  const theme = Theming.use();
  const viewportRef = useSyncedRef(pid.viewport);

  const handleEdgesChange: Core.PIDProps["onEdgesChange"] = useCallback(
    (edges) => {
      dispatch(setEdges({ layoutKey, edges }));
    },
    [dispatch, layoutKey]
  );

  const handleNodesChange: Core.PIDProps["onNodesChange"] = useCallback(
    (nodes) => {
      dispatch(setNodes({ layoutKey, nodes }));
    },
    [dispatch, layoutKey]
  );

  const handleViewportChange: Core.PIDProps["onViewportChange"] = useCallback(
    (vp) => {
      dispatch(setViewport({ layoutKey, viewport: vp }));
    },
    [layoutKey]
  );

  const handleEditableChange: Core.PIDProps["onEditableChange"] = useCallback(
    (cbk) => {
      dispatch(setEditable({ layoutKey, editable: cbk }));
    },
    [layoutKey]
  );

  const handleControlStatusChange: Control.ControllerProps["onStatusChange"] =
    useCallback(
      (control) => {
        dispatch(setControlStatus({ layoutKey, control }));
      },
      [layoutKey]
    );

  const acquireControl = useCallback(
    (v: boolean) => {
      dispatch(
        toggleControl({
          layoutKey,
          status: v ? "acquired" : "released",
        })
      );
    },
    [layoutKey]
  );

  const elRenderer = useCallback(
    (props: Core.ElementProps) => {
      return <ElementRenderer layoutKey={layoutKey} {...props} />;
    },
    [layoutKey]
  );

  const ref = useRef<HTMLDivElement>(null);

  const handleDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      const valid = Haul.filterByType("pid-element", items);
      if (ref.current == null) return valid;
      const region = box.construct(ref.current);
      valid.forEach(({ key: type }) => {
        const spec = PIDElement.REGISTRY[type];
        if (spec == null) return;
        const zoomXY = xy.construct(pid.viewport.zoom);
        const s = scale.XY.translate(xy.scale(box.topLeft(region), -1))
          .magnify({
            x: 1 / zoomXY.x,
            y: 1 / zoomXY.y,
          })
          .translate(xy.scale(pid.viewport.position, -1));
        dispatch(
          addElement({
            layoutKey,
            key: nanoid(),
            node: {
              position: s.pos({ x: event.clientX, y: event.clientY }),
            },
            props: {
              type,
              ...spec.initialProps(theme),
            },
          })
        );
      });
      return valid;
    },
    [pid.viewport, theme]
  );

  const dropProps = Haul.useDrop({
    type: "PID",
    key: layoutKey,
    canDrop: Haul.canDropOfType("pid-element"),
    onDrop: handleDrop,
  });

  const mode = useSelectViewporMode();
  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  Triggers.use({
    triggers: [
      ["Control", "V"],
      ["Control", "C"],
    ],
    callback: useCallback(
      ({ triggers, cursor, stage }: Triggers.UseEvent) => {
        if (ref.current == null || stage !== "end") return;
        const region = box.construct(ref.current);
        const copy = triggers.some((t) => t.includes("C"));
        const pos = calculatePos(region, cursor, viewportRef.current);
        if (copy) dispatch(copySelection({ pos }));
        else dispatch(pasteSelection({ pos, layoutKey }));
      },
      [dispatch, layoutKey, viewportRef]
    ),
  });

  return (
    <div ref={ref} style={{ width: "inherit", height: "inherit" }}>
      <Control.Controller
        name={name}
        authority={1}
        acquireTrigger={pid.controlAcquireTrigger}
        onStatusChange={handleControlStatusChange}
      >
        <Core.PID
          onViewportChange={handleViewportChange}
          edges={pid.edges}
          nodes={pid.nodes}
          viewport={pid.viewport}
          onEdgesChange={handleEdgesChange}
          onNodesChange={handleNodesChange}
          onEditableChange={handleEditableChange}
          editable={pid.editable}
          triggers={triggers}
          {...dropProps}
        >
          <Core.NodeRenderer>{elRenderer}</Core.NodeRenderer>
          <Core.Background />
          <Core.Controls reverse>
            <Core.ToggleEditControl disabled={pid.control === "acquired"} />
            <Core.FitViewControl />
            <Button.ToggleIcon
              value={pid.control === "acquired"}
              onChange={acquireControl}
              tooltip={
                <Text.Text level="small">
                  {pid.control === "acquired" ? "Release control" : "Acquire control"}
                </Text.Text>
              }
            >
              <Icon.Circle fill="white" />
            </Button.ToggleIcon>
          </Core.Controls>
        </Core.PID>
      </Control.Controller>
    </div>
  );
};
