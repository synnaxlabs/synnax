import numpy as np
import pandas as pd
from typing import Iterator
import synnax as sy

TOTAL_ROWS = 100_000_000
ROWS_PER_ITERATION = 100_000
ITERATIONS = TOTAL_ROWS // ROWS_PER_ITERATION
START_TIME = sy.TimeStamp.now()


def new_data_iterator(index: bool = False) -> Iterator[pd.DataFrame]:
    start = START_TIME
    end = start + (ROWS_PER_ITERATION * sy.TimeSpan.MICROSECOND)
    timestamps = np.linspace(start, end, ROWS_PER_ITERATION, dtype=np.int64)
    values = np.linspace(0, 1, ROWS_PER_ITERATION, dtype=np.float32)
    for i in range(ITERATIONS):
        timestamps += (ROWS_PER_ITERATION * sy.TimeSpan.MICROSECOND)
        if index:
            df = pd.DataFrame({'data': values}, index=timestamps)
            df.index.name = 'time'
            yield df
        else:
            yield pd.DataFrame({'data': values, 'time': timestamps})
