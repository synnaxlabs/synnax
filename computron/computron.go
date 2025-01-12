package computron

/*
#cgo CFLAGS: -I${SRCDIR}/python_install/include/python3.11 -I${SRCDIR}/python_install/lib/python3.11/site-packages/numpy/core/include
#cgo darwin CFLAGS: -mmacosx-version-min=14.0

#cgo linux LDFLAGS: -L${SRCDIR}/python_install/lib/combined -lpython3.11-combined -ldl
#cgo darwin LDFLAGS: -mmacosx-version-min=14.0 -L${SRCDIR}/python_install/lib/combined -lpython3.11-combined -ldl
#cgo windows LDFLAGS: -L${SRCDIR}/python_install/lib/combined -lpython311

#define PY_SSIZE_T_CLEAN
#define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
#include <Python.h>

#include <numpy/arrayobject.h>
#include <stdlib.h>
#include <string.h>

static PyObject* warning_module;
static char* current_warning;

static int init_numpy() { import_array1(-1); return 0;  }

static int is_array(PyObject* obj) { return PyArray_Check(obj); }

static void py_incref(PyObject* obj) { Py_INCREF(obj); }

static void py_decref(PyObject* obj) { Py_DECREF(obj); }

void set_dict_items(PyObject* dict, char** keys, PyObject** values, int count) {
   for (int i = 0; i < count; i++) {
	   py_incref(values[i]); // Increment reference count for value
	   PyDict_SetItemString(dict, keys[i], values[i]);
   }
}

static PyObject* wrapped_PyCompileString(const char *str, const char *filename, int start) {
   return Py_CompileString(str, filename, start);
}

static PyObject* wrapped_PyArray_SimpleNew(int nd, npy_intp* dims, int typenum) {
   return PyArray_SimpleNew(nd, dims, typenum);
}

static int wrapped_PyArray_TYPE(PyArrayObject* arr) { return PyArray_TYPE(arr); }

static int wrapped_PyArray_NDIM(PyArrayObject* arr) { return PyArray_NDIM(arr); }

static npy_intp* wrapped_PyArray_DIMS(PyArrayObject* arr) { return PyArray_DIMS(arr);}

static int wrapped_PyArray_ITEMSIZE(PyArrayObject* arr) { return PyArray_ITEMSIZE(arr); }

static void* wrapped_PyArray_DATA(PyArrayObject* arr) { return PyArray_DATA(arr); }

static char* get_py_error() {
    PyObject *ptype, *pvalue, *ptraceback;
    PyErr_Fetch(&ptype, &pvalue, &ptraceback);
    if (!pvalue) {
        return NULL;
    }

    // Normalize the exception
    PyErr_NormalizeException(&ptype, &pvalue, &ptraceback);

    // Get the string representation of the error
    PyObject* str = PyObject_Str(pvalue);
    if (!str) {
        return NULL;
    }

    // Convert PyUnicode to C string
    const char* error = PyUnicode_AsUTF8(str);
    if (!error) {
        Py_DECREF(str);
        return NULL;
    }

    // Make a copy of the error string
    char* result = strdup(error);

    // Clean up
    Py_DECREF(str);
    if (ptraceback) {
        PyObject* module = PyImport_ImportModule("traceback");
        if (module != NULL) {
            PyObject* format_tb = PyObject_GetAttrString(module, "format_tb");
            if (format_tb != NULL) {
                PyObject* tb_list = PyObject_CallFunctionObjArgs(format_tb, ptraceback, NULL);
                if (tb_list != NULL) {
                    // Convert traceback list to string
                    PyObject* tb_str = PyObject_Str(tb_list);
                    if (tb_str != NULL) {
                        const char* tb_chars = PyUnicode_AsUTF8(tb_str);
                        if (tb_chars != NULL) {
                            char* new_result = malloc(strlen(result) + strlen(tb_chars) + 2);
                            sprintf(new_result, "%s\n%s", result, tb_chars);
                            free(result);
                            result = new_result;
                        }
                        Py_DECREF(tb_str);
                    }
                    Py_DECREF(tb_list);
                }
                Py_DECREF(format_tb);
            }
            Py_DECREF(module);
        }
    }

    // Restore the error state
    PyErr_Restore(ptype, pvalue, ptraceback);

    return result;
}


static void warning_callback(const char* message) {
    if (current_warning != NULL) {
        free(current_warning);
    }
    current_warning = strdup(message);
}

static PyObject* custom_warning_handler(PyObject *self, PyObject *args, PyObject *kwargs) {
    PyObject *message, *category, *filename, *lineno, *file, *line;

    // Accept all 6 arguments but only require the first 4
    if (!PyArg_UnpackTuple(args, "custom_warning_handler", 4, 6,
                          &message, &category, &filename, &lineno, &file, &line)) {
        return NULL;
    }

    // Convert to string
    PyObject* str_message = PyObject_Str(message);
    PyObject* str_filename = PyObject_Str(filename);
    PyObject* str_lineno = PyObject_Str(lineno);
    PyObject* str_category = PyObject_Str(category);

    if (str_message != NULL && str_filename != NULL && str_lineno != NULL && str_category != NULL) {
        const char* warning_text = PyUnicode_AsUTF8(str_message);
        const char* fname = PyUnicode_AsUTF8(str_filename);
        const char* lno = PyUnicode_AsUTF8(str_lineno);
        const char* cat = PyUnicode_AsUTF8(str_category);

        if (warning_text != NULL && fname != NULL && lno != NULL && cat != NULL) {
            // Format warning string similar to Python's warning format
            char* formatted = malloc(strlen(warning_text) + strlen(fname) + strlen(lno) + strlen(cat) + 50);
            sprintf(formatted, "%s:%s: %s: %s", fname, lno, cat, warning_text);
            warning_callback(formatted);
            free(formatted);
        }

        Py_DECREF(str_message);
        Py_DECREF(str_filename);
        Py_DECREF(str_lineno);
        Py_DECREF(str_category);
    }

    Py_RETURN_NONE;
}

static PyMethodDef warning_handler_def = {
    "custom_warning_handler",
    (PyCFunction)custom_warning_handler,
    METH_VARARGS | METH_KEYWORDS,
    NULL
};

static void setup_warning_handler(void) {
    warning_module = PyImport_ImportModule("warnings");
    if (warning_module != NULL) {
        PyObject* new_handler = PyCFunction_New(&warning_handler_def, NULL);
        if (new_handler != NULL) {
            PyObject_SetAttrString(warning_module, "showwarning", new_handler);
            Py_DECREF(new_handler);
        }
    }
}

static const char* get_current_warning(void) {
    if (current_warning != NULL) {
        const char* warning = current_warning;
        current_warning = NULL;  // Clear the warning after reading
        return warning;
    }
    return NULL;
}

*/
import "C"
import (
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	xembed "github.com/synnaxlabs/x/embed"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	xsync "github.com/synnaxlabs/x/sync"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

const (
	targetPythonVersion             = "3.11.7"
	dirPerm             os.FileMode = 0755
	filePerm            os.FileMode = 0644
)

const pythonLibPath string = "" +
	// windows
	`%[1]s\lib\python3.11;%[1]s\lib\python3.11\site-packages;%[1]s\lib\combined` +
	// !windows
	`/lib/python3.11:%[1]s/lib/python3.11/site-packages:%[1]s/lib/combined`

func getPythonPath(installDir string) string {
	return fmt.Sprintf(pythonLibPath, installDir)
}

type Config struct {
	// Instrumentation is used for logging, tracing, and metrics
	alamos.Instrumentation
	// PythonInstallDir is the directory where the embedded Python installation is
	// extracted.
	// [OPTIONAL] [DEFAULT: /tmp/synnax/computron]
	PythonInstallDir string
}

var (
	_ config.Config[Config] = Config{}
	//DefaultConfig is the default configuration for the computron service.
	DefaultConfig = Config{
		PythonInstallDir: filepath.Join(os.TempDir(), "synnax", "computron"),
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("computron")
	validate.NotEmptyString(v, "PythonInstallDir", c.PythonInstallDir)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.PythonInstallDir = override.String(c.PythonInstallDir, other.PythonInstallDir)
	return c
}

type Interpreter struct {
	cfg     Config
	globals *C.PyObject
}

// lockThreadAndGIL locks the current thread and the Python Global Interpreter Lock (GIL).
func lockThreadAndGIL() func() {
	runtime.LockOSThread()
	gilState := C.PyGILState_Ensure()
	return func() {
		C.PyGILState_Release(gilState)
		runtime.UnlockOSThread()
	}
}

// New creates a new embedded Python interpreter.
func New(cfgs ...Config) (*Interpreter, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	cfg.L.Info("starting embedded Python service",
		zap.String("install_dir", cfg.PythonInstallDir),
		zap.String("version", targetPythonVersion),
	)
	s := &Interpreter{cfg: cfg}
	if err = s.initPython(); err != nil {
		return nil, err
	}
	err = s.initGlobals()
	if err != nil {
		return nil, err
	}
	cfg.L.Info("embedded Python service started successfully")
	return s, nil
}

func (s *Interpreter) initPython() error {
	// Check if the directory exists
	installDir := filepath.Join(s.cfg.PythonInstallDir, "python_install")
	if _, err := os.Stat(s.cfg.PythonInstallDir); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check Python installation directory")
	}
	// Read the contents of the VERSION file
	contents, err := os.ReadFile(filepath.Join(installDir, "VERSION"))
	v := string(contents)
	v = strings.ReplaceAll(v, " ", "")
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to read Python version file")
	}
	if strings.Contains(v, targetPythonVersion) {
		// Check if the version is the same as the embedded Python version. If so,
		// everything is already set up and we can return early.
		s.cfg.L.Info("Python already installed. skipping installation")
	} else {
		s.cfg.L.Info("extracting embedded Python installation. this may take a few seconds")
		if err = xembed.Extract(
			embeddedPython,
			s.cfg.PythonInstallDir,
			dirPerm,
			filePerm,
		); err != nil {
			return errors.Newf("failed to extract embedded Python files: %v", err)
		}
		s.cfg.L.Info("embedded Python installation extracted")
	}

	if err := os.Setenv("PYTHONPATH", getPythonPath(installDir)); err != nil {
		return errors.Wrapf(err, "failed to set PYTHONPATH")
	}

	if err := os.Setenv("PYTHONHOME", installDir); err != nil {
		return errors.Wrapf(err, "failed to set PYTHONHOME")
	}

	// Lock the OS thread before initializing Python
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	C.Py_Initialize()
	// Threads are automatically initialized for python interpreter for python 3.11+
	C.setup_warning_handler()

	if res := C.init_numpy(); res != 0 {
		return errors.New("failed to initialize NumPy")
	}

	C.PyEval_SaveThread()

	return nil
}

// initGlobals initializes global variables for the Python interpreter, specifically NumPy.
func (s *Interpreter) initGlobals() error {
	unlock := lockThreadAndGIL()
	defer unlock()

	s.globals = C.PyDict_New()
	if s.globals == nil {
		return errors.Newf("failed to create Python globals dictionary")
	}
	numpyName := C.CString("numpy")
	defer C.free(unsafe.Pointer(numpyName))
	numpyModule := C.PyImport_ImportModule(numpyName)
	if numpyModule == nil {
		if errStr := C.get_py_error(); errStr != nil {
			defer C.free(unsafe.Pointer(errStr))
			return errors.Newf("Python error importing numpy: %s", C.GoString(errStr))
		}
		return errors.New("failed to import numpy")
	}
	npKey := C.CString("np")
	defer C.free(unsafe.Pointer(npKey))
	C.PyDict_SetItemString(s.globals, npKey, numpyModule)
	C.py_decref(numpyModule)
	return nil
}

// Calculation represents a compiled Python calculation.
type Calculation struct {
	// Python namespace with pre-imported modules.
	globals *C.PyObject
	// The compiled Python bytecode.
	compiled *C.PyObject
}

// NewCalculation takes a Python code string and compiles it into a Calculation object.
func (s *Interpreter) NewCalculation(code string) (*Calculation, error) {
	compiled, err := compile(code)
	if err != nil {
		return nil, err
	}
	return &Calculation{compiled: compiled, globals: s.globals}, nil
}

// cache to avoid recompiling the same code
var compiledCodeCache xsync.Map[string, *C.PyObject]

// compile takes a Python code string and returns a compiled Python code object (*C.PyObject).
func compile(code string) (*C.PyObject, error) {
	unlock := lockThreadAndGIL()
	defer unlock()

	if compiledCode, exists := compiledCodeCache.Load(code); exists {
		return compiledCode, nil
	}
	cCode := C.CString(code)
	defer C.free(unsafe.Pointer(cCode))
	filename := C.CString("<string>")
	defer C.free(unsafe.Pointer(filename))
	compiledCode := C.wrapped_PyCompileString(cCode, filename, C.Py_file_input)
	if compiledCode == nil {
		if errStr := C.get_py_error(); errStr != nil {
			defer C.free(unsafe.Pointer(errStr))
			return nil, errors.Newf("Python compilation error: %s", C.GoString(errStr))
		}
		return nil, errors.New("failed to compile code")
	}
	// Increase the reference count to keep it in the cache
	C.py_incref(compiledCode)
	compiledCodeCache.Store(code, compiledCode)
	return compiledCode, nil
}

func (c *Calculation) Run(vars map[string]interface{}) (telem.Series, error) {
	series, _, err := c.RunWarning(vars)
	return series, err
}

// Run executes Python code and returns a telem.Series
func (c *Calculation) RunWarning(vars map[string]interface{}) (telem.Series, string,
	error) {
	var s telem.Series

	unlock := lockThreadAndGIL()
	defer unlock()

	localsC := C.PyDict_New()
	if localsC == nil {
		return s, "", errors.Newf("failed to create Python locals dictionary")
	}
	defer C.py_decref(localsC)

	// Set vars variables in locals
	if len(vars) > 0 {
		// Prepare arrays of keys and values
		var (
			count  = len(vars)
			keys   = make([]*C.char, count)
			values = make([]*C.PyObject, count)
			i      = 0
		)
		for k, v := range vars {
			ck := C.CString(k)
			keys[i] = ck
			pyObj, ok := v.(*C.PyObject)
			if !ok {
				for j := 0; j <= i; j++ {
					C.free(unsafe.Pointer(keys[j]))
				}
				return s, "", errors.Newf("value for key %s is not a *C.PyObject", k)
			}
			values[i] = pyObj
			i++
		}
		C.set_dict_items(localsC, &keys[0], &values[0], C.int(count))
		for i := 0; i < count; i++ {
			C.free(unsafe.Pointer(keys[i]))
		}
	}

	// Execute the compiled code object with locals
	ret := C.PyEval_EvalCode(c.compiled, c.globals, localsC)
	if ret == nil {
		if errStr := C.get_py_error(); errStr != nil {
			defer C.free(unsafe.Pointer(errStr))
			return s, "", errors.Newf("Python error: %s", C.GoString(errStr))
		}
		return s, "", errors.New("failed to execute code")
	}
	C.py_decref(ret) // Decrease ref count for the result of execution

	// Retrieve 'result' from locals
	cr := C.CString("result")
	defer C.free(unsafe.Pointer(cr))
	r := C.PyDict_GetItemString(localsC, cr)
	if r == nil {
		// If 'result' not in locals, check in globals (in case code modifies globals)
		r = C.PyDict_GetItemString(c.globals, cr)
		return s, "", errors.New("no 'result' variable in vars or locals")
	}
	// Increase reference count since we are going to use r
	C.py_incref(r)
	series, err := ToSeries(r)
	C.py_decref(r) // Decrease ref count after use

	warning := getCurrentWarning()
	return series, warning, err
}

// Map telem.DataType to NumPy data types
var (
	toNP = map[telem.DataType]int{
		telem.Uint8T:   C.NPY_UINT8,
		telem.Uint16T:  C.NPY_UINT16,
		telem.Uint32T:  C.NPY_UINT32,
		telem.Uint64T:  C.NPY_UINT64,
		telem.Int8T:    C.NPY_INT8,
		telem.Int16T:   C.NPY_INT16,
		telem.Int32T:   C.NPY_INT32,
		telem.Int64T:   C.NPY_INT64,
		telem.Float32T: C.NPY_FLOAT32,
		telem.Float64T: C.NPY_FLOAT64,
	}
	toDT = map[int]telem.DataType{
		C.NPY_UINT8:   telem.Uint8T,
		C.NPY_UINT16:  telem.Uint16T,
		C.NPY_UINT32:  telem.Uint32T,
		C.NPY_UINT64:  telem.Uint64T,
		C.NPY_INT8:    telem.Int8T,
		C.NPY_INT16:   telem.Int16T,
		C.NPY_INT32:   telem.Int32T,
		C.NPY_INT64:   telem.Int64T,
		C.NPY_FLOAT32: telem.Float32T,
		C.NPY_FLOAT64: telem.Float64T,
	}
)

// NewSeries creates a NumPy array from a telem.Series
func NewSeries(s telem.Series) (*C.PyObject, error) {
	unlock := lockThreadAndGIL()
	defer unlock()

	dataType, ok := toNP[s.DataType]
	if !ok {
		return nil, errors.Newf("unsupported data type: %s", s.DataType)
	}
	if len(s.Data) == 0 {
		return nil, errors.Newf("empty data")
	}
	length := C.npy_intp(s.Len())

	// Create a new NumPy array
	arr := C.wrapped_PyArray_SimpleNew(1, &length, C.int(dataType))
	if arr == nil {
		return nil, errors.Newf("failed to create numpy array")
	}

	// Copy data from Go to NumPy array
	dataPtr := C.wrapped_PyArray_DATA((*C.PyArrayObject)(unsafe.Pointer(arr)))
	C.memcpy(dataPtr, unsafe.Pointer(&s.Data[0]), C.size_t(len(s.Data)))

	return arr, nil
}

// ToSeries converts a NumPy array to a telem.Series
func ToSeries(pyArray *C.PyObject) (telem.Series, error) {
	var s telem.Series
	if pyArray == nil {
		return s, errors.New("pyArray is nil")
	}

	unlock := lockThreadAndGIL()
	defer unlock()

	if C.is_array(pyArray) == 0 {
		return s, errors.New("cannot convert non-NumPy object to Series")
	}
	arr := (*C.PyArrayObject)(unsafe.Pointer(pyArray))
	npType := C.wrapped_PyArray_TYPE(arr)
	dt, found := toDT[int(npType)]
	if !found {
		return s, errors.Newf("unsupported numpy data type: %d", int(npType))
	}
	data := C.wrapped_PyArray_DATA(arr)
	if data == nil {
		return s, errors.Newf("failed to get data pointer from numpy array")
	}
	nDim := int(C.wrapped_PyArray_NDIM(arr))
	if nDim <= 0 {
		return s, errors.Newf("invalid number of dimensions: %d", nDim)
	}
	length := 1
	dims := (*[1 << 30]C.npy_intp)(unsafe.Pointer(C.wrapped_PyArray_DIMS(arr)))[:nDim:nDim]
	for i := 0; i < nDim; i++ {
		length *= int(dims[i])
	}
	itemSize := int(C.wrapped_PyArray_ITEMSIZE(arr))
	totalSize := length * itemSize

	dataBytes := make([]byte, totalSize)
	C.memcpy(unsafe.Pointer(&dataBytes[0]), data, C.size_t(totalSize))

	s.DataType = dt
	s.Data = dataBytes
	return s, nil
}

func getCurrentWarning() string {
	warning := C.get_current_warning()
	if warning != nil {
		return C.GoString(warning)
	}
	return ""
}
