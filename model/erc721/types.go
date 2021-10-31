package erc721

type SwapPairRegisterEventState string

type SwapPairCreateState string

const (
	SwapPairRegisterEventStateOngoing   SwapPairRegisterEventState = "ongoing"
	SwapPairRegisterEventStateConfirmed SwapPairRegisterEventState = "confirmed"
	SwapPairRegisterEventStateFailed    SwapPairRegisterEventState = "failed"
)

const (
	SwapPairCreateStateCreated   SwapPairCreateState = "created"
	SwapPairCreateStateSent      SwapPairCreateState = "sent"
	SwapPairCreateStateConfirmed SwapPairCreateState = "confirmed"
	SwapPairCreateStateFailed    SwapPairCreateState = "failed"
	SwapPairCreateStateMissing   SwapPairCreateState = "missing"
)
