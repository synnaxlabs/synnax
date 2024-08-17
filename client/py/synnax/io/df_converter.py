#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pandas as pd

booleanMap = {
    "True": True,
    "False": False,
    "TRUE": True,
    "FALSE": False,
    "true": True,
    "false": False,
}


def convert_df(df: pd.DataFrame) -> pd.DataFrame:
    for col in df.columns:
        all_bools = all(str(val) in booleanMap for val in df[col])
        if all_bools:
            df[col] = df[col].map(lambda x: booleanMap[str(x)])
    return df
