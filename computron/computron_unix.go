//go:build !windows

package computron

/*
#cgo CFLAGS: -I${SRCDIR}/python_install/include/python3.9 -I${SRCDIR}/python_install/lib/python3.9/site-packages/numpy/core/include
#cgo LDFLAGS: -L${SRCDIR}/python_install/lib/combined -lpython3.9-combined -ldl
*/
