package service

// Error errors type of coap-gateway
type Error string

func (e Error) Error() string { return string(e) }

//ErrEmptyCARootPool ca root pool is empty
var ErrEmptyCARootPool = Error("CA Root pool is empty.")
