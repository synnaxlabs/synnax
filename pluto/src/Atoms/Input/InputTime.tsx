import { ComponentSize } from "@/util";

export interface InputTimeProps {
  size?: ComponentSize;
  onChange?: (value: string) => void;
}

export const InputTime = ({ size, onChange }: InputTimeProps) => {
  return (
    <input
      type="time"
      step="1"
      onChange={(e) => onChange && onChange(e.target.value)}
      className={`pluto-input__input pluto-input__input--${size}`}
    />
  );
};
