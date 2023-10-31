import pandas as pd
import numpy as np

df = pd.read_csv("/Users/emilianobonilla/Downloads/4_9_wetdress_data.csv")

df = df.dropna()

# remove all columns that do not contain "Time", "vlv", "pressure", or "tc"
df = df[df.columns[df.columns.str.contains("Time|\.en|pressure|tc")]]

# remove all columns whose data is permanently 0
df = df.loc[:, (df != 0).any(axis=0)]
# remove all columns that have the same value for all rows
df = df.loc[:, df.nunique() != 1]
# remove all columns with 'tank'
df = df.loc[:, ~df.columns.str.contains("tnk")]

# clean column names, replacing '.' with "_" removing (hs), replacing "[" with "_", removing "]"
df.columns = df.columns.str.replace(".", "_")
df.columns = df.columns.str.replace("(hs)", "", regex=False)
df.columns = df.columns.str.replace("[", "_")
df.columns = df.columns.str.replace("]", "")

# trim all column names
df.columns = df.columns.str.strip()

df.to_csv("cleaned.csv", index=False)
