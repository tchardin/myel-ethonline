# Retrieval Market Library

This library uses Filecoin and IPFS golang libraries to bootstrap a retrieval market.
In this market, agents can have the role of Client, Provider and eventually both.

## State Machine

The retrieval market is structured around deals between a client and provider.
To keep track of a deal's lifecycle we use Filecoin's finite state machine framework.
Both providers and clients have their own state machines where the status of a deal and its 
metadata is mutated based on external and internal events.

### Client Events

- `ClientEventOpen` indicates a deal was initiated

`Status = DealStatusNew`

├── `Client.Retrieve(...)` 
└── `Client.go`

- `ClientEventDealProposed` means a deal was successfully sent to a provider

`Status = DealStatusWaitForAcceptance`

├── `ProposeDeal(...)`
└── on `DealStatusNew`

- `ClientEventDealAccepted` means a provider accepted a deal

`Status = DealStatusAccepted`

├── `clientEventForResponse(...)`
├── `ClientDataTransferSubscriber(...)`
└── `dtutils.go`

- `ClientEventPaymentChannelCreateInitiated` means we are waiting for our payment channel to appear on chain

`Status = DealStatusPaymentChannelCreating`

├── `SetupPaymentChannelStart(...)` `if paych == address.Undef`
├── on `DealStatusAccepted`
└── 
- `ClientEventPaymentChannelReady` means the newly created payment channel is ready for the deal to resume

`Status = DealStatusPaymentChannelAllocatingLane`

├── `WaitPaymentChannelReady(...)`
├── on `DealStatusPaymentChannelCreating`
└──

- `ClientEventPaymentChannelAddingFunds` means we are waiting for funds to be added to a payment channel

`Status = DealStatusPaymentChannelAllocatingLane`

├── `SetupPaymentChannelStart(...)` `if paych`
├──
└──

- `ClientEventLaneAllocated` is called when a lane is allocated

`Status = DealStatusOngoing`

├── `AllocateLane(...)`
├── on `DealStatusPaymentChannelAllocatingLane`
└──

- `ClientEventPaymentRequested` indicates the provider requested a payment

`Status = DealStatusFundsNeeded`

├── `clientEventForResponse(...)`
├── `ClientDataTransferSubscriber(...)` `case NewVoucherResult`
└──

- `ClientEventBlocksReceived` indicates the provider has sent blocks

`Status = // No change`

├── `ClientDataTransferSubscriber(...)` `case Progress`
├──
└──

- `ClientEventSendFunds` emits when we reach the threshold to send the next payment

`Status = DealStatusSendFunds`

├── `ProcessPaymentRequested(...)`
├── on `DealStatusFundsNeeded` or `DealStatusFundsNeededLastPayment`
└──

- `ClientEventPaymentSent` indicates a payment was sent to the provider

`Status = DealStatusOngoing`

├── `SendFunds(...)`
└── on `DealStatusSendFunds` or `DealStatusSendFundsLastPayment`

- `ClientEventComplete` indicates a deal has completed

`Status = DealStatusComplete`

├── `clientEventForResponse(...)`
└── `ClientDataTransferSubscriber(...)` `case NewVoucherResult`


### Provider Events

// TODO

├──
└──
├──
└──

├──
└──






















