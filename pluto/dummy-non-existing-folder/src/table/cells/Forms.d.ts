import "./Forms.css";
import { Select } from "@synnaxlabs/lyra/select";
import { type Variant } from "./registry";
export interface FormProps {
    onVariantChange: (variant: Variant) => void;
}
export declare const ValueForm: ({ onVariantChange }: FormProps) => import("react").JSX.Element;
export declare const TextForm: ({ onVariantChange }: FormProps) => import("react").JSX.Element;
export interface SelectVariantProps extends Omit<Select.StaticProps<Variant>, "data" | "resourceName"> {
}
export declare const SelectVariant: ({ className, ...rest }: SelectVariantProps) => import("react").JSX.Element;
//# sourceMappingURL=Forms.d.ts.map