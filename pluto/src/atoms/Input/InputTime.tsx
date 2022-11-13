import { forwardRef } from "react";
import { ComponentSize } from "@/util";
import { InputProps } from "./Input";

export interface InputTimeProps extends InputProps {}

export const InputTime = forwardRef<HTMLInputElement, InputTimeProps>(
	({ size, ...props }: InputTimeProps, ref) => {
		return (
			<input
				ref={ref}
				type="time"
				step="1"
				className={`pluto-input__input pluto-input__input--${size}`}
				{...props}
			/>
		);
	}
);
