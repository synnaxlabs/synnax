// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo, useRef } from "react";

import { PID as PPID, PIDProps, PIDElementProps } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { ELEMENTS } from "../elements";
import { useSelectPID, useSelectPIDElementProps } from "../store/selectors";
import {
  setPIDEdges,
  setPIDEditable,
  setPIDElementProps,
  setPIDNodes,
  setPIDViewport,
} from "../store/slice";

import { LayoutRenderer } from "@/layout";

const PIDElementRenderer = ({
  elementKey,
  position,
  selected,
  layoutKey,
  editable,
  zoom,
}: PIDElementProps & { layoutKey: string }): ReactElement | null => {
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

  const C = ELEMENTS[type];

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
  const pid = useSelectPID(layoutKey);
  const dispatch = useDispatch();

  const handleEdgesChange: PIDProps["onEdgesChange"] = useCallback(
    (edges) => {
      dispatch(setPIDEdges({ layoutKey, edges }));
    },
    [dispatch, layoutKey]
  );

  const handleNodesChange: PIDProps["onNodesChange"] = useCallback(
    (nodes) => {
      dispatch(setPIDNodes({ layoutKey, nodes }));
    },
    [dispatch, layoutKey]
  );

  const handleViewportChange: PIDProps["onViewportChange"] = useCallback(
    (vp) => {
      dispatch(setPIDViewport({ layoutKey, viewport: vp }));
    },
    [layoutKey]
  );

  const handleEditableChange: PIDProps["onEditableChange"] = useCallback(
    (cbk) => {
      dispatch(setPIDEditable({ layoutKey, editable: cbk(pid.editable) }));
    },
    [layoutKey, pid.editable]
  );

  const pidElementRenderer = useCallback(
    (props: PIDElementProps) => {
      return <PIDElementRenderer layoutKey={layoutKey} {...props} />;
    },
    [layoutKey]
  );

  return (
    <PPID
      onViewportChange={handleViewportChange}
      edges={pid.edges}
      nodes={pid.nodes}
      viewport={pid.viewport}
      onEdgesChange={handleEdgesChange}
      onNodesChange={handleNodesChange}
      onEditableChange={handleEditableChange}
      editable={pid.editable}
    >
      {pidElementRenderer}
    </PPID>
  );
};
