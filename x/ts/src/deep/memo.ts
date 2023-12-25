import { equal } from "@/deep/equal"


export const memo = <F extends (...args: any[]) => any>(func: F): F => {
    let prevArgs: Parameters<F> = undefined as any
    let prevResult: ReturnType<F> = undefined as any
    const v = ((...args: Parameters<F>) => {
        if (equal(prevArgs, args)) return prevResult
        const result = func(...args)
        prevArgs = args
        prevResult = result
        return result
    })
    return v as F;
}