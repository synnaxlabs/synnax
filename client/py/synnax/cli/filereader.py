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
    def __init__(self, filepath):
        self.filepath = filepath

    def getData(self):
        return np.genfromtxt(self.filepath, delimiter=",")


class XLSXReader:
    def __init(self, filepath):
        self.filepath = filepath

    def getData(self):
        return np.array(pd.read_excel(self.filepath))
