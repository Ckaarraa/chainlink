package pipeline

import (
	"bytes"
	"regexp"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
)

var (
	abiRegex        = regexp.MustCompile(`\A\s*([a-zA-Z0-9_]+)\s*\(\s*([a-zA-Z0-9\[\]_\s,]+\s*)?\)`)
	indexedKeyword  = []byte("indexed")
	calldataKeyword = []byte("calldata")
	memoryKeyword   = []byte("memory")
	storageKeyword  = []byte("storage")
	spaceDelim      = []byte(" ")
	commaDelim      = []byte(",")
)

func parseABIString(theABI string, isLog bool) (name string, args abi.Arguments, indexedArgs abi.Arguments, _ error) {
	matches := abiRegex.FindAllSubmatch([]byte(theABI), -1)
	if len(matches) != 1 || len(matches[0]) != 3 {
		return "", nil, nil, errors.Errorf("bad ABI specification: %v", theABI)
	}
	name = string(matches[0][1])
	var argStrs [][]byte
	if len(bytes.TrimSpace(matches[0][2])) > 0 {
		argStrs = bytes.Split(matches[0][2], commaDelim)
	}

	for _, argStr := range argStrs {
		argStr = bytes.Replace(argStr, calldataKeyword, nil, -1) // Strip `calldata` modifiers
		argStr = bytes.Replace(argStr, memoryKeyword, nil, -1)   // Strip `memory` modifiers
		argStr = bytes.Replace(argStr, storageKeyword, nil, -1)  // Strip `storage` modifiers
		argStr = bytes.TrimSpace(argStr)
		parts := bytes.Split(argStr, spaceDelim)
		var (
			argParts [][]byte
			typeStr  []byte
			argName  []byte
			indexed  bool
		)
		for i := range parts {
			parts[i] = bytes.TrimSpace(parts[i])
			if len(parts[i]) > 0 {
				argParts = append(argParts, parts[i])
			}
		}
		switch len(argParts) {
		case 0:
			return "", nil, nil, errors.Errorf("bad ABI specification, empty argument: %v", theABI)

		case 1:
			return "", nil, nil, errors.Errorf("bad ABI specification, missing argument name: %v", theABI)

		case 2:
			if isLog && bytes.Equal(argParts[1], indexedKeyword) {
				return "", nil, nil, errors.Errorf("bad ABI specification, missing argument name: %v", theABI)
			}
			typeStr = argParts[0]
			argName = argParts[1]

		case 3:
			if !isLog {
				return "", nil, nil, errors.Errorf("bad ABI specification, too many components in argument: %v", theABI)
			} else if bytes.Equal(argParts[0], indexedKeyword) || bytes.Equal(argParts[2], indexedKeyword) {
				return "", nil, nil, errors.Errorf("bad ABI specification, 'indexed' keyword must appear between argument type and name: %v", theABI)
			} else if !bytes.Equal(argParts[1], indexedKeyword) {
				return "", nil, nil, errors.Errorf("bad ABI specification, unknown keyword '%v' between argument type and name: %v", string(argParts[1]), theABI)
			}
			typeStr = argParts[0]
			argName = argParts[2]
			indexed = true

		default:
			return "", nil, nil, errors.Errorf("bad ABI specification, too many components in argument: %v", theABI)
		}
		typ, err := abi.NewType(string(typeStr), "", nil)
		if err != nil {
			return "", nil, nil, errors.Errorf("bad ABI specification: %v", err.Error())
		}
		args = append(args, abi.Argument{Type: typ, Name: string(argName), Indexed: indexed})
		if indexed {
			indexedArgs = append(indexedArgs, abi.Argument{Type: typ, Name: string(argName), Indexed: indexed})
		}
	}
	return name, args, indexedArgs, nil
}
