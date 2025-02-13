package main

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	ERC20_ABI                = `[{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}]`
	ERC20_TRANSFER_SIGNATURE = "Transfer(address,address,uint256)"
)

type TokenTransferLog struct {
	Contract    string
	From        string
	To          string
	Amount      *big.Int
	AmountInEth *big.Float
}

func GetTokenDecimals(tokenAddress common.Address) (int64, error) {
	parsedABI, err := abi.JSON(strings.NewReader(ERC20_ABI))
	if err != nil {
		return 0, err
	}

	callData, err := parsedABI.Pack("decimals")
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: callData,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return 0, err
	}

	var decimals uint8
	err = parsedABI.UnpackIntoInterface(&decimals, "decimals", result)
	if err != nil {
		return 0, err
	}

	return int64(decimals), nil
}

func Base10Power(power int64) *big.Float {
	return new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(power), nil))
}

func ParseTokenAmount(tokenContractAddress string, amountInWei *big.Int) *big.Float {
	if decimals, err := GetTokenDecimals(common.HexToAddress(tokenContractAddress)); err == nil {
		return new(big.Float).Quo(new(big.Float).SetInt(amountInWei), Base10Power(decimals))
	}
	return big.NewFloat(0)
}

func ExtractReceiptLogs(receipt *types.Receipt) []TokenTransferLog {
	var logs []TokenTransferLog
	for _, log := range receipt.Logs {

		if len(log.Topics) >= 3 && log.Topics[0] == crypto.Keccak256Hash([]byte(ERC20_TRANSFER_SIGNATURE)) {
			tokenContract := log.Address.Hex()
			from := common.HexToAddress(log.Topics[1].Hex()).Hex()
			to := common.HexToAddress(log.Topics[2].Hex()).Hex()

			// Decode amount
			amount := new(big.Int).SetBytes(log.Data)

			tokenTransfer := TokenTransferLog{
				Contract:    tokenContract,
				From:        from,
				To:          to,
				Amount:      amount,
				AmountInEth: ParseTokenAmount(tokenContract, amount),
			}
			logs = append(logs, tokenTransfer)
		}
	}

	return logs
}
