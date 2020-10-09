package rtmkt

// ProviderEvent is an event that occurs in a deal lifecycle on the provider
type ProviderEvent uint64

const (
	// ProviderEventOpen indicates a new deal was received from a client
	ProviderEventOpen ProviderEvent = iota

	// ProviderEventDealNotFound happens when the provider cannot find the piece for the
	// deal proposed by the client
	ProviderEventDealNotFound

	// ProviderEventDealRejected happens when a provider rejects a deal proposed
	// by the client
	ProviderEventDealRejected

	// ProviderEventDealAccepted happens when a provider accepts a deal
	ProviderEventDealAccepted

	// ProviderEventBlockSent happens when the provider reads another block
	// in the piece
	ProviderEventBlockSent

	// ProviderEventBlocksCompleted happens when the provider reads the last block
	// in the piece
	ProviderEventBlocksCompleted

	// ProviderEventPaymentRequested happens when a provider asks for payment from
	// a client for blocks sent
	ProviderEventPaymentRequested

	// ProviderEventSaveVoucherFailed happens when an attempt to save a payment
	// voucher fails
	ProviderEventSaveVoucherFailed

	// ProviderEventPartialPaymentReceived happens when a provider receives and processes
	// a payment that is less than what was requested to proceed with the deal
	ProviderEventPartialPaymentReceived

	// ProviderEventPaymentReceived happens when a provider receives a payment
	// and resumes processing a deal
	ProviderEventPaymentReceived

	// ProviderEventComplete indicates a retrieval deal was completed for a client
	ProviderEventComplete

	// ProviderEventDataTransferError emits when something go wrong at the data transfer level
	ProviderEventDataTransferError

	// ProviderEventCancelComplete happens when a deal cancellation is transmitted to the provider
	ProviderEventCancelComplete

	// ProviderEventCleanupComplete happens when a deal is finished cleaning up and enters a complete state
	ProviderEventCleanupComplete

	// ProviderEventMultiStoreError occurs when an error happens attempting to operate on the multistore
	ProviderEventMultiStoreError

	// ProviderEventClientCancelled happens when the provider gets a cancel message from the client's data transfer
	ProviderEventClientCancelled
)

// ProviderEvents is a human readable map of provider event name -> event description
var ProviderEvents = map[ProviderEvent]string{
	ProviderEventOpen:                   "ProviderEventOpen",
	ProviderEventDealNotFound:           "ProviderEventDealNotFound",
	ProviderEventDealRejected:           "ProviderEventDealRejected",
	ProviderEventDealAccepted:           "ProviderEventDealAccepted",
	ProviderEventBlockSent:              "ProviderEventBlockSent",
	ProviderEventBlocksCompleted:        "ProviderEventBlocksCompleted",
	ProviderEventPaymentRequested:       "ProviderEventPaymentRequested",
	ProviderEventSaveVoucherFailed:      "ProviderEventSaveVoucherFailed",
	ProviderEventPartialPaymentReceived: "ProviderEventPartialPaymentReceived",
	ProviderEventPaymentReceived:        "ProviderEventPaymentReceived",
	ProviderEventComplete:               "ProviderEventComplete",
	ProviderEventDataTransferError:      "ProviderEventDataTransferError",
	ProviderEventCancelComplete:         "ProviderEventCancelComplete",
	ProviderEventCleanupComplete:        "ProviderEventCleanupComplete",
	ProviderEventMultiStoreError:        "ProviderEventMultiStoreError",
	ProviderEventClientCancelled:        "ProviderEventClientCancelled",
}
