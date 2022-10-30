from os import path

import numpy as np
import pandas as pd

from .argreader import Filetype

# Uses factory method to get correct filereader - This supports modularity and adding support for different filetypes down the road


class FileReader:
    @staticmethod
    def get(ftype, filepath):
        if ftype is Filetype.CSV:
            return CSVReader(filepath)
        elif ftype is Filetype.XLSX:
            return XLSXReader(filepath)
        else:
            print("Invalid filetype! (Should never get here!)")
            exit(-5)


class CSVReader:
    CHUNKSIZE = 10**6

    def __init__(self, filepath):
        self.filepath = filepath

    def get_data(self):
        return

    def get_cols_sample(self, col):
        return pd.read_csv(self.filepath, nrows=10)[col]

    def get_headers(self):
        return pd.read_csv(self.filepath, nrows=10).columns

    def push_data_chunks(self, converter):
        for chunk in pd.read_csv(self.filepath, chunksize=self.CHUNKSIZE):
            converter.parseChunk(chunk)


class XLSXReader:
    def __init(self, filepath):
        self.filepath = filepath
        self.headers = pd.read_csv(
            filepath,
        )

    def getHeaders(self):
        pass

    def get_data(self):
        return np.array(pd.read_excel(self.filepath))

    def push_data(self, converter):
        for chunk in pd.read_excel(self.filepath, chunksize=self.CHUNKSIZE):
            converter.parseChunk(chunk)
