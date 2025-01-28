// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Modals.css";

import { Nav } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { useRemover } from "@/layout/hooks";
import { Modal } from "@/layout/Modal";
import { useSelectModals } from "@/layout/selectors";

export const Modals = () => {
  const layouts = useSelectModals();
  const remove = useRemover();
  return (
    <>
      {layouts.map(
        (l) =>
          l.location === "modal" && <Modal key={l.key} state={l} remove={remove} />,
      )}
    </>
  );
};

export interface ModalBarProps extends Nav.BarProps {}

export const BottomNavBar = ({ className, ...props }: ModalBarProps) => (
  <Nav.Bar
    location="bottom"
    size="8rem"
    className={CSS(CSS.B("bottom-nav-bar"), className)}
    {...props}
  />
);
