package quickbuddy
/*
#include <unistd.h>
*/
import "C"
import (
	"errors"
)

func TTYName(fd int) (string, error) {
	var t *C.char = C.ttyname(C.int(fd))
	if t == nil {
		return "", errors.New("error calling ttyname()")
	}
	s := C.GoString(t)
	return s, nil
}
