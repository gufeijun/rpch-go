package rpch

import "errors"

var (
	errShortRead         = newProtoError("rpch: short read")
	errInvalidMagic      = newProtoError("rpch: invalid magic number")
	errInvalidKind       = newProtoError("rpch: invalid type kind")
	errBadRequestLine    = newProtoError("rpch: invalid request line")
	errBadRequestService = newProtoError("rpch: request non-existent service")
	errBadRequestMethod  = newProtoError("rpch: request non-existent method ")
	errBadRequestMessage = newProtoError("rpch: unrecognized request message")
	errBadRequestType    = newProtoError("rpch: unrecognized request builtin type")
	errBadRequestArgCnt  = newProtoError("rpch: request argument count dose not confirm to method signature")
	errBadStreamType     = newProtoError("rpch: unrecognized stream type")
)

var (
	errShortWrite           = errors.New("rpch: short write")
	errClientClosed         = errors.New("rpch: call on a closed client")
	errClientMultipleStream = errors.New("rpch: should at most have one stream request arg")
	ErrBadResponse          = errors.New("rpch: response can not be nil at the same time error is nil too")
)

type protoError struct {
	errMsg string
}

func newProtoError(err string) *protoError {
	return &protoError{
		errMsg: err,
	}
}

func (pe *protoError) Error() string {
	return pe.errMsg
}
