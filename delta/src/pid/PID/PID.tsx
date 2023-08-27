// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useRef } from "react";

import { Icon } from "@synnaxlabs/media";
import { PID as Core, PIDElement, Control, Button } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { LayoutRenderer, useSelectRequiredLayout } from "@/layout";
import { useSelectPID, useSelectPIDElementProps } from "@/pid/store/selectors";
import {
  togglePIDControl,
  setPIDControlState,
  setPIDEdges,
  setPIDEditable,
  setPIDElementProps,
  setPIDNodes,
  setPIDViewport,
} from "@/pid/store/slice";

const PIDElementRenderer = ({
  elementKey,
  position,
  selected,
  layoutKey,
  editable,
  zoom,
}: Core.ElementProps & { layoutKey: string }): ReactElement | null => {
  const el = useSelectPIDElementProps(layoutKey, elementKey);
  if (el == null) return null;
  const {
    props: { type, ...props },
  } = el;
  const dispatch = useDispatch();

  const handleChange = useCallback(
    (props: object) => {
      dispatch(
        setPIDElementProps({ layoutKey, key: elementKey, props: { type, ...props } })
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

export const PID: LayoutRenderer = ({ layoutKey }) => {
  const { name } = useSelectRequiredLayout(layoutKey);
  const pid = useSelectPID(layoutKey);
  const dispatch = useDispatch();

  const handleEdgesChange: Core.PIDProps["onEdgesChange"] = useCallback(
    (edges) => {
      dispatch(setPIDEdges({ layoutKey, edges }));
    },
    [dispatch, layoutKey]
  );

  const handleNodesChange: Core.PIDProps["onNodesChange"] = useCallback(
    (nodes) => {
      dispatch(setPIDNodes({ layoutKey, nodes }));
    },
    [dispatch, layoutKey]
  );

  const handleViewportChange: Core.PIDProps["onViewportChange"] = useCallback(
    (vp) => {
      dispatch(setPIDViewport({ layoutKey, viewport: vp }));
    },
    [layoutKey]
  );

  const handleEditableChange: Core.PIDProps["onEditableChange"] = useCallback(
    (cbk) => {
      dispatch(setPIDEditable({ layoutKey, editable: cbk }));
    },
    [layoutKey]
  );

  const handleControlStatusChange: Control.ControllerProps["onStatusChange"] =
    useCallback(
      (control) => {
        dispatch(setPIDControlState({ layoutKey, control }));
      },
      [layoutKey]
    );

  const acquireControl = useCallback(
    (v: boolean) => {
      dispatch(
        togglePIDControl({
          layoutKey,
          status: v ? "acquired" : "released",
        })
      );
    },
    [layoutKey]
  );

  const elRenderer = useCallback(
    (props: Core.ElementProps) => {
      return <PIDElementRenderer layoutKey={layoutKey} {...props} />;
    },
    [layoutKey]
  );

  return (
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
      >
        <Core.NodeRenderer>{elRenderer}</Core.NodeRenderer>
        <Core.Background />
        <Core.Controls reverse>
          <Button.ToggleIcon
            value={pid.control === "acquired"}
            onChange={acquireControl}
          >
            <Icon.Circle fill="white" />
          </Button.ToggleIcon>
        </Core.Controls>
      </Core.PID>
    </Control.Controller>
  );
};
