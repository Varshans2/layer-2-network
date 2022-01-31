package staking

import (
	"fmt"
	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"math"
	"math/big"

	"github.com/0xPolygon/polygon-edge/helper/keccak"
	"github.com/0xPolygon/polygon-edge/types"
)

var (
	StakingSCAddress = types.StringToAddress("1001")
)

// PadLeftOrTrim left-pads the passed in byte array to the specified size,
// or trims the array if it exceeds the passed in size
func PadLeftOrTrim(bb []byte, size int) []byte {
	l := len(bb)
	if l == size {
		return bb
	}

	if l > size {
		return bb[l-size:]
	}

	tmp := make([]byte, size)
	copy(tmp[size-l:], bb)

	return tmp
}

// getAddressMapping returns the key for the SC storage mapping (address => something)
//
// More information:
// https://docs.soliditylang.org/en/latest/internals/layout_in_storage.html
func getAddressMapping(address types.Address, slot int64) []byte {
	bigSlot := big.NewInt(slot)

	finalSlice := append(
		PadLeftOrTrim(address.Bytes(), 32),
		PadLeftOrTrim(bigSlot.Bytes(), 32)...,
	)
	keccakValue := keccak.Keccak256(nil, finalSlice)

	return keccakValue
}

// getIndexWithOffset is a helper method for adding an offset to the already found keccak hash
func getIndexWithOffset(keccakHash []byte, offset int64) []byte {
	bigOffset := big.NewInt(offset)
	bigKeccak := big.NewInt(0).SetBytes(keccakHash)

	bigKeccak.Add(bigKeccak, bigOffset)

	return bigKeccak.Bytes()
}

// getStorageIndexes is a helper function for getting the correct indexes
// of the storage slots which need to be modified during bootstrap.
//
// It is SC dependant, and based on the SC located at:
// https://github.com/0xPolygon/staking-contracts/
func getStorageIndexes(address types.Address, index int64) *StorageIndexes {
	storageIndexes := StorageIndexes{}

	// Get the indexes for the mappings
	// The index for the mapping is retrieved with:
	// keccak(address . slot)
	// . stands for concatenation (basically appending the bytes)
	storageIndexes.AddressToIsValidatorIndex = getAddressMapping(address, addressToIsValidatorSlot)
	storageIndexes.AddressToStakedAmountIndex = getAddressMapping(address, addressToStakedAmountSlot)
	storageIndexes.AddressToValidatorIndexIndex = getAddressMapping(address, addressToValidatorIndexSlot)

	// Get the indexes for _validators, _stakedAmount, _maxNumValidators
	// Index for regular types is calculated as just the regular slot
	storageIndexes.StakedAmountIndex = big.NewInt(stakedAmountSlot).Bytes()
	storageIndexes.MaximumNumValidatorsIndex = big.NewInt(maximumNumValidator).Bytes()

	// Index for array types is calculated as keccak(slot) + index
	// The slot for the dynamic arrays that's put in the keccak needs to be in hex form (padded 64 chars)
	storageIndexes.ValidatorsIndex = getIndexWithOffset(
		keccak.Keccak256(nil, PadLeftOrTrim(big.NewInt(validatorsSlot).Bytes(), 32)),
		index,
	)

	// For any dynamic array in Solidity, the size of the actual array should be
	// located on slot x
	storageIndexes.ValidatorsArraySizeIndex = []byte{byte(validatorsSlot)}

	return &storageIndexes
}

// StorageIndexes is a wrapper for different storage indexes that
// need to be modified
type StorageIndexes struct {
	ValidatorsIndex              []byte // []address
	ValidatorsArraySizeIndex     []byte // []address size
	AddressToIsValidatorIndex    []byte // mapping(address => bool)
	AddressToStakedAmountIndex   []byte // mapping(address => uint256)
	AddressToValidatorIndexIndex []byte // mapping(address => uint256)
	StakedAmountIndex            []byte // uint256
	MaximumNumValidatorsIndex    []byte // uint256
}

// Slot definitions for SC storage
var (
	validatorsSlot              = int64(0) // Slot 0
	addressToIsValidatorSlot    = int64(1) // Slot 1
	addressToStakedAmountSlot   = int64(2) // Slot 2
	addressToValidatorIndexSlot = int64(3) // Slot 3
	stakedAmountSlot            = int64(4) // Slot 4
	maximumNumValidator         = int64(5) // Slot 5
)

const (
	DefaultStakedBalance = "0x8AC7230489E80000" // 10 ETH
	//nolint: lll
	StakingSCBytecode = "0x60806040526004361061008a5760003560e01c806350d68ed81161005957806350d68ed8146101865780636a768705146101b1578063ca1e7819146101dc578063f90ecacc14610207578063facd743b14610244576100f8565b80632367f6b5146100fd5780632def66201461013a578063373d6132146101515780633a4b66f11461017c576100f8565b366100f8576100ae3373ffffffffffffffffffffffffffffffffffffffff16610281565b156100ee576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016100e590610fc9565b60405180910390fd5b6100f6610294565b005b600080fd5b34801561010957600080fd5b50610124600480360381019061011f9190610d18565b610474565b6040516101319190611004565b60405180910390f35b34801561014657600080fd5b5061014f6104bd565b005b34801561015d57600080fd5b506101666105a8565b6040516101739190611004565b60405180910390f35b6101846105b2565b005b34801561019257600080fd5b5061019b61061b565b6040516101a89190610fe9565b60405180910390f35b3480156101bd57600080fd5b506101c6610627565b6040516101d3919061101f565b60405180910390f35b3480156101e857600080fd5b506101f161062c565b6040516101fe9190610f0c565b60405180910390f35b34801561021357600080fd5b5061022e60048036038101906102299190610d45565b6106ba565b60405161023b9190610ef1565b60405180910390f35b34801561025057600080fd5b5061026b60048036038101906102669190610d18565b6106f9565b6040516102789190610f2e565b60405180910390f35b600080823b905060008111915050919050565b600560009054906101000a900463ffffffff1663ffffffff16600080549050106102f3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016102ea90610f89565b60405180910390fd5b34600460008282546103059190611084565b9250508190555034600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825461035b9190611084565b92505081905550600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff161580156104155750670de0b6b3a76400006fffffffffffffffffffffffffffffffff16600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410155b15610424576104233361074f565b5b3373ffffffffffffffffffffffffffffffffffffffff167f9e71bc8eea02a63969f509818f2dafb9254532904319f9dbda79b67bd34a5f3d3460405161046a9190611004565b60405180910390a2565b6000600260008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b6104dc3373ffffffffffffffffffffffffffffffffffffffff16610281565b1561051c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161051390610fc9565b60405180910390fd5b6000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020541161059e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161059590610f69565b60405180910390fd5b6105a6610855565b565b6000600454905090565b6105d13373ffffffffffffffffffffffffffffffffffffffff16610281565b15610611576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161060890610fc9565b60405180910390fd5b610619610294565b565b670de0b6b3a764000081565b600481565b606060008054806020026020016040519081016040528092919081815260200182805480156106b057602002820191906000526020600020905b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610666575b5050505050905090565b600081815481106106ca57600080fd5b906000526020600020016000915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000600160008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff169050919050565b60018060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff021916908315150217905550600080549050600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000819080600181540180825580915050600190039060005260206000200160009091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b600463ffffffff16600080549050116108a3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161089a90610f49565b60405180910390fd5b6000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060009054906101000a900460ff16156109435761094233610a39565b5b6000600260003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550806004600082825461099a91906110da565b925050819055503373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f193505050501580156109e7573d6000803e3d6000fd5b503373ffffffffffffffffffffffffffffffffffffffff167f0f5bb82176feb1b5e747e28471aa92156a04d9f3ab9f45f28e2d704232b93f7582604051610a2e9190611004565b60405180910390a250565b600080549050600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410610abf576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610ab690610fa9565b60405180910390fd5b6000600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905060006001600080549050610b1791906110da565b9050808214610c05576000808281548110610b3557610b346111e0565b5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690508060008481548110610b7757610b766111e0565b5b9060005260206000200160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555082600360008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550505b6000600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060006101000a81548160ff0219169083151502179055506000600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506000805480610cb457610cb36111b1565b5b6001900381819060005260206000200160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690559055505050565b600081359050610cfd81611353565b92915050565b600081359050610d128161136a565b92915050565b600060208284031215610d2e57610d2d61120f565b5b6000610d3c84828501610cee565b91505092915050565b600060208284031215610d5b57610d5a61120f565b5b6000610d6984828501610d03565b91505092915050565b6000610d7e8383610d8a565b60208301905092915050565b610d938161110e565b82525050565b610da28161110e565b82525050565b6000610db38261104a565b610dbd8185611062565b9350610dc88361103a565b8060005b83811015610df9578151610de08882610d72565b9750610deb83611055565b925050600181019050610dcc565b5085935050505092915050565b610e0f81611120565b82525050565b6000610e22604483611073565b9150610e2d82611214565b606082019050919050565b6000610e45601d83611073565b9150610e5082611289565b602082019050919050565b6000610e68602783611073565b9150610e73826112b2565b604082019050919050565b6000610e8b601283611073565b9150610e9682611301565b602082019050919050565b6000610eae601a83611073565b9150610eb98261132a565b602082019050919050565b610ecd8161112c565b82525050565b610edc81611168565b82525050565b610eeb81611172565b82525050565b6000602082019050610f066000830184610d99565b92915050565b60006020820190508181036000830152610f268184610da8565b905092915050565b6000602082019050610f436000830184610e06565b92915050565b60006020820190508181036000830152610f6281610e15565b9050919050565b60006020820190508181036000830152610f8281610e38565b9050919050565b60006020820190508181036000830152610fa281610e5b565b9050919050565b60006020820190508181036000830152610fc281610e7e565b9050919050565b60006020820190508181036000830152610fe281610ea1565b9050919050565b6000602082019050610ffe6000830184610ec4565b92915050565b60006020820190506110196000830184610ed3565b92915050565b60006020820190506110346000830184610ee2565b92915050565b6000819050602082019050919050565b600081519050919050565b6000602082019050919050565b600082825260208201905092915050565b600082825260208201905092915050565b600061108f82611168565b915061109a83611168565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156110cf576110ce611182565b5b828201905092915050565b60006110e582611168565b91506110f083611168565b92508282101561110357611102611182565b5b828203905092915050565b600061111982611148565b9050919050565b60008115159050919050565b60006fffffffffffffffffffffffffffffffff82169050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b600063ffffffff82169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600080fd5b7f4e756d626572206f662076616c696461746f72732063616e2774206265206c6560008201527f7373207468616e204d696e696d756d52657175697265644e756d56616c69646160208201527f746f727300000000000000000000000000000000000000000000000000000000604082015250565b7f4f6e6c79207374616b65722063616e2063616c6c2066756e6374696f6e000000600082015250565b7f56616c696461746f72207365742068617320726561636865642066756c6c206360008201527f6170616369747900000000000000000000000000000000000000000000000000602082015250565b7f696e646578206f7574206f662072616e67650000000000000000000000000000600082015250565b7f4f6e6c7920454f412063616e2063616c6c2066756e6374696f6e000000000000600082015250565b61135c8161110e565b811461136757600080fd5b50565b61137381611168565b811461137e57600080fd5b5056fea2646970667358221220c07688cc0da884ef89fcd94b708a752e77db137a8f550de2e276a2355fbf809264736f6c63430008070033"
)

// PredeployStakingSC is a helper method for setting up the staking smart contract account,
// using the passed in validators as prestaked validators
func PredeployStakingSC(
	validators []types.Address,
	maxValidatorCount uint,
) (*chain.GenesisAccount, error) {
	// Set the code for the staking smart contract
	// Code retrieved from https://github.com/0xPolygon/staking-contracts
	scHex, _ := hex.DecodeHex(StakingSCBytecode)
	stakingAccount := &chain.GenesisAccount{
		Code: scHex,
	}

	// Parse the default staked balance value into *big.Int
	val := DefaultStakedBalance
	bigDefaultStakedBalance, err := types.ParseUint256orHex(&val)

	if err != nil {
		return nil, fmt.Errorf("unable to generate DefaultStatkedBalance, %w", err)
	}

	// Generate the empty account storage map
	storageMap := make(map[types.Hash]types.Hash)
	bigTrueValue := big.NewInt(1)
	stakedAmount := big.NewInt(0)

	if maxValidatorCount > math.MaxUint32 {
		maxValidatorCount = math.MaxUint32
	}
	maxNumValidators := big.NewInt(int64(maxValidatorCount))

	for indx, validator := range validators {
		// Update the total staked amount
		stakedAmount.Add(stakedAmount, bigDefaultStakedBalance)

		// Get the storage indexes
		storageIndexes := getStorageIndexes(validator, int64(indx))

		// Set the value for the validators array
		storageMap[types.BytesToHash(storageIndexes.ValidatorsIndex)] =
			types.BytesToHash(
				validator.Bytes(),
			)

		// Set the value for the address -> validator array index mapping
		storageMap[types.BytesToHash(storageIndexes.AddressToIsValidatorIndex)] =
			types.BytesToHash(bigTrueValue.Bytes())

		// Set the value for the address -> staked amount mapping
		storageMap[types.BytesToHash(storageIndexes.AddressToStakedAmountIndex)] =
			types.StringToHash(hex.EncodeBig(bigDefaultStakedBalance))

		// Set the value for the address -> validator index mapping
		storageMap[types.BytesToHash(storageIndexes.AddressToValidatorIndexIndex)] =
			types.StringToHash(hex.EncodeUint64(uint64(indx)))

		// Set the value for the total staked amount
		storageMap[types.BytesToHash(storageIndexes.StakedAmountIndex)] =
			types.BytesToHash(stakedAmount.Bytes())

		// Set the value for the size of the validators array
		storageMap[types.BytesToHash(storageIndexes.ValidatorsArraySizeIndex)] =
			types.StringToHash(hex.EncodeUint64(uint64(indx + 1)))

		// Set the value for the maximum number of validators
		storageMap[types.BytesToHash(storageIndexes.MaximumNumValidatorsIndex)] =
			types.BytesToHash(maxNumValidators.Bytes())
	}

	// Save the storage map
	stakingAccount.Storage = storageMap

	// Set the Staking SC balance to numValidators * defaultStakedBalance
	stakingAccount.Balance = stakedAmount

	return stakingAccount, nil
}
