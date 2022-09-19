import { Dispatch, SetStateAction, useEffect, useRef, useState } from "react";

export default function usePersistedState(
  name: string,
  defaultValue: number
): [number, React.Dispatch<SetStateAction<number>>] {
  const [value, setValue] = useState<number>(defaultValue);
  const nameRef = useRef(name);

  useEffect(() => {
    try {
      const storedValue = Number(localStorage.getItem(name));
      if (storedValue !== null) setValue(storedValue);
      else localStorage.setItem(name, defaultValue.toString());
    } catch {
      setValue(defaultValue);
    }
  }, []);

  useEffect(() => {
    try {
      localStorage.setItem(nameRef.current, value.toString());
    } catch {}
  }, [value]);

  useEffect(() => {
    const lastName = nameRef.current;
    if (name !== lastName) {
      try {
        localStorage.setItem(name, value.toString());
        nameRef.current = name;
        localStorage.removeItem(lastName);
      } catch {}
    }
  }, [name]);

  return [value, setValue as Dispatch<SetStateAction<number>>];
};
