// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
