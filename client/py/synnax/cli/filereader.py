from .argreader import Filetype

from numpy import genfromtext
from os import path

# Uses factory method to get correct filereader


class FileReader():
    @staticmethod
    def get(ftype, filepath):
        if ftype is Filetype.CSV:
            return CSVReader(filepath)
        elif ftype is Filetype.XLSX:
            return XLSXReader(filepath)
        else:
            print('Invalid filetype! (Should never get here!)')
            exit(-5)


class CSVReader():
    def __init__(self, filepath):
        self.filepath = filepath

    def getData(self):
        return genfromtxt(self.filepath, delimiter=',')


class XLSXReader():
    def __init(self, filepath):
        self.filepath = filepath
