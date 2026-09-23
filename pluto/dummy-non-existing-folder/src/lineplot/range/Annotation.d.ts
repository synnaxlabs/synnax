import { type z } from "zod";
import { Aether } from "../../aether";
import { range } from "./aether";
interface AnnotationProps extends z.input<typeof range.annotationStateZ>, Aether.ComponentProps {
}
export declare const Annotation: ({ aetherKey, ...rest }: AnnotationProps) => null;
export {};
//# sourceMappingURL=Annotation.d.ts.map