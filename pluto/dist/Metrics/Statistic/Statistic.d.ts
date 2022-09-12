import { textLevel } from "../../Atoms/Typography/Text";
export interface ValueProps {
    value: number;
    level?: textLevel;
    label?: string;
    variant?: "primary" | "error";
    color?: string;
}
export declare const Statistic: ({ value, level, variant, label, color, }: ValueProps) => JSX.Element;
export default Statistic;
