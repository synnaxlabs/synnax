#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import concurrent.futures


def _process(thread: int, chunk: list, f):
    for i, p in enumerate(chunk):
        print(f"Thread {thread} processing {i} of {len(chunk)}")
        chunk[i] = f(i, p)
    return chunk


def process_products(products: list, f, workers=10):
    pool = concurrent.futures.ThreadPoolExecutor(max_workers=workers)
    # split the products into chunks
    chunk_size = len(products) // workers
    chunks = [products[i : i + chunk_size] for i in range(0, len(products), chunk_size)]
    futures = [pool.submit(_process, i, chunk, f) for i, chunk in enumerate(chunks)]
    concurrent.futures.wait(futures)
    out = list()
    for future in futures:
        out.extend(future.result())
    return out


def square(i, n):
    return n * n
