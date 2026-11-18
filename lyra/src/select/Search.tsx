// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode, useEffect, useRef, useState } from "react";

import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { useSyncedRef } from "@/hooks/ref";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { useItemTerm } from "@/menu/filter";
import { useRegistryContext } from "@/select/registry";
import { useClosed, useInline } from "@/select/scope";

/** Props for {@link Search}. */
export interface SearchProps {
  placeholder?: string;
  /** Called as the user types, to filter the frame's data. */
  onSearch?: (term: string) => void;
  /** Buttons rendered after the field. */
  actions?: Input.TextProps["children"];
  loading?: boolean;
}

/**
 * The search field of a selection dialog. It hides the fixed {@link Item}s whose text
 * does not match, and passes the term to `onSearch` for the frame's data. The term
 * clears when the dialog closes, and `onSearch` then receives an empty term. While the
 * options show in their menu, the words of the filter of the menu that the name of the
 * submenu does not hold are the term, and no field shows.
 */
export const Search = ({
  placeholder = "Search...",
  onSearch,
  actions,
  loading = false,
}: SearchProps): ReactNode => {
  const closed = useClosed();
  const registry = useRegistryContext("Select.Search");
  const [term, setTerm] = useState("");
  const { variant } = Dialog.useContext();
  const onSearchRef = useSyncedRef(onSearch);
  const searchedRef = useRef(false);
  const inline = useInline();
  const itemTerm = useItemTerm();
  useEffect(() => {
    if (!inline) return;
    registry.setTerm(itemTerm);
    searchedRef.current = itemTerm !== "";
    onSearchRef.current?.(itemTerm);
  }, [inline, itemTerm, registry, onSearchRef]);
  useEffect(
    () => () => {
      registry.setTerm("");
      if (searchedRef.current) onSearchRef.current?.("");
      searchedRef.current = false;
    },
    [registry, onSearchRef],
  );
  if (closed || inline) return null;
  const input = (
    <Input.Search
      value={term}
      flush
      startContent={loading ? <Icon.Loading /> : <Icon.Search />}
      placeholder={placeholder}
      size={variant === "modal" ? "large" : "medium"}
      rounded
      grow
      full="x"
      onChange={(v) => {
        setTerm(v);
        searchedRef.current = v !== "";
        registry.setTerm(v);
        onSearch?.(v);
      }}
    />
  );
  if (actions == null) return input;
  return (
    <Flex.Box pack x>
      {input}
      {actions}
    </Flex.Box>
  );
};
