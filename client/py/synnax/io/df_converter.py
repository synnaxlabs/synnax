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
