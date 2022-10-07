import { useEffect } from "react";
import { ComponentType, Key, useState } from "react";
import { useKeyHeld } from "../../util/useKeys";
import { TypedListEntry } from "./ListContext";

export interface SelectListProps<K extends any> {
  children: ComponentType<{
    selected: K[];
    setSelected: (K: K[]) => void;
  }>;
}

export default function SelectList<K extends any>({
  children: Children,
}: SelectListProps<K>) {
  const [selected, setSelected] = useState<K[]>([]);
  return <Children selected={selected} setSelected={setSelected} />;
}

export const useMultiSelect = <K extends Key, E extends TypedListEntry<K>>(
  data: E[]
): {
  selected: K[];
  onSelect: (key: K) => void;
  clearSelected: () => void;
} => {
  const [selected, setSelected] = useState<K[]>([]);
  const [shiftSelected, setShiftSelected] = useState<K | undefined>(undefined);
  const shiftPressed = useKeyHeld("Shift");

  useEffect(() => {
    if (!shiftPressed) {
      setShiftSelected(undefined);
    }
  }, [shiftPressed]);

  const onSelect = (key: K) => {
    if (shiftPressed && shiftSelected !== undefined) {
      const indexes = [
        data.findIndex((v) => v.key === key),
        data.findIndex((v) => v.key === shiftSelected),
      ].sort((a, b) => a - b);

      const nextKeys = data.slice(indexes[0], indexes[1] + 1).map((v) => v.key);

      if (
        nextKeys
          .slice(1, nextKeys.length - 1)
          .every((k) => selected.includes(k))
      ) {
        setSelected(selected.filter((k) => !nextKeys.includes(k)));
      } else {
        setSelected([...selected, ...nextKeys]);
      }
      setShiftSelected(undefined);
    } else {
      if (shiftPressed) setShiftSelected(key);
      if (selected.includes(key))
        setSelected(selected.filter((i) => i !== key));
      else setSelected([...selected, key]);
    }
  };

  const clearSelected = () => setSelected([]);
  return { selected, onSelect, clearSelected };
};

export const useHidden = <K extends Key>(): [K[], (key: K) => void] => {
  const [hidden, setHidden] = useState<K[]>([]);
  const onHide = (key: K) =>
    setHidden((hidden) =>
      hidden.includes(key) ? hidden.filter((i) => i !== key) : [...hidden, key]
    );
  return [hidden, onHide];
};
