// Package granite defines constants used after the Granite upgrade.
package granite

import "github.com/ava-labs/coreth/utils"

// MinGasPrice is the Flare-family floor for the dynamic ACP-176 base fee
// after the Granite upgrade.
const MinGasPrice = 500 * utils.GWei
