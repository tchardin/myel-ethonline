// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package channelverify

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ChannelverifyABI is the input ABI used to generate the binding from.
const ChannelverifyABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"getChainlinkToken\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_requestId\",\"type\":\"bytes32\"},{\"name\":\"_settleheight\",\"type\":\"uint256\"}],\"name\":\"fulfillVerifyChannel\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdrawLink\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"succeeded\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_oracle\",\"type\":\"address\"},{\"name\":\"_jobId\",\"type\":\"string\"},{\"name\":\"_channel\",\"type\":\"string\"}],\"name\":\"verifyChannel\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_requestId\",\"type\":\"bytes32\"},{\"name\":\"_payment\",\"type\":\"uint256\"},{\"name\":\"_callbackFunctionId\",\"type\":\"bytes4\"},{\"name\":\"_expiration\",\"type\":\"uint256\"}],\"name\":\"cancelRequest\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_link\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"requestId\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"settleHeight\",\"type\":\"uint256\"}],\"name\":\"RequestVerifyChannelFulfilled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"}],\"name\":\"OwnershipRenounced\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"ChainlinkRequested\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"ChainlinkFulfilled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"id\",\"type\":\"bytes32\"}],\"name\":\"ChainlinkCancelled\",\"type\":\"event\"}]"

// ChannelverifyBin is the compiled bytecode used for deploying new contracts.
var ChannelverifyBin = "0x6080604052600160045534801561001557600080fd5b50604051602080611e378339810180604052810190808051906020019092919050505033600660006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555061009181610097640100000000026401000000009004565b506100db565b80600260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b611d4d806100ea6000396000f300608060405260043610610099576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063165d35e11461009b578063715018a6146100f257806380579774146101095780638da5cb5b146101445780638dc654a21461019b57806393415250146101b2578063d1c72cf9146101dd578063ec65d0f8146102ac578063f2fde38b1461031a575b005b3480156100a757600080fd5b506100b061035d565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156100fe57600080fd5b5061010761036c565b005b34801561011557600080fd5b50610142600480360381019080803560001916906020019092919080359060200190929190505050610471565b005b34801561015057600080fd5b5061015961070b565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156101a757600080fd5b506101b0610731565b005b3480156101be57600080fd5b506101c76109c3565b6040518082815260200191505060405180910390f35b3480156101e957600080fd5b506102aa600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f01602080910402602001604051908101604052809392919081815260200183838082843782019150505050505091929192905050506109c9565b005b3480156102b857600080fd5b5061031860048036038101908080356000191690602001909291908035906020019092919080357bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916906020019092919080359060200190929190505050610b67565b005b34801561032657600080fd5b5061035b600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610bd5565b005b6000610367610c3d565b905090565b600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156103c857600080fd5b600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167ff8df31144d9c2f0f6b59d69b8b98abd5459d07f2742c4df920b25aae33c6482060405160405180910390a26000600660006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b8160056000826000191660001916815260200190815260200160002060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610576576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260288152602001807f536f75726365206d75737420626520746865206f7261636c65206f662074686581526020017f207265717565737400000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b60056000826000191660001916815260200190815260200160002060006101000a81549073ffffffffffffffffffffffffffffffffffffffff021916905580600019167f7cc135e0cebb02c3480ae5d74d377283180a2601f8f644edf7987b009316c63a60405160405180910390a28183600019167fca7a73e635ebcfef6b8987589eda3ef5108d0f1193b125dbfb59402bdca496fb60405160405180910390a360008214156106fd57600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16670de0b6b3a764000060405180602001905060006040518083038185875af19250505015156106f0576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260128152602001807f556e61626c6520746f207472616e73666572000000000000000000000000000081525060200191505060405180910390fd5b6001600781905550610706565b60006007819055505b505050565b600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561078f57600080fd5b610797610c3d565b90508073ffffffffffffffffffffffffffffffffffffffff1663a9059cbb338373ffffffffffffffffffffffffffffffffffffffff166370a08231306040518263ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001915050602060405180830381600087803b15801561085157600080fd5b505af1158015610865573d6000803e3d6000fd5b505050506040513d602081101561087b57600080fd5b81019080805190602001909291905050506040518363ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b15801561091157600080fd5b505af1158015610925573d6000803e3d6000fd5b505050506040513d602081101561093b57600080fd5b810190808051906020019092919050505015156109c0576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260128152602001807f556e61626c6520746f207472616e73666572000000000000000000000000000081525060200191505060405180910390fd5b50565b60075481565b6109d1611c7f565b600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610a2d57600080fd5b610a63610a3984610c67565b3063805797747c010000000000000000000000000000000000000000000000000000000002610c92565b9050610b4a6040805190810160405280600481526020017f626f647900000000000000000000000000000000000000000000000000000000815250610b3a606060405190810160405280603181526020017f7b226d6574686f64223a202246696c65636f696e2e537461746552656164537481526020017f617465222c2022706172616d73223a5b22000000000000000000000000000000815250856040805190810160405280601381526020017f222c206e756c6c5d2c20226964223a2031207d00000000000000000000000000815250610cc3565b83610f559092919063ffffffff16565b610b608482670de0b6b3a7640000600102610f88565b5050505050565b600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610bc357600080fd5b610bcf84848484611314565b50505050565b600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610c3157600080fd5b610c3a816114af565b50565b6000600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60006060829050600081511415610c845760006001029150610c8c565b602083015191505b50919050565b610c9a611c7f565b610ca2611c7f565b610cb9858585846115ab909392919063ffffffff16565b9150509392505050565b6060806060806060806000808a965089955088945084518651885101016040519080825280601f01601f191660200182016040528015610d125781602001602082028038833980820191505090505b50935083925060009150600090505b8651811015610dd4578681815181101515610d3857fe5b9060200101517f010000000000000000000000000000000000000000000000000000000000000090047f0100000000000000000000000000000000000000000000000000000000000000028383806001019450815181101515610d9757fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053508080600101915050610d21565b600090505b8551811015610e8c578581815181101515610df057fe5b9060200101517f010000000000000000000000000000000000000000000000000000000000000090047f0100000000000000000000000000000000000000000000000000000000000000028383806001019450815181101515610e4f57fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053508080600101915050610dd9565b600090505b8451811015610f44578481815181101515610ea857fe5b9060200101517f010000000000000000000000000000000000000000000000000000000000000090047f0100000000000000000000000000000000000000000000000000000000000000028383806001019450815181101515610f0757fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053508080600101915050610e91565b829750505050505050509392505050565b610f6c82846080015161166590919063ffffffff16565b610f8381846080015161166590919063ffffffff16565b505050565b600030600454604051602001808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166c01000000000000000000000000028152601401828152602001925050506040516020818303038152906040526040518082805190602001908083835b6020831015156110245780518252602082019150602081019050602083039250610fff565b6001836020036101000a038019825116818451168082178552505050505050905001915050604051809103902090506004548360600181815250508360056000836000191660001916815260200190815260200160002060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555080600019167fb5e6e01e79f91267dc17b4e6314d5d4d03593d2ceee0fbb452b750bd70ea5af960405160405180910390a2600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16634000aea085846111338761168a565b6040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200183815260200180602001828103825283818151815260200191508051906020019080838360005b838110156111d65780820151818401526020810190506111bb565b50505050905090810190601f1680156112035780820380516001836020036101000a031916815260200191505b50945050505050602060405180830381600087803b15801561122457600080fd5b505af1158015611238573d6000803e3d6000fd5b505050506040513d602081101561124e57600080fd5b810190808051906020019092919050505015156112f9576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260238152602001807f756e61626c6520746f207472616e73666572416e6443616c6c20746f206f726181526020017f636c65000000000000000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b60016004600082825401925050819055508090509392505050565b600060056000866000191660001916815260200190815260200160002060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905060056000866000191660001916815260200190815260200160002060006101000a81549073ffffffffffffffffffffffffffffffffffffffff021916905584600019167fe1fe3afa0f7f761ff0a8b89086790efd5140d2907ebd5b7ff6bfcb5e075fd4c560405160405180910390a28073ffffffffffffffffffffffffffffffffffffffff16636ee4d553868686866040518563ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808560001916600019168152602001848152602001837bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19168152602001828152602001945050505050600060405180830381600087803b15801561149057600080fd5b505af11580156114a4573d6000803e3d6000fd5b505050505050505050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156114eb57600080fd5b8073ffffffffffffffffffffffffffffffffffffffff16600660009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a380600660006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b6115b3611c7f565b6115c385608001516101006118b5565b50838560000190600019169081600019168152505082856020019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250508185604001907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff191690817bffffffffffffffffffffffffffffffffffffffffffffffffffffffff191681525050849050949350505050565b611672826003835161190f565b6116858183611a6d90919063ffffffff16565b505050565b6060600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16634042994690507c01000000000000000000000000000000000000000000000000000000000260008084600001518560200151866040015187606001516001896080015160000151604051602401808973ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200188815260200187600019166000191681526020018673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001857bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19167bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200184815260200183815260200180602001828103825283818151815260200191508051906020019080838360005b838110156118185780820151818401526020810190506117fd565b50505050905090810190601f1680156118455780820380516001836020036101000a031916815260200191505b509950505050505050505050604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050509050919050565b6118bd611ced565b60006020838115156118cb57fe5b061415156118e8576020828115156118df57fe5b06602003820191505b81836020018181525050604051808452600081528281016020016040525082905092915050565b6017811115156119415761193b8160058460ff169060020a0260ff161784611a8f90919063ffffffff16565b50611a68565b60ff811115156119885761196b601860058460ff169060020a021784611a8f90919063ffffffff16565b5061198281600185611aaf9092919063ffffffff16565b50611a67565b61ffff811115156119d0576119b3601960058460ff169060020a021784611a8f90919063ffffffff16565b506119ca81600285611aaf9092919063ffffffff16565b50611a66565b63ffffffff81111515611a1a576119fd601a60058460ff169060020a021784611a8f90919063ffffffff16565b50611a1481600485611aaf9092919063ffffffff16565b50611a65565b67ffffffffffffffff81111515611a6457611a4b601b60058460ff169060020a021784611a8f90919063ffffffff16565b50611a6281600885611aaf9092919063ffffffff16565b505b5b5b5b5b505050565b611a75611ced565b611a8783846000015151848551611ad1565b905092915050565b611a97611ced565b611aa78384600001515184611b8e565b905092915050565b611ab7611ced565b611ac8848560000151518585611bde565b90509392505050565b611ad9611ced565b600080600085518511151515611aee57600080fd5b87602001518588011115611b1957611b18886002611b128b602001518b8a01611c3f565b02611c5b565b5b875180518860208301019450808988011115611b355788870182525b60208801935050505b602085101515611b635781518352602083019250602082019150602085039450611b3e565b6001856020036101000a03905080198251168184511681811785525050879350505050949350505050565b611b96611ced565b836020015183101515611bb557611bb4846002866020015102611c5b565b5b8351805160208583010184815381861415611bd1576001820183525b5050508390509392505050565b611be6611ced565b600085602001518584011115611c0657611c0586600287860102611c5b565b5b6001836101000a0390508551838682010185831982511617815281518588011115611c315784870182525b505085915050949350505050565b600081831115611c5157829050611c55565b8190505b92915050565b606082600001519050611c6e83836118b5565b50611c798382611a6d565b50505050565b60c06040519081016040528060008019168152602001600073ffffffffffffffffffffffffffffffffffffffff16815260200160007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916815260200160008152602001611ce7611d07565b81525090565b604080519081016040528060608152602001600081525090565b6040805190810160405280606081526020016000815250905600a165627a7a723058203bc35ac7980219329dcaf22d4ae5877db50bbff45371904c01964e1711bd3bd70029"

// DeployChannelverify deploys a new Ethereum contract, binding an instance of Channelverify to it.
func DeployChannelverify(auth *bind.TransactOpts, backend bind.ContractBackend, _link common.Address) (common.Address, *types.Transaction, *Channelverify, error) {
	parsed, err := abi.JSON(strings.NewReader(ChannelverifyABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ChannelverifyBin), backend, _link)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Channelverify{ChannelverifyCaller: ChannelverifyCaller{contract: contract}, ChannelverifyTransactor: ChannelverifyTransactor{contract: contract}, ChannelverifyFilterer: ChannelverifyFilterer{contract: contract}}, nil
}

// Channelverify is an auto generated Go binding around an Ethereum contract.
type Channelverify struct {
	ChannelverifyCaller     // Read-only binding to the contract
	ChannelverifyTransactor // Write-only binding to the contract
	ChannelverifyFilterer   // Log filterer for contract events
}

// ChannelverifyCaller is an auto generated read-only Go binding around an Ethereum contract.
type ChannelverifyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChannelverifyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ChannelverifyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChannelverifyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ChannelverifyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ChannelverifySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ChannelverifySession struct {
	Contract     *Channelverify    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ChannelverifyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ChannelverifyCallerSession struct {
	Contract *ChannelverifyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ChannelverifyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ChannelverifyTransactorSession struct {
	Contract     *ChannelverifyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ChannelverifyRaw is an auto generated low-level Go binding around an Ethereum contract.
type ChannelverifyRaw struct {
	Contract *Channelverify // Generic contract binding to access the raw methods on
}

// ChannelverifyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ChannelverifyCallerRaw struct {
	Contract *ChannelverifyCaller // Generic read-only contract binding to access the raw methods on
}

// ChannelverifyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ChannelverifyTransactorRaw struct {
	Contract *ChannelverifyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewChannelverify creates a new instance of Channelverify, bound to a specific deployed contract.
func NewChannelverify(address common.Address, backend bind.ContractBackend) (*Channelverify, error) {
	contract, err := bindChannelverify(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Channelverify{ChannelverifyCaller: ChannelverifyCaller{contract: contract}, ChannelverifyTransactor: ChannelverifyTransactor{contract: contract}, ChannelverifyFilterer: ChannelverifyFilterer{contract: contract}}, nil
}

// NewChannelverifyCaller creates a new read-only instance of Channelverify, bound to a specific deployed contract.
func NewChannelverifyCaller(address common.Address, caller bind.ContractCaller) (*ChannelverifyCaller, error) {
	contract, err := bindChannelverify(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyCaller{contract: contract}, nil
}

// NewChannelverifyTransactor creates a new write-only instance of Channelverify, bound to a specific deployed contract.
func NewChannelverifyTransactor(address common.Address, transactor bind.ContractTransactor) (*ChannelverifyTransactor, error) {
	contract, err := bindChannelverify(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyTransactor{contract: contract}, nil
}

// NewChannelverifyFilterer creates a new log filterer instance of Channelverify, bound to a specific deployed contract.
func NewChannelverifyFilterer(address common.Address, filterer bind.ContractFilterer) (*ChannelverifyFilterer, error) {
	contract, err := bindChannelverify(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyFilterer{contract: contract}, nil
}

// bindChannelverify binds a generic wrapper to an already deployed contract.
func bindChannelverify(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ChannelverifyABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Channelverify *ChannelverifyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Channelverify.Contract.ChannelverifyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Channelverify *ChannelverifyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Channelverify.Contract.ChannelverifyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Channelverify *ChannelverifyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Channelverify.Contract.ChannelverifyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Channelverify *ChannelverifyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Channelverify.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Channelverify *ChannelverifyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Channelverify.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Channelverify *ChannelverifyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Channelverify.Contract.contract.Transact(opts, method, params...)
}

// GetChainlinkToken is a free data retrieval call binding the contract method 0x165d35e1.
//
// Solidity: function getChainlinkToken() view returns(address)
func (_Channelverify *ChannelverifyCaller) GetChainlinkToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Channelverify.contract.Call(opts, &out, "getChainlinkToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetChainlinkToken is a free data retrieval call binding the contract method 0x165d35e1.
//
// Solidity: function getChainlinkToken() view returns(address)
func (_Channelverify *ChannelverifySession) GetChainlinkToken() (common.Address, error) {
	return _Channelverify.Contract.GetChainlinkToken(&_Channelverify.CallOpts)
}

// GetChainlinkToken is a free data retrieval call binding the contract method 0x165d35e1.
//
// Solidity: function getChainlinkToken() view returns(address)
func (_Channelverify *ChannelverifyCallerSession) GetChainlinkToken() (common.Address, error) {
	return _Channelverify.Contract.GetChainlinkToken(&_Channelverify.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Channelverify *ChannelverifyCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Channelverify.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Channelverify *ChannelverifySession) Owner() (common.Address, error) {
	return _Channelverify.Contract.Owner(&_Channelverify.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Channelverify *ChannelverifyCallerSession) Owner() (common.Address, error) {
	return _Channelverify.Contract.Owner(&_Channelverify.CallOpts)
}

// Succeeded is a free data retrieval call binding the contract method 0x93415250.
//
// Solidity: function succeeded() view returns(uint256)
func (_Channelverify *ChannelverifyCaller) Succeeded(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Channelverify.contract.Call(opts, &out, "succeeded")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Succeeded is a free data retrieval call binding the contract method 0x93415250.
//
// Solidity: function succeeded() view returns(uint256)
func (_Channelverify *ChannelverifySession) Succeeded() (*big.Int, error) {
	return _Channelverify.Contract.Succeeded(&_Channelverify.CallOpts)
}

// Succeeded is a free data retrieval call binding the contract method 0x93415250.
//
// Solidity: function succeeded() view returns(uint256)
func (_Channelverify *ChannelverifyCallerSession) Succeeded() (*big.Int, error) {
	return _Channelverify.Contract.Succeeded(&_Channelverify.CallOpts)
}

// CancelRequest is a paid mutator transaction binding the contract method 0xec65d0f8.
//
// Solidity: function cancelRequest(bytes32 _requestId, uint256 _payment, bytes4 _callbackFunctionId, uint256 _expiration) returns()
func (_Channelverify *ChannelverifyTransactor) CancelRequest(opts *bind.TransactOpts, _requestId [32]byte, _payment *big.Int, _callbackFunctionId [4]byte, _expiration *big.Int) (*types.Transaction, error) {
	return _Channelverify.contract.Transact(opts, "cancelRequest", _requestId, _payment, _callbackFunctionId, _expiration)
}

// CancelRequest is a paid mutator transaction binding the contract method 0xec65d0f8.
//
// Solidity: function cancelRequest(bytes32 _requestId, uint256 _payment, bytes4 _callbackFunctionId, uint256 _expiration) returns()
func (_Channelverify *ChannelverifySession) CancelRequest(_requestId [32]byte, _payment *big.Int, _callbackFunctionId [4]byte, _expiration *big.Int) (*types.Transaction, error) {
	return _Channelverify.Contract.CancelRequest(&_Channelverify.TransactOpts, _requestId, _payment, _callbackFunctionId, _expiration)
}

// CancelRequest is a paid mutator transaction binding the contract method 0xec65d0f8.
//
// Solidity: function cancelRequest(bytes32 _requestId, uint256 _payment, bytes4 _callbackFunctionId, uint256 _expiration) returns()
func (_Channelverify *ChannelverifyTransactorSession) CancelRequest(_requestId [32]byte, _payment *big.Int, _callbackFunctionId [4]byte, _expiration *big.Int) (*types.Transaction, error) {
	return _Channelverify.Contract.CancelRequest(&_Channelverify.TransactOpts, _requestId, _payment, _callbackFunctionId, _expiration)
}

// FulfillVerifyChannel is a paid mutator transaction binding the contract method 0x80579774.
//
// Solidity: function fulfillVerifyChannel(bytes32 _requestId, uint256 _settleheight) returns()
func (_Channelverify *ChannelverifyTransactor) FulfillVerifyChannel(opts *bind.TransactOpts, _requestId [32]byte, _settleheight *big.Int) (*types.Transaction, error) {
	return _Channelverify.contract.Transact(opts, "fulfillVerifyChannel", _requestId, _settleheight)
}

// FulfillVerifyChannel is a paid mutator transaction binding the contract method 0x80579774.
//
// Solidity: function fulfillVerifyChannel(bytes32 _requestId, uint256 _settleheight) returns()
func (_Channelverify *ChannelverifySession) FulfillVerifyChannel(_requestId [32]byte, _settleheight *big.Int) (*types.Transaction, error) {
	return _Channelverify.Contract.FulfillVerifyChannel(&_Channelverify.TransactOpts, _requestId, _settleheight)
}

// FulfillVerifyChannel is a paid mutator transaction binding the contract method 0x80579774.
//
// Solidity: function fulfillVerifyChannel(bytes32 _requestId, uint256 _settleheight) returns()
func (_Channelverify *ChannelverifyTransactorSession) FulfillVerifyChannel(_requestId [32]byte, _settleheight *big.Int) (*types.Transaction, error) {
	return _Channelverify.Contract.FulfillVerifyChannel(&_Channelverify.TransactOpts, _requestId, _settleheight)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Channelverify *ChannelverifyTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Channelverify.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Channelverify *ChannelverifySession) RenounceOwnership() (*types.Transaction, error) {
	return _Channelverify.Contract.RenounceOwnership(&_Channelverify.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Channelverify *ChannelverifyTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Channelverify.Contract.RenounceOwnership(&_Channelverify.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Channelverify *ChannelverifyTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _Channelverify.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Channelverify *ChannelverifySession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _Channelverify.Contract.TransferOwnership(&_Channelverify.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Channelverify *ChannelverifyTransactorSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _Channelverify.Contract.TransferOwnership(&_Channelverify.TransactOpts, _newOwner)
}

// VerifyChannel is a paid mutator transaction binding the contract method 0xd1c72cf9.
//
// Solidity: function verifyChannel(address _oracle, string _jobId, string _channel) returns()
func (_Channelverify *ChannelverifyTransactor) VerifyChannel(opts *bind.TransactOpts, _oracle common.Address, _jobId string, _channel string) (*types.Transaction, error) {
	return _Channelverify.contract.Transact(opts, "verifyChannel", _oracle, _jobId, _channel)
}

// VerifyChannel is a paid mutator transaction binding the contract method 0xd1c72cf9.
//
// Solidity: function verifyChannel(address _oracle, string _jobId, string _channel) returns()
func (_Channelverify *ChannelverifySession) VerifyChannel(_oracle common.Address, _jobId string, _channel string) (*types.Transaction, error) {
	return _Channelverify.Contract.VerifyChannel(&_Channelverify.TransactOpts, _oracle, _jobId, _channel)
}

// VerifyChannel is a paid mutator transaction binding the contract method 0xd1c72cf9.
//
// Solidity: function verifyChannel(address _oracle, string _jobId, string _channel) returns()
func (_Channelverify *ChannelverifyTransactorSession) VerifyChannel(_oracle common.Address, _jobId string, _channel string) (*types.Transaction, error) {
	return _Channelverify.Contract.VerifyChannel(&_Channelverify.TransactOpts, _oracle, _jobId, _channel)
}

// WithdrawLink is a paid mutator transaction binding the contract method 0x8dc654a2.
//
// Solidity: function withdrawLink() returns()
func (_Channelverify *ChannelverifyTransactor) WithdrawLink(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Channelverify.contract.Transact(opts, "withdrawLink")
}

// WithdrawLink is a paid mutator transaction binding the contract method 0x8dc654a2.
//
// Solidity: function withdrawLink() returns()
func (_Channelverify *ChannelverifySession) WithdrawLink() (*types.Transaction, error) {
	return _Channelverify.Contract.WithdrawLink(&_Channelverify.TransactOpts)
}

// WithdrawLink is a paid mutator transaction binding the contract method 0x8dc654a2.
//
// Solidity: function withdrawLink() returns()
func (_Channelverify *ChannelverifyTransactorSession) WithdrawLink() (*types.Transaction, error) {
	return _Channelverify.Contract.WithdrawLink(&_Channelverify.TransactOpts)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Channelverify *ChannelverifyTransactor) Fallback(opts *bind.TransactOpts, calldata []byte) (*types.Transaction, error) {
	return _Channelverify.contract.RawTransact(opts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Channelverify *ChannelverifySession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Channelverify.Contract.Fallback(&_Channelverify.TransactOpts, calldata)
}

// Fallback is a paid mutator transaction binding the contract fallback function.
//
// Solidity: fallback() payable returns()
func (_Channelverify *ChannelverifyTransactorSession) Fallback(calldata []byte) (*types.Transaction, error) {
	return _Channelverify.Contract.Fallback(&_Channelverify.TransactOpts, calldata)
}

// ChannelverifyChainlinkCancelledIterator is returned from FilterChainlinkCancelled and is used to iterate over the raw logs and unpacked data for ChainlinkCancelled events raised by the Channelverify contract.
type ChannelverifyChainlinkCancelledIterator struct {
	Event *ChannelverifyChainlinkCancelled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChannelverifyChainlinkCancelledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChannelverifyChainlinkCancelled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChannelverifyChainlinkCancelled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChannelverifyChainlinkCancelledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChannelverifyChainlinkCancelledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChannelverifyChainlinkCancelled represents a ChainlinkCancelled event raised by the Channelverify contract.
type ChannelverifyChainlinkCancelled struct {
	Id  [32]byte
	Raw types.Log // Blockchain specific contextual infos
}

// FilterChainlinkCancelled is a free log retrieval operation binding the contract event 0xe1fe3afa0f7f761ff0a8b89086790efd5140d2907ebd5b7ff6bfcb5e075fd4c5.
//
// Solidity: event ChainlinkCancelled(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) FilterChainlinkCancelled(opts *bind.FilterOpts, id [][32]byte) (*ChannelverifyChainlinkCancelledIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Channelverify.contract.FilterLogs(opts, "ChainlinkCancelled", idRule)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyChainlinkCancelledIterator{contract: _Channelverify.contract, event: "ChainlinkCancelled", logs: logs, sub: sub}, nil
}

// WatchChainlinkCancelled is a free log subscription operation binding the contract event 0xe1fe3afa0f7f761ff0a8b89086790efd5140d2907ebd5b7ff6bfcb5e075fd4c5.
//
// Solidity: event ChainlinkCancelled(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) WatchChainlinkCancelled(opts *bind.WatchOpts, sink chan<- *ChannelverifyChainlinkCancelled, id [][32]byte) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Channelverify.contract.WatchLogs(opts, "ChainlinkCancelled", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChannelverifyChainlinkCancelled)
				if err := _Channelverify.contract.UnpackLog(event, "ChainlinkCancelled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseChainlinkCancelled is a log parse operation binding the contract event 0xe1fe3afa0f7f761ff0a8b89086790efd5140d2907ebd5b7ff6bfcb5e075fd4c5.
//
// Solidity: event ChainlinkCancelled(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) ParseChainlinkCancelled(log types.Log) (*ChannelverifyChainlinkCancelled, error) {
	event := new(ChannelverifyChainlinkCancelled)
	if err := _Channelverify.contract.UnpackLog(event, "ChainlinkCancelled", log); err != nil {
		return nil, err
	}
	return event, nil
}

// ChannelverifyChainlinkFulfilledIterator is returned from FilterChainlinkFulfilled and is used to iterate over the raw logs and unpacked data for ChainlinkFulfilled events raised by the Channelverify contract.
type ChannelverifyChainlinkFulfilledIterator struct {
	Event *ChannelverifyChainlinkFulfilled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChannelverifyChainlinkFulfilledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChannelverifyChainlinkFulfilled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChannelverifyChainlinkFulfilled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChannelverifyChainlinkFulfilledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChannelverifyChainlinkFulfilledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChannelverifyChainlinkFulfilled represents a ChainlinkFulfilled event raised by the Channelverify contract.
type ChannelverifyChainlinkFulfilled struct {
	Id  [32]byte
	Raw types.Log // Blockchain specific contextual infos
}

// FilterChainlinkFulfilled is a free log retrieval operation binding the contract event 0x7cc135e0cebb02c3480ae5d74d377283180a2601f8f644edf7987b009316c63a.
//
// Solidity: event ChainlinkFulfilled(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) FilterChainlinkFulfilled(opts *bind.FilterOpts, id [][32]byte) (*ChannelverifyChainlinkFulfilledIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Channelverify.contract.FilterLogs(opts, "ChainlinkFulfilled", idRule)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyChainlinkFulfilledIterator{contract: _Channelverify.contract, event: "ChainlinkFulfilled", logs: logs, sub: sub}, nil
}

// WatchChainlinkFulfilled is a free log subscription operation binding the contract event 0x7cc135e0cebb02c3480ae5d74d377283180a2601f8f644edf7987b009316c63a.
//
// Solidity: event ChainlinkFulfilled(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) WatchChainlinkFulfilled(opts *bind.WatchOpts, sink chan<- *ChannelverifyChainlinkFulfilled, id [][32]byte) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Channelverify.contract.WatchLogs(opts, "ChainlinkFulfilled", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChannelverifyChainlinkFulfilled)
				if err := _Channelverify.contract.UnpackLog(event, "ChainlinkFulfilled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseChainlinkFulfilled is a log parse operation binding the contract event 0x7cc135e0cebb02c3480ae5d74d377283180a2601f8f644edf7987b009316c63a.
//
// Solidity: event ChainlinkFulfilled(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) ParseChainlinkFulfilled(log types.Log) (*ChannelverifyChainlinkFulfilled, error) {
	event := new(ChannelverifyChainlinkFulfilled)
	if err := _Channelverify.contract.UnpackLog(event, "ChainlinkFulfilled", log); err != nil {
		return nil, err
	}
	return event, nil
}

// ChannelverifyChainlinkRequestedIterator is returned from FilterChainlinkRequested and is used to iterate over the raw logs and unpacked data for ChainlinkRequested events raised by the Channelverify contract.
type ChannelverifyChainlinkRequestedIterator struct {
	Event *ChannelverifyChainlinkRequested // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChannelverifyChainlinkRequestedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChannelverifyChainlinkRequested)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChannelverifyChainlinkRequested)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChannelverifyChainlinkRequestedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChannelverifyChainlinkRequestedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChannelverifyChainlinkRequested represents a ChainlinkRequested event raised by the Channelverify contract.
type ChannelverifyChainlinkRequested struct {
	Id  [32]byte
	Raw types.Log // Blockchain specific contextual infos
}

// FilterChainlinkRequested is a free log retrieval operation binding the contract event 0xb5e6e01e79f91267dc17b4e6314d5d4d03593d2ceee0fbb452b750bd70ea5af9.
//
// Solidity: event ChainlinkRequested(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) FilterChainlinkRequested(opts *bind.FilterOpts, id [][32]byte) (*ChannelverifyChainlinkRequestedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Channelverify.contract.FilterLogs(opts, "ChainlinkRequested", idRule)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyChainlinkRequestedIterator{contract: _Channelverify.contract, event: "ChainlinkRequested", logs: logs, sub: sub}, nil
}

// WatchChainlinkRequested is a free log subscription operation binding the contract event 0xb5e6e01e79f91267dc17b4e6314d5d4d03593d2ceee0fbb452b750bd70ea5af9.
//
// Solidity: event ChainlinkRequested(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) WatchChainlinkRequested(opts *bind.WatchOpts, sink chan<- *ChannelverifyChainlinkRequested, id [][32]byte) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _Channelverify.contract.WatchLogs(opts, "ChainlinkRequested", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChannelverifyChainlinkRequested)
				if err := _Channelverify.contract.UnpackLog(event, "ChainlinkRequested", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseChainlinkRequested is a log parse operation binding the contract event 0xb5e6e01e79f91267dc17b4e6314d5d4d03593d2ceee0fbb452b750bd70ea5af9.
//
// Solidity: event ChainlinkRequested(bytes32 indexed id)
func (_Channelverify *ChannelverifyFilterer) ParseChainlinkRequested(log types.Log) (*ChannelverifyChainlinkRequested, error) {
	event := new(ChannelverifyChainlinkRequested)
	if err := _Channelverify.contract.UnpackLog(event, "ChainlinkRequested", log); err != nil {
		return nil, err
	}
	return event, nil
}

// ChannelverifyOwnershipRenouncedIterator is returned from FilterOwnershipRenounced and is used to iterate over the raw logs and unpacked data for OwnershipRenounced events raised by the Channelverify contract.
type ChannelverifyOwnershipRenouncedIterator struct {
	Event *ChannelverifyOwnershipRenounced // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChannelverifyOwnershipRenouncedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChannelverifyOwnershipRenounced)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChannelverifyOwnershipRenounced)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChannelverifyOwnershipRenouncedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChannelverifyOwnershipRenouncedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChannelverifyOwnershipRenounced represents a OwnershipRenounced event raised by the Channelverify contract.
type ChannelverifyOwnershipRenounced struct {
	PreviousOwner common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipRenounced is a free log retrieval operation binding the contract event 0xf8df31144d9c2f0f6b59d69b8b98abd5459d07f2742c4df920b25aae33c64820.
//
// Solidity: event OwnershipRenounced(address indexed previousOwner)
func (_Channelverify *ChannelverifyFilterer) FilterOwnershipRenounced(opts *bind.FilterOpts, previousOwner []common.Address) (*ChannelverifyOwnershipRenouncedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}

	logs, sub, err := _Channelverify.contract.FilterLogs(opts, "OwnershipRenounced", previousOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyOwnershipRenouncedIterator{contract: _Channelverify.contract, event: "OwnershipRenounced", logs: logs, sub: sub}, nil
}

// WatchOwnershipRenounced is a free log subscription operation binding the contract event 0xf8df31144d9c2f0f6b59d69b8b98abd5459d07f2742c4df920b25aae33c64820.
//
// Solidity: event OwnershipRenounced(address indexed previousOwner)
func (_Channelverify *ChannelverifyFilterer) WatchOwnershipRenounced(opts *bind.WatchOpts, sink chan<- *ChannelverifyOwnershipRenounced, previousOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}

	logs, sub, err := _Channelverify.contract.WatchLogs(opts, "OwnershipRenounced", previousOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChannelverifyOwnershipRenounced)
				if err := _Channelverify.contract.UnpackLog(event, "OwnershipRenounced", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipRenounced is a log parse operation binding the contract event 0xf8df31144d9c2f0f6b59d69b8b98abd5459d07f2742c4df920b25aae33c64820.
//
// Solidity: event OwnershipRenounced(address indexed previousOwner)
func (_Channelverify *ChannelverifyFilterer) ParseOwnershipRenounced(log types.Log) (*ChannelverifyOwnershipRenounced, error) {
	event := new(ChannelverifyOwnershipRenounced)
	if err := _Channelverify.contract.UnpackLog(event, "OwnershipRenounced", log); err != nil {
		return nil, err
	}
	return event, nil
}

// ChannelverifyOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Channelverify contract.
type ChannelverifyOwnershipTransferredIterator struct {
	Event *ChannelverifyOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChannelverifyOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChannelverifyOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChannelverifyOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChannelverifyOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChannelverifyOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChannelverifyOwnershipTransferred represents a OwnershipTransferred event raised by the Channelverify contract.
type ChannelverifyOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Channelverify *ChannelverifyFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ChannelverifyOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Channelverify.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyOwnershipTransferredIterator{contract: _Channelverify.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Channelverify *ChannelverifyFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ChannelverifyOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Channelverify.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChannelverifyOwnershipTransferred)
				if err := _Channelverify.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Channelverify *ChannelverifyFilterer) ParseOwnershipTransferred(log types.Log) (*ChannelverifyOwnershipTransferred, error) {
	event := new(ChannelverifyOwnershipTransferred)
	if err := _Channelverify.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	return event, nil
}

// ChannelverifyRequestVerifyChannelFulfilledIterator is returned from FilterRequestVerifyChannelFulfilled and is used to iterate over the raw logs and unpacked data for RequestVerifyChannelFulfilled events raised by the Channelverify contract.
type ChannelverifyRequestVerifyChannelFulfilledIterator struct {
	Event *ChannelverifyRequestVerifyChannelFulfilled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ChannelverifyRequestVerifyChannelFulfilledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ChannelverifyRequestVerifyChannelFulfilled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ChannelverifyRequestVerifyChannelFulfilled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ChannelverifyRequestVerifyChannelFulfilledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ChannelverifyRequestVerifyChannelFulfilledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ChannelverifyRequestVerifyChannelFulfilled represents a RequestVerifyChannelFulfilled event raised by the Channelverify contract.
type ChannelverifyRequestVerifyChannelFulfilled struct {
	RequestId    [32]byte
	SettleHeight *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterRequestVerifyChannelFulfilled is a free log retrieval operation binding the contract event 0xca7a73e635ebcfef6b8987589eda3ef5108d0f1193b125dbfb59402bdca496fb.
//
// Solidity: event RequestVerifyChannelFulfilled(bytes32 indexed requestId, uint256 indexed settleHeight)
func (_Channelverify *ChannelverifyFilterer) FilterRequestVerifyChannelFulfilled(opts *bind.FilterOpts, requestId [][32]byte, settleHeight []*big.Int) (*ChannelverifyRequestVerifyChannelFulfilledIterator, error) {

	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}
	var settleHeightRule []interface{}
	for _, settleHeightItem := range settleHeight {
		settleHeightRule = append(settleHeightRule, settleHeightItem)
	}

	logs, sub, err := _Channelverify.contract.FilterLogs(opts, "RequestVerifyChannelFulfilled", requestIdRule, settleHeightRule)
	if err != nil {
		return nil, err
	}
	return &ChannelverifyRequestVerifyChannelFulfilledIterator{contract: _Channelverify.contract, event: "RequestVerifyChannelFulfilled", logs: logs, sub: sub}, nil
}

// WatchRequestVerifyChannelFulfilled is a free log subscription operation binding the contract event 0xca7a73e635ebcfef6b8987589eda3ef5108d0f1193b125dbfb59402bdca496fb.
//
// Solidity: event RequestVerifyChannelFulfilled(bytes32 indexed requestId, uint256 indexed settleHeight)
func (_Channelverify *ChannelverifyFilterer) WatchRequestVerifyChannelFulfilled(opts *bind.WatchOpts, sink chan<- *ChannelverifyRequestVerifyChannelFulfilled, requestId [][32]byte, settleHeight []*big.Int) (event.Subscription, error) {

	var requestIdRule []interface{}
	for _, requestIdItem := range requestId {
		requestIdRule = append(requestIdRule, requestIdItem)
	}
	var settleHeightRule []interface{}
	for _, settleHeightItem := range settleHeight {
		settleHeightRule = append(settleHeightRule, settleHeightItem)
	}

	logs, sub, err := _Channelverify.contract.WatchLogs(opts, "RequestVerifyChannelFulfilled", requestIdRule, settleHeightRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ChannelverifyRequestVerifyChannelFulfilled)
				if err := _Channelverify.contract.UnpackLog(event, "RequestVerifyChannelFulfilled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRequestVerifyChannelFulfilled is a log parse operation binding the contract event 0xca7a73e635ebcfef6b8987589eda3ef5108d0f1193b125dbfb59402bdca496fb.
//
// Solidity: event RequestVerifyChannelFulfilled(bytes32 indexed requestId, uint256 indexed settleHeight)
func (_Channelverify *ChannelverifyFilterer) ParseRequestVerifyChannelFulfilled(log types.Log) (*ChannelverifyRequestVerifyChannelFulfilled, error) {
	event := new(ChannelverifyRequestVerifyChannelFulfilled)
	if err := _Channelverify.contract.UnpackLog(event, "RequestVerifyChannelFulfilled", log); err != nil {
		return nil, err
	}
	return event, nil
}
