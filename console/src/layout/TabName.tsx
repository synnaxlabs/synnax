// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { useNameHook } from "@/layout/context";
import { type UseName } from "@/layout/slice";

export interface TabNameProps extends Tabs.NameProps {
  // type selects the per-type name hook (Renderer.useName). When no hook is
  // registered for the type, the tab shows its static name with no rename.
  type: string;
  // nameKey is the key the name is resolved and renamed by — typically the backing
  // resource's key, which may differ from the tab's identity (tabKey).
  nameKey: string;
  // onNameChange, when set, receives the resolved name whenever it changes, so
  // consumers can mirror it into another store (e.g. the layout slice).
  onNameChange?: (name: string) => void;
}

interface ResolvedTabNameProps extends Omit<TabNameProps, "type"> {
  useName: UseName;
}

const ResolvedTabName = ({
  useName,
  nameKey,
  tabKey,
  name: initialName,
  onRename: externalRename,
  onNameChange,
  ...rest
}: ResolvedTabNameProps): ReactElement => {
  const [name, setName] = useState(initialName);
  const handleResolved = useCallback(
    (next: string) => {
      setName(next);
      onNameChange?.(next);
    },
    [onNameChange],
  );
  const { onRename, retrieve } = useName(nameKey, handleResolved);
  useEffect(() => {
    retrieve();
  }, [retrieve]);
  const handleRename = useCallback(
    (_: string, next: string) => {
      handleResolved(next);
      externalRename?.(tabKey, next);
      onRename(next);
    },
    [handleResolved, externalRename, tabKey, onRename],
  );
  return (
    <Tabs.DefaultName tabKey={tabKey} name={name} onRename={handleRename} {...rest} />
  );
};

// TabName renders a tab's display name for a layout/resource of the given type,
// resolving the name through that type's registered name hook and keeping it in
// sync. Rename writes back through the same hook. When the type has no name hook,
// the tab falls back to its static name with no rename affordance.
export const TabName = ({ type, ...props }: TabNameProps): ReactElement => {
  const useName = useNameHook(type);
  if (useName == null) return <Tabs.DefaultName {...props} />;
  return <ResolvedTabName key={type} useName={useName} {...props} />;
};
