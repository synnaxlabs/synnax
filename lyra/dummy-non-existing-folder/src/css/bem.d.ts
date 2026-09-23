export interface BEM {
    /** @returns the class name for a block. */
    B: (...blocks: string[]) => string;
    /** @returns the class name for an element of the enclosing block. */
    E: (element: string) => string;
    /** @returns the class name for a modifier. */
    M: (...modifiers: string[]) => string;
    /** @returns the class name for an element of the given block. */
    BE: (block: string, ...elements: string[]) => string;
    /** @returns the class name for a modifier of the given block. */
    BM: (block: string, ...modifiers: string[]) => string;
    /** @returns the class name for a modified element of the given block. */
    BEM: (block: string, element: string, ...modifiers: string[]) => string;
    /** @returns the name of a custom property, including the leading dashes. */
    variable: (...variables: string[]) => string;
}
export declare const newBEM: (prefix: string) => BEM;
//# sourceMappingURL=bem.d.ts.map