import { ComponentType, forwardRef } from "react";
import { Space } from "../Space";
import { Input, InputProps } from "./Input";
import { InputHelpText } from "./InputHelpText";
import { InputLabel } from "./InputLabel";
import clsx from "clsx";

export interface InputItemProps extends Omit<InputProps, "children"> {
	label?: string;
	helpText?: string;
	children?: ComponentType<InputProps>;
}

export const InputItem = forwardRef<HTMLInputElement, InputItemProps>(
	(
		{
			label,
			helpText,
			style,
			className,
			children: Children = Input,
			...props
		}: InputItemProps,
		ref
	) => {
		return (
			<Space
				size="small"
				className={clsx("pluto-input-item", className)}
				direction="vertical"
				style={style}
			>
				<InputLabel>{label}</InputLabel>
				<Children ref={ref} {...props} />
				<InputHelpText>{helpText}</InputHelpText>
			</Space>
		);
	}
);

InputItem.displayName = "InputItem";
